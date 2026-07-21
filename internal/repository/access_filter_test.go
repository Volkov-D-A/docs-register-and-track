package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestApplyDocumentListAccess(t *testing.T) {
	t.Run("adds one shared predicate for nomenclature and substituted users", func(t *testing.T) {
		where := []string{"d.kind = $1"}
		args := []interface{}{models.DocumentKindIncomingLetter}
		argIdx := 2

		applyDocumentListAccess(&where, &args, &argIdx, models.DocumentAccessScope{
			Restricted:             true,
			AllowedNomenclatureIDs: []string{"nom-1"},
			AccessibleByUserID:     "user-1",
			AccessibleByUserIDs:    []string{"user-1", "user-2"},
		})

		predicate := strings.Join(where, " AND ")
		assert.Contains(t, predicate, "d.nomenclature_id = ANY($2)")
		assert.Contains(t, predicate, "a.executor_id = ANY($3::uuid[])")
		assert.Contains(t, predicate, "ce.user_id = ANY($3::uuid[])")
		assert.Contains(t, predicate, "au.user_id = ANY($4::uuid[])")
		assert.Contains(t, predicate, "acknowledgment_users au")
		assert.Len(t, args, 4)
		assert.Equal(t, 5, argIdx)
	})

	t.Run("fails closed for an empty restricted scope", func(t *testing.T) {
		where := []string{"d.kind = $1"}
		args := []interface{}{models.DocumentKindIncomingLetter}
		argIdx := 2

		applyDocumentListAccess(&where, &args, &argIdx, models.DocumentAccessScope{Restricted: true})

		assert.Equal(t, []string{"d.kind = $1", "1=0"}, where)
		assert.Len(t, args, 1)
		assert.Equal(t, 2, argIdx)
	})

	t.Run("leaves an explicit full-access scope unchanged", func(t *testing.T) {
		where := []string{"d.kind = $1"}
		args := []interface{}{models.DocumentKindIncomingLetter}
		argIdx := 2

		applyDocumentListAccess(&where, &args, &argIdx, models.DocumentAccessScope{})

		assert.Equal(t, []string{"d.kind = $1"}, where)
		assert.Len(t, args, 1)
		assert.Equal(t, 2, argIdx)
	})
}

func TestDocumentListAccessScopeLegacyCompatibility(t *testing.T) {
	scope := documentListAccessScope(nil, []string{"nom-1"}, "user-1", nil)
	assert.True(t, scope.Restricted)
	assert.Equal(t, "user-1", scope.AccessibleByUserID)
	assert.Equal(t, []string{"nom-1"}, scope.AllowedNomenclatureIDs)
}

func TestNormalizePagination(t *testing.T) {
	page, size := normalizePagination(0, 0)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, size)
	page, size = normalizePagination(3, 500)
	assert.Equal(t, 3, page)
	assert.Equal(t, 100, size)
}
