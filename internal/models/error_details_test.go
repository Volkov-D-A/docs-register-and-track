package models

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicErrorMessageRequiresExplicitClassification(t *testing.T) {
	for _, err := range []error{nil, errors.New("private"), NewInternal("private", nil), &AppError{Code: 409, Message: "private"}, &AppError{Code: 500, Message: "private", Production: true}} {
		_, ok := PublicErrorMessage(err)
		assert.False(t, ok)
	}
	msg, ok := PublicErrorMessage(fmt.Errorf("private wrapper: %w", NewBadRequestWrapped("safe validation", errors.New("private cause"))))
	require.True(t, ok)
	assert.Equal(t, "safe validation", msg)
}

func TestErrorDetailsSurviveWrapping(t *testing.T) {
	source := NewInternal("generic", errors.Join(errors.New("private DB"), errors.New("private storage")))
	err := NewConflictWrapped("safe conflict", WithRequestID(source, "request-1"))
	assert.Equal(t, "request-1", ErrorRequestID(err))
	assert.Contains(t, ErrorCauses(err), "private DB")
	assert.Contains(t, ErrorCauses(err), "private storage")
	assert.ErrorIs(t, WithRequestID(ErrUnauthorized, "request-2"), ErrUnauthorized)
	assert.Same(t, ErrUnauthorized, WithRequestID(ErrUnauthorized, ""))
	assert.Nil(t, WithRequestID(nil, "request-2"))
}
