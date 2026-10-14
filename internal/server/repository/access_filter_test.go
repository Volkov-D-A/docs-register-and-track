package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestApplyDocumentListAccess(t *testing.T) {
	t.Run("uses one current-user closure before filtering and pagination", func(t *testing.T) {
		where := []string{"d.kind=$1"}
		args := []interface{}{models.DocumentKindIncomingLetter}
		index := 2
		applyDocumentListAccess(&where, &args, &index, models.DocumentAccessScope{UserID: "user-id"})
		predicate := strings.Join(where, " AND ")
		assert.Contains(t, predicate, "WITH RECURSIVE visible")
		assert.Contains(t, predicate, "u.id=$2")
		assert.Contains(t, predicate, "document_deadline_controls")
		assert.Contains(t, predicate, "document_links")
		assert.NotContains(t, predicate, "assignment_recipients")
		assert.NotContains(t, predicate, "user_substitutions")
		assert.Equal(t, []interface{}{models.DocumentKindIncomingLetter, "user-id"}, args)
		assert.Equal(t, 3, index)
	})

	t.Run("fails closed for an empty restricted scope", func(t *testing.T) {
		where := []string{"d.kind = $1"}
		args := []interface{}{models.DocumentKindIncomingLetter}
		argIdx := 2

		applyDocumentListAccess(&where, &args, &argIdx, models.DocumentAccessScope{})

		assert.Equal(t, []string{"d.kind = $1", "1=0"}, where)
		assert.Len(t, args, 1)
		assert.Equal(t, 2, argIdx)
	})

	t.Run("fails closed even for an empty formerly unrestricted scope", func(t *testing.T) {
		where := []string{"d.kind = $1"}
		args := []interface{}{models.DocumentKindIncomingLetter}
		argIdx := 2

		applyDocumentListAccess(&where, &args, &argIdx, models.DocumentAccessScope{})

		assert.Equal(t, []string{"d.kind = $1", "1=0"}, where)
		assert.Len(t, args, 1)
		assert.Equal(t, 2, argIdx)
	})
}

func TestNormalizePagination(t *testing.T) {
	page, size := normalizePagination(0, 0)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, size)
	page, size = normalizePagination(3, 500)
	assert.Equal(t, 3, page)
	assert.Equal(t, 100, size)
}
