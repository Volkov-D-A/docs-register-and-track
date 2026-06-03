package logger

import (
	"bytes"
	"log"
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

func TestTechnicalContextHandlerWithAttrsAndGroup(t *testing.T) {
	var out bytes.Buffer
	baseHandler := &technicalContextHandler{Handler: slog.NewJSONHandler(&out, nil)}
	handler := baseHandler.
		WithAttrs([]slog.Attr{slog.String("component", "tests")}).
		WithGroup("request")

	slog.New(handler).Info("grouped log", "id", "42")

	logLine := out.String()
	if !strings.Contains(logLine, `"component":"tests"`) {
		t.Fatalf("log line does not contain component attr: %s", logLine)
	}
	if !strings.Contains(logLine, `"request":{"id":"42"}`) {
		t.Fatalf("log line does not contain grouped attr: %s", logLine)
	}
}

func TestStdLogFilter(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
		log.SetOutput(&stdLogFilter{})
	})
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, nil)))

	filter := &stdLogFilter{}
	n, err := filter.Write([]byte("[WebView2] Environment created successfully\n"))
	if err != nil {
		t.Fatalf("ignored Write returned error: %v", err)
	}
	if n == 0 {
		t.Fatal("ignored Write returned zero bytes")
	}
	if out.Len() != 0 {
		t.Fatalf("ignored log should not be written, got %s", out.String())
	}

	n, err = filter.Write([]byte("webview error\n"))
	if err != nil {
		t.Fatalf("error Write returned error: %v", err)
	}
	if n == 0 {
		t.Fatal("error Write returned zero bytes")
	}

	logLine := out.String()
	if !strings.Contains(logLine, `"msg":"webview error"`) {
		t.Fatalf("log line does not contain std log message: %s", logLine)
	}
	if !strings.Contains(logLine, `"source":"std_log"`) {
		t.Fatalf("log line does not contain std log source: %s", logLine)
	}
}
