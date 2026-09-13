package backup

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type deletionFiles struct {
	catalogFiles
	fail string
}

func (d *deletionFiles) Write(_ context.Context, name string, in io.Reader) error {
	raw, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(string(d.catalogFiles), name), raw, 0600)
}
func (d *deletionFiles) Remove(_ context.Context, name string) error {
	if name == d.fail {
		return errors.New("injected SMB failure")
	}
	return os.Remove(filepath.Join(string(d.catalogFiles), name))
}

func TestDeletionResumesPartialSetWithoutTouchingOtherFiles(t *testing.T) {
	dir := t.TempDir()
	remote := &deletionFiles{catalogFiles: catalogFiles(dir)}
	s := &Service{Directory: t.TempDir()}
	copy := RemoteCopy{ID: uuid.NewString(), Format: 3, Size: 3, SHA256: strings.Repeat("a", 64), CreatedAt: time.Now().UTC()}
	raw, err := json.Marshal(copy)
	require.NoError(t, err)
	for _, name := range []string{copy.ID + ".delete.json", copy.ID + ".manifest.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), raw, 0600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, copy.ID+".tar.gz"), []byte("abc"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated"), []byte("keep"), 0600))
	remote.fail = copy.ID + ".manifest.json"
	require.Error(t, s.deleteRemote(context.Background(), remote, copy, 1))
	require.NoFileExists(t, filepath.Join(dir, copy.ID+".tar.gz"))
	copies, err := Catalog(context.Background(), remote)
	require.NoError(t, err)
	require.Len(t, copies, 1)
	require.Equal(t, "deleting", copies[0].Verification)
	remote.fail = ""
	restarted := &Service{Directory: s.Directory}
	require.NoError(t, restarted.deleteRemote(context.Background(), remote, copy, 1))
	require.FileExists(t, filepath.Join(dir, "unrelated"))
	copies, err = Catalog(context.Background(), remote)
	require.NoError(t, err)
	require.Empty(t, copies)
}

func TestDeletionPreservesLastCopyEvenWithZeroMinimum(t *testing.T) {
	dir := t.TempDir()
	remote := &deletionFiles{catalogFiles: catalogFiles(dir)}
	s := &Service{Directory: t.TempDir()}
	copy := RemoteCopy{ID: uuid.NewString(), Format: 3, Size: 3, SHA256: strings.Repeat("a", 64), CreatedAt: time.Now().UTC()}
	raw, err := json.Marshal(copy)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, copy.ID+".manifest.json"), raw, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, copy.ID+".tar.gz"), []byte("abc"), 0600))
	require.ErrorContains(t, s.deleteRemote(context.Background(), remote, copy, 0), "минимум 1")
	require.FileExists(t, filepath.Join(dir, copy.ID+".tar.gz"))
	require.NoFileExists(t, filepath.Join(dir, copy.ID+".delete.json"))
}
