package repository

import (
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func accessibleUserIDs(primary string, additional []string) []string {
	seen := make(map[string]struct{}, len(additional)+1)
	result := make([]string, 0, len(additional)+1)
	if primary != "" {
		seen[primary] = struct{}{}
		result = append(result, primary)
	}
	for _, id := range additional {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// documentListAccessScope resolves the transitional filter fields into one
// explicit scope. Direct repository callers retain their previous behaviour;
// production list requests receive AccessScope from DocumentQueryService.
func documentListAccessScope(explicit *models.DocumentAccessScope, allowedNomenclatureIDs []string, accessibleByUserID string, accessibleByUserIDs []string) models.DocumentAccessScope {
	if explicit != nil {
		return *explicit
	}
	return models.DocumentAccessScope{
		Restricted:             len(allowedNomenclatureIDs) > 0 || accessibleByUserID != "" || len(accessibleByUserIDs) > 0,
		AllowedNomenclatureIDs: allowedNomenclatureIDs,
		AccessibleByUserID:     accessibleByUserID,
		AccessibleByUserIDs:    accessibleByUserIDs,
	}
}

// applyDocumentListAccess appends the SQL equivalent of DocumentAccessScope.
// It only emits placeholders and keeps the query's argument counter in sync.
func applyDocumentListAccess(where *[]string, args *[]interface{}, argIdx *int, scope models.DocumentAccessScope) {
	if !scope.Restricted {
		return
	}

	accessibleIDs := accessibleUserIDs(scope.AccessibleByUserID, scope.AccessibleByUserIDs)
	if len(accessibleIDs) == 0 && len(scope.AllowedNomenclatureIDs) == 0 {
		*where = append(*where, "1=0")
		return
	}

	accessClauses := make([]string, 0, 3)
	if len(scope.AllowedNomenclatureIDs) > 0 {
		accessClauses = append(accessClauses, fmt.Sprintf("d.nomenclature_id = ANY($%d)", *argIdx))
		*args = append(*args, pq.Array(scope.AllowedNomenclatureIDs))
		*argIdx++
	}

	if len(accessibleIDs) > 0 {
		useAccessArray := len(scope.AccessibleByUserIDs) > 0
		assignmentUserPredicate := fmt.Sprintf("a.executor_id = $%d", *argIdx)
		coExecutorUserPredicate := fmt.Sprintf("ce.user_id = $%d", *argIdx)
		ackUserPredicate := fmt.Sprintf("au.user_id = $%d", *argIdx+1)
		assignmentArg := interface{}(scope.AccessibleByUserID)
		ackArg := interface{}(scope.AccessibleByUserID)
		if useAccessArray {
			assignmentUserPredicate = fmt.Sprintf("a.executor_id = ANY($%d::uuid[])", *argIdx)
			coExecutorUserPredicate = fmt.Sprintf("ce.user_id = ANY($%d::uuid[])", *argIdx)
			ackUserPredicate = fmt.Sprintf("au.user_id = ANY($%d::uuid[])", *argIdx+1)
			assignmentArg = pq.Array(accessibleIDs)
			ackArg = pq.Array(accessibleIDs)
		}

		accessClauses = append(accessClauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM assignments a
			WHERE a.document_id = d.id
			  AND (
				%s
				OR EXISTS (
					SELECT 1
					FROM assignment_co_executors ce
					WHERE ce.assignment_id = a.id AND %s
				)
			  )
		)`, assignmentUserPredicate, coExecutorUserPredicate))
		*args = append(*args, assignmentArg)
		*argIdx++

		accessClauses = append(accessClauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM acknowledgment_users au
			JOIN acknowledgments a ON au.acknowledgment_id = a.id
			WHERE %s
			  AND a.document_id = d.id
		)`, ackUserPredicate))
		*args = append(*args, ackArg)
		*argIdx++
	}

	*where = append(*where, "("+strings.Join(accessClauses, " OR ")+")")
}

func normalizePagination(page, pageSize int) (int, int) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if page <= 0 {
		page = 1
	}
	return page, pageSize
}
