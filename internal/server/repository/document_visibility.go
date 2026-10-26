package repository

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (r *DocumentAccessRepository) CanReadDocument(userID, documentID uuid.UUID) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(visibleDocumentClosure("$1")+`SELECT EXISTS(SELECT 1 FROM visible WHERE id=$2)`, userID, documentID).Scan(&allowed)
	return allowed, err
}
func (r *DocumentAccessRepository) GetReadableDocumentIDs(userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(visibleDocumentClosure("$1")+`SELECT id FROM visible WHERE id=ANY($2::uuid[])`, userID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}
func documentReadPredicate(userPlaceholder, documentAlias string) string {
	return `EXISTS (` + visibleDocumentClosure(userPlaceholder) + fmt.Sprintf(`SELECT 1 FROM visible WHERE visible.id=%s.id)`, documentAlias)
}

// Every seed must refer to the authenticated, active user. UNION terminates cycles
// and deduplicates multiple executor and link paths before any aggregate/limit.
func visibleDocumentClosure(user string) string {
	return `WITH RECURSIVE visible(id) AS (
 SELECT d.id FROM documents d JOIN users u ON u.id=` + user + ` AND u.is_active
 WHERE EXISTS(SELECT 1 FROM document_permissions p WHERE p.kind_code=d.kind AND p.is_allowed
   AND p.action IN ('register','control_deadlines')
   AND ((p.subject_type='user' AND p.subject_key=u.id::text)
     OR (p.subject_type='department' AND p.subject_key=u.department_id::text)))
 OR EXISTS(SELECT 1 FROM department_nomenclature n WHERE n.department_id=u.department_id AND n.nomenclature_id=d.nomenclature_id)
 OR EXISTS(SELECT 1 FROM document_deadline_controls c JOIN deadline_executors e
   ON (e.id=c.executor_id OR e.id=ANY(c.co_executor_ids))
   WHERE c.document_id=d.id AND c.status IN ('active','completed') AND e.user_id=u.id)
 UNION
 SELECT CASE WHEN l.source_document_id=v.id THEN l.target_document_id ELSE l.source_document_id END
 FROM visible v JOIN document_links l ON (l.source_document_id=v.id OR l.target_document_id=v.id)
 ) `
}
