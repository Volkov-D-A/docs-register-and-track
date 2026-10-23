package repository

import (
	"encoding/json"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestPermissionSubjectsAndSystemPermissionKeyIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	userID := insertIntegrationUser(t, db, "schema-permissions")
	for _, subject := range []string{"user", "department"} {
		_, err := db.Exec(`INSERT INTO document_permissions (kind_code,subject_type,subject_key,action) VALUES ('incoming_letter',$1,$2,'register')`, subject, userID.String())
		require.NoError(t, err)
	}
	_, err := db.Exec(`INSERT INTO document_permissions (kind_code,subject_type,subject_key,action) VALUES ('incoming_letter','role','clerk','register')`)
	var pgErr *pq.Error
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, pq.ErrorCode("23514"), pgErr.Code)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id,permission) VALUES ($1,'admin')`, userID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions (user_id,permission) VALUES ($1,'admin')`, userID)
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, pq.ErrorCode("23505"), pgErr.Code)
	require.Equal(t, "user_system_permissions_pkey", pgErr.Constraint)
}

// Each removed prefix-only index has a remaining index for its actual query shape.
func TestAuditIndexesSupportCurrentQueriesIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.Exec(`SET LOCAL enable_seqscan=off`)
	require.NoError(t, err)
	for _, item := range []struct {
		name, query, index string
		args               []any
	}{
		{"system permissions", `SELECT permission FROM user_system_permissions WHERE user_id=$1`, "user_system_permissions_pkey", []any{uuid.New()}},
		{"document kind and order", `SELECT id FROM documents WHERE kind=$1 ORDER BY created_at DESC LIMIT 20`, "idx_documents_kind_created_at", []any{"incoming_letter"}},
		{"ordered resolutions", `SELECT resolution FROM document_resolutions WHERE document_id=$1 ORDER BY position`, "idx_document_resolutions_document_position", []any{uuid.New()}},
		{"outgoing graph edges", `SELECT target_document_id FROM document_links WHERE source_document_id=$1`, "document_links_source_document_id_target_document_id_key", []any{uuid.New()}},
	} {
		t.Run(item.name, func(t *testing.T) {
			var raw []byte
			require.NoError(t, tx.QueryRow(`EXPLAIN (FORMAT JSON) `+item.query, item.args...).Scan(&raw))
			var plan []map[string]any
			require.NoError(t, json.Unmarshal(raw, &plan))
			indexes := []string{}
			var visit func(any)
			visit = func(value any) {
				switch node := value.(type) {
				case map[string]any:
					if name, ok := node["Index Name"].(string); ok {
						indexes = append(indexes, name)
					}
					for _, child := range node {
						visit(child)
					}
				case []any:
					for _, child := range node {
						visit(child)
					}
				}
			}
			visit(plan[0])
			require.Contains(t, indexes, item.index, string(raw))
		})
	}
}
