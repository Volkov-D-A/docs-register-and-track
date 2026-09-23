package backup

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPrepareRejectsFutureSchemaBeforeRunningPostgresTools(t *testing.T) {
	stage := t.TempDir()
	dump := filepath.Join(stage, "database.dump")
	require.NoError(t, os.WriteFile(dump, []byte("dump"), 0600))
	digest, size, err := FileDigest(dump)
	require.NoError(t, err)
	m := Manifest{ID: uuid.NewString(), Format: 3, Schema: 999, CreatedAt: time.Now().UTC(), DatabaseSHA256: digest, DatabaseSize: size, Objects: []Object{}}
	require.NoError(t, writeJSONFile(filepath.Join(stage, "manifest.json"), m))
	archive := filepath.Join(t.TempDir(), m.ID+".tar.gz")
	require.NoError(t, Pack(context.Background(), stage, archive, m))
	_, err = PrepareRestore(context.Background(), PostgreSQL{}, archive, filepath.Join(t.TempDir(), "contents"), 1024, 12)
	require.ErrorContains(t, err, "newer application")
}

func TestDumpSchemaInspectionRejectsDirtyMissingAndDuplicateRows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture is a POSIX executable; production Windows build is checked separately")
	}
	for _, tc := range []struct {
		name, rows string
		valid      bool
	}{{"old", "11\tf", true}, {"dirty", "12\tt", false}, {"missing", "", false}, {"duplicate", "12\tf\n12\tf", false}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fixture := "#!/bin/sh\ncat <<'DUMP'\nCOPY public.schema_migrations (version, dirty) FROM stdin;\n" + tc.rows + "\n\\.\nDUMP\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "pg_restore"), []byte(fixture), 0700))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			version, err := (PostgreSQL{}).DumpSchema(context.Background(), "unused")
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, 11, version)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestDownloadSelectedRejectsChangedOrCorruptRemoteCopy(t *testing.T) {
	good := []byte("verified archive")
	bad := []byte("modified archive")
	expected := RemoteCopy{
		ID: uuid.NewString(), Format: 3, Size: int64(len(good)),
		SHA256: fmt.Sprintf("%x", sha256.Sum256(good)), CreatedAt: time.Now().UTC(),
	}
	for _, tc := range []struct {
		name, issue  string
		remoteData   []byte
		changeMarker bool
		maxBytes     int64
	}{
		{name: "valid copy", remoteData: good, maxBytes: 1024},
		{name: "marker replaced", issue: "изменилась", remoteData: good, changeMarker: true, maxBytes: 1024},
		{name: "archive too large", issue: "staging limit", remoteData: good, maxBytes: 1},
		{name: "archive corrupted", issue: "checksum mismatch", remoteData: bad, maxBytes: 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remoteDir, stageDir := t.TempDir(), t.TempDir()
			marker := expected
			if tc.changeMarker {
				marker.CreatedAt = marker.CreatedAt.Add(time.Second)
			}
			raw, err := json.Marshal(marker)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(remoteDir, markerName(expected.ID)), raw, 0600))
			require.NoError(t, os.WriteFile(filepath.Join(remoteDir, expected.ID+".tar.gz"), tc.remoteData, 0600))
			archive, err := downloadSelected(context.Background(), catalogFiles(remoteDir), expected, stageDir, tc.maxBytes)
			if tc.issue != "" {
				require.ErrorContains(t, err, tc.issue)
				if tc.changeMarker || tc.maxBytes < expected.Size {
					require.NoFileExists(t, filepath.Join(stageDir, expected.ID+".tar.gz"))
				}
				return
			}
			require.NoError(t, err)
			contents, err := os.ReadFile(archive)
			require.NoError(t, err)
			require.Equal(t, good, contents)
		})
	}
}
