package backup

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDatabaseOnlyBackupRestoreIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	for _, tool := range []string{"pg_dump", "pg_restore"} {
		_, err := exec.LookPath(tool)
		require.NoError(t, err, "PostgreSQL tools are required")
	}
	connection, err := url.Parse(os.Getenv("DOCFLOW_INTEGRATION_DSN"))
	require.NoError(t, err)
	require.Equal(t, "postgres", connection.Scheme, "this backup fixture requires a URL DSN")
	port, err := strconv.Atoi(connection.Port())
	require.NoError(t, err)
	password, _ := connection.User.Password()
	pg := PostgreSQL{Config: config.DatabaseConfig{Host: connection.Hostname(), Port: port, User: connection.User.Username(), Password: password, DBName: strings.TrimPrefix(connection.Path, "/"), SSLMode: "disable"}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	user, nom, doc, main, co, control := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,login,password_hash,last_name,first_name,no_patronymic,password_change_required) VALUES($1,'backup-control','test','Backup','User',true,false)`, user)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions(user_id,permission) VALUES($1,'admin')`, user)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nomenclature(id,name,index,year,kind_code) VALUES($1,'Backup','B',2026,'incoming_letter')`, nom)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO documents(id,kind,nomenclature_id,registration_number,registration_date,document_type,content,created_by) VALUES($1,'incoming_letter',$2,'B/1','2026-10-14','Письмо','Backup',$3)`, doc, nom, user)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO deadline_executors(id,name,institution) VALUES($1,'Основной','Учреждение'),($2,'Соисполнитель','Учреждение')`, main, co)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO document_deadline_controls(id,document_id,created_by,executor_id,co_executor_ids,subject,due_date) VALUES($1,$2,$3,$4,ARRAY[$5]::uuid[],'Сохранить контроль','2026-10-20')`, control, doc, user, main, co)
	require.NoError(t, err)
	service := Service{DB: db}
	require.NoError(t, service.requireDedicatedTarget(ctx))
	require.NoError(t, ValidateControlReferences(ctx, db))
	stage := t.TempDir()
	dump := filepath.Join(stage, "database.dump")
	require.NoError(t, pg.Dump(ctx, dump, 16<<20))
	digest, size, err := FileDigest(dump)
	require.NoError(t, err)
	m := Manifest{Format: ArchiveFormat, ID: uuid.NewString(), CreatedAt: time.Now().UTC(), DatabaseSHA256: digest, DatabaseSize: size}
	require.NoError(t, db.QueryRow(`SELECT version FROM schema_migrations`).Scan(&m.Schema))
	require.NoError(t, writeJSONFile(filepath.Join(stage, "manifest.json"), m))
	archive := filepath.Join(t.TempDir(), "copy.tar.gz")
	require.NoError(t, Pack(ctx, stage, archive, m))
	prepared, err := PrepareRestore(ctx, pg, archive, filepath.Join(t.TempDir(), "verified"), 16<<20, m.Schema)
	require.NoError(t, err)
	require.Equal(t, ArchiveFormat, prepared.Manifest.Format)
	// This reset is confined to the integrationdb-validated disposable database.
	_, err = db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	require.NoError(t, err)
	recovery := t.TempDir()
	require.NoError(t, Restore(ctx, pg, archive, recovery, 16<<20, m.Schema))
	var subject, executor string
	var coCount int
	require.NoError(t, db.QueryRow(`SELECT subject,executor_id::text,cardinality(co_executor_ids) FROM document_deadline_controls WHERE id=$1`, control).Scan(&subject, &executor, &coCount))
	require.Equal(t, "Сохранить контроль", subject)
	require.Equal(t, main.String(), executor)
	require.Equal(t, 1, coCount)
	_, err = os.Stat(filepath.Join(recovery, "recovery-required"))
	require.True(t, os.IsNotExist(err))
}
