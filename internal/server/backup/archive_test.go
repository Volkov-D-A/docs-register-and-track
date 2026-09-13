package backup

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchiveRoundtripRejectsCorruptionAndLimits(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "database.dump"), []byte("dump bytes"), 0600))
	digest, size, err := FileDigest(filepath.Join(dir, "database.dump"))
	require.NoError(t, err)
	m := Manifest{Format: 3, ID: "00000000-0000-4000-8000-000000000001", Schema: 11, CreatedAt: time.Now().UTC(), DatabaseSHA256: digest, DatabaseSize: size, Objects: []Object{}}
	require.NoError(t, writeJSONFile(filepath.Join(dir, "manifest.json"), m))
	archive := filepath.Join(t.TempDir(), "backup.tar.gz")
	require.NoError(t, Pack(context.Background(), dir, archive, m))
	restored, err := Unpack(context.Background(), archive, filepath.Join(t.TempDir(), "restored"), 1024)
	require.NoError(t, err)
	require.Equal(t, m, restored)
	_, err = Unpack(context.Background(), archive, filepath.Join(t.TempDir(), "small"), 5)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "database.dump"), []byte("bad"), 0600))
	archive2 := filepath.Join(t.TempDir(), "bad.tar.gz")
	require.NoError(t, Pack(context.Background(), dir, archive2, m))
	_, err = Unpack(context.Background(), archive2, filepath.Join(t.TempDir(), "bad"), 1024)
	require.ErrorContains(t, err, "checksum")
}
