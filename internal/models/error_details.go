package models

import "errors"

// PublicErrorMessage allows only explicitly public, classified client errors.
// Constructor messages must be user-facing text, never arbitrary dependency errors.
func PublicErrorMessage(err error) (string, bool) {
	appErr, ok := AsAppError(err)
	if !ok || !appErr.Production || appErr.StatusCode() < 400 || appErr.StatusCode() >= 500 {
		return "", false
	}
	return appErr.SafeMessage(), true
}

// ErrorCauses preserves wrapped and joined causes, including AppError.Internal,
// for diagnostics. The limit also bounds malformed/cyclic error chains.
func ErrorCauses(err error) []string {
	var result []string
	var visit func(error)
	visit = func(current error) {
		if current == nil || len(result) >= 32 {
			return
		}
		result = append(result, current.Error())
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				visit(child)
			}
		} else {
			visit(errors.Unwrap(current))
		}
	}
	visit(err)
	return result
}

type requestError struct {
	cause error
	id    string
}

func (e *requestError) Error() string     { return e.cause.Error() + " (requestId: " + e.id + ")" }
func (e *requestError) Unwrap() error     { return e.cause }
func (e *requestError) RequestID() string { return e.id }

func WithRequestID(err error, id string) error {
	if err == nil || id == "" {
		return err
	}
	return &requestError{cause: err, id: id}
}

func ErrorRequestID(err error) string {
	var source interface{ RequestID() string }
	if errors.As(err, &source) {
		return source.RequestID()
	}
	return ""
}
