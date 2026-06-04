package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppErrorDefaultsAndFormatting(t *testing.T) {
	err := &AppError{Code: 418, Kind: "TEAPOT", Message: "чайник"}

	assert.Equal(t, "TEAPOT: чайник", err.Error())
	assert.Equal(t, "чайник", err.SafeMessage())
	assert.Equal(t, "TEAPOT", err.SafeKind())
	assert.Equal(t, 418, err.StatusCode())

	empty := &AppError{}
	assert.Equal(t, "Error 0: ", empty.Error())
	assert.Equal(t, "произошла ошибка", empty.SafeMessage())
	assert.Equal(t, "INTERNAL_ERROR", empty.SafeKind())
	assert.Equal(t, 500, empty.StatusCode())
}

func TestAppErrorConstructorsAndUnwrap(t *testing.T) {
	internal := errors.New("db failed")
	err := NewInternal("ошибка базы", internal)

	assert.Equal(t, 500, err.Code)
	assert.Equal(t, "INTERNAL_ERROR", err.Kind)
	assert.ErrorIs(t, err, internal)

	assert.Equal(t, 400, NewBadRequest("bad").Code)
	assert.Equal(t, 401, NewUnauthorized("auth").Code)
	assert.Equal(t, 403, NewForbidden("forbidden").Code)
	assert.Equal(t, 404, NewNotFound("missing").Code)
	assert.Equal(t, 409, NewConflict("conflict").Code)
	assert.Equal(t, "IDEMPOTENCY_CONFLICT", NewIdempotencyConflict("idem").Kind)

	assert.ErrorIs(t, NewBadRequestWrapped("bad", internal), internal)
	assert.ErrorIs(t, NewForbiddenWrapped("forbidden", internal), internal)
	assert.ErrorIs(t, NewNotFoundWrapped("missing", internal), internal)
	assert.ErrorIs(t, NewConflictWrapped("conflict", internal), internal)
}

func TestAsAppError(t *testing.T) {
	appErr := NewBadRequest("bad")

	got, ok := AsAppError(appErr)

	require.True(t, ok)
	assert.Same(t, appErr, got)

	got, ok = AsAppError(errors.New("plain"))
	assert.False(t, ok)
	assert.Nil(t, got)
}
