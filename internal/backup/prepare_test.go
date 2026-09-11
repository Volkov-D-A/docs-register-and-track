package backup

import (
	"context"
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
