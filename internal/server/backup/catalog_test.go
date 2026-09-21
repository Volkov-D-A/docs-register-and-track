package backup

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type catalogFiles string

func (d catalogFiles) List(context.Context) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(string(d))
	if err != nil {
		return nil, err
	}
	var result []os.FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}
func (d catalogFiles) Open(_ context.Context, name string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(string(d), name))
}

func TestCatalogIndependentOfHistoryAndHonestAboutVerification(t *testing.T) {
	dir := t.TempDir()
	id, missing, orphan, mismatch := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, key := range []string{id, missing, mismatch} {
		raw, err := json.Marshal(RemoteCopy{ID: key, Format: 3, Size: 3, SHA256: strings.Repeat("a", 64), CreatedAt: time.Now().UTC()})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, markerName(key)), raw, 0600))
	}
	for _, key := range []string{id, orphan} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, key+".tar.gz"), []byte("abc"), 0600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, mismatch+".tar.gz"), []byte("truncated"), 0600))
	legacy := "backup_20260910_090000"
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacy+".tar.gz"), []byte("abc"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, legacy+".tar.gz.manifest"), []byte("format_version=2\narchive="+legacy+".tar.gz\nsize_bytes=3\nsha256="+strings.Repeat("a", 64)+"\n"), 0600))
	copies, err := Catalog(context.Background(), catalogFiles(dir))
	require.NoError(t, err)
	require.Len(t, copies, 4)
	for _, copy := range copies {
		if copy.ID == id {
			require.Equal(t, "unverified", copy.Verification)
			require.True(t, copy.CanDelete)
		} else {
			require.Equal(t, "incomplete", copy.Verification)
			require.NotEmpty(t, copy.Issue)
			require.False(t, copy.CanDelete)
		}
	}
}

func TestCopyIdentifiersAndManifest(t *testing.T) {
	for _, id := range []string{"../archive", "/tmp/archive", "backup_20260910_090000/../a", "backup_20260910_090000", "backup_20260910_090000_123456789", "", strings.ToUpper(uuid.NewString())} {
		require.False(t, validCopyID(id), id)
	}
	id := uuid.NewString()
	marker := RemoteCopy{ID: id, Format: 3, Size: 42, SHA256: strings.Repeat("a", 64), CreatedAt: time.Now().UTC()}
	raw, err := json.Marshal(marker)
	require.NoError(t, err)
	parsed, err := parseCopyMarker(id, raw)
	require.NoError(t, err)
	require.Equal(t, marker, parsed)
	for _, mutate := range []func(*RemoteCopy){
		func(m *RemoteCopy) { m.Format = 2 },
		func(m *RemoteCopy) { m.ID = "../escape" },
		func(m *RemoteCopy) { m.Size = -1 },
		func(m *RemoteCopy) { m.Size = 1 << 60 },
		func(m *RemoteCopy) { m.SHA256 = strings.Repeat("z", 64) },
		func(m *RemoteCopy) { m.CreatedAt = time.Time{} },
	} {
		invalid := marker
		mutate(&invalid)
		raw, err := json.Marshal(invalid)
		require.NoError(t, err)
		_, err = parseCopyMarker(id, raw)
		require.Error(t, err)
	}
	for _, raw := range []string{"format_version=2\narchive=" + id + ".tar.gz", strings.Repeat("a", 4097)} {
		_, err := parseCopyMarker(id, []byte(raw))
		require.Error(t, err)
	}
}

func TestCatalogUnavailableAndCancelled(t *testing.T) {
	_, err := Catalog(context.Background(), catalogFiles(filepath.Join(t.TempDir(), "missing")))
	require.Error(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, uuid.NewString()+".tar.gz"), nil, 0600))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Catalog(ctx, catalogFiles(dir))
	require.ErrorIs(t, err, context.Canceled)
}
