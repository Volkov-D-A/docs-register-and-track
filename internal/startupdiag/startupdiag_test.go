package startupdiag

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteIncludesOperatorFields(t *testing.T) {
	var out bytes.Buffer
	Write(&out, Failure{
		Component:  "PostgreSQL",
		ConfigPath: "/opt/docflow/config/config.json",
		Summary:    "Не удалось подключиться к базе данных.",
		NextStep:   "Проверьте доступность PostgreSQL из рабочего места.",
		Err:        errors.New("dial tcp 10.0.0.5:5432: connect: connection refused"),
	})

	text := out.String()
	for _, want := range []string{
		"DocFlow startup diagnostics",
		"Component: PostgreSQL",
		"Config path: /opt/docflow/config/config.json",
		"Problem: Не удалось подключиться к базе данных.",
		"Next step: Проверьте доступность PostgreSQL из рабочего места.",
		"Technical details: dial tcp 10.0.0.5:5432: connect: connection refused",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("diagnostics output does not contain %q:\n%s", want, text)
		}
	}
}

func TestWriteUsesDefaults(t *testing.T) {
	var out bytes.Buffer
	Write(&out, Failure{})

	text := out.String()
	for _, want := range []string{
		"Component: startup",
		"Problem: Приложение не смогло завершить запуск.",
		"Next step: Откройте docs/diagnostics_runbook.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("diagnostics output does not contain default %q:\n%s", want, text)
		}
	}
}
