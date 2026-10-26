package repository

import (
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// documentVisibilityScopes computes the readable closure once per principal,
// shared by all requested kinds. Empty identities never grant access.
func documentVisibilityScopes(scopes map[models.DocumentKind]models.DocumentAccessScope, args *[]interface{}) (string, string) {
	users := []string{}
	kindsByUser := map[string][]string{}
	for _, spec := range models.AllDocumentKindSpecs() {
		scope := scopes[spec.Code]
		if scope.UserID == "" {
			continue
		}
		if _, exists := kindsByUser[scope.UserID]; !exists {
			users = append(users, scope.UserID)
		}
		kindsByUser[scope.UserID] = append(kindsByUser[scope.UserID], string(spec.Code))
	}
	if len(users) == 0 {
		return "", ""
	}
	closures := []string{}
	allowed := []string{}
	for index, userID := range users {
		name := fmt.Sprintf("visible_%d", index)
		closure := visibleDocumentClosure(fmt.Sprintf("$%d", len(*args)+1))
		closures = append(closures, strings.ReplaceAll(strings.TrimPrefix(closure, "WITH RECURSIVE "), "visible", name))
		*args = append(*args, userID, pq.Array(kindsByUser[userID]))
		allowed = append(allowed, fmt.Sprintf("(d.kind = ANY($%d::text[]) AND d.id IN (SELECT id FROM %s))", len(*args), name))
	}
	return "WITH RECURSIVE " + strings.Join(closures, ", "), strings.Join(allowed, " OR ")
}
