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
