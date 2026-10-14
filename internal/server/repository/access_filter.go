package repository

import (
	"fmt"

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

func documentListAccessScope(explicit *models.DocumentAccessScope) models.DocumentAccessScope {
	if explicit != nil {
		return *explicit
	}
	return models.DocumentAccessScope{}
}

// applyDocumentListAccess appends the SQL equivalent of DocumentAccessScope.
// It only emits placeholders and keeps the query's argument counter in sync.
func applyDocumentListAccess(where *[]string, args *[]interface{}, argIdx *int, scope models.DocumentAccessScope) {
	if scope.UserID == "" {
		*where = append(*where, "1=0")
		return
	}
	*where = append(*where, documentReadPredicate(fmt.Sprintf("$%d", *argIdx), "d"))
	*args = append(*args, scope.UserID)
	*argIdx++
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
