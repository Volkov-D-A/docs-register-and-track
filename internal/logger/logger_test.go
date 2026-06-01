package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestTechnicalContextHandlerAddsOnlyUserID(t *testing.T) {
	const userID = "4f8fb26b-c235-4dd9-8bd7-926053db2e20"
	const fullName = "Ivanov Ivan Ivanovich"

	previous := GetAppUserID
	GetAppUserID = func() string {
		return userID
	}
	t.Cleanup(func() {
		GetAppUserID = previous
	})

	var out bytes.Buffer
	handler := &technicalContextHandler{Handler: slog.NewJSONHandler(&out, nil)}
	slog.New(handler).Info("test log", "note", "technical")

	logLine := out.String()
	if !strings.Contains(logLine, `"app_user_id":"`+userID+`"`) {
		t.Fatalf("log line does not contain app_user_id: %s", logLine)
	}
	if strings.Contains(logLine, `"app_user":`) {
		t.Fatalf("log line contains legacy app_user field: %s", logLine)
	}
	if strings.Contains(logLine, fullName) {
		t.Fatalf("log line contains full name: %s", logLine)
	}
}
