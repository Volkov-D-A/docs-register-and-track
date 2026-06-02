package startupdiag

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Failure describes a startup failure in terms that are safe to show to an
// operator while preserving technical detail in logs.
type Failure struct {
	Component  string
	ConfigPath string
	Summary    string
	NextStep   string
	Err        error
}

func (f Failure) normalized() Failure {
	f.Component = strings.TrimSpace(f.Component)
	f.ConfigPath = strings.TrimSpace(f.ConfigPath)
	f.Summary = strings.TrimSpace(f.Summary)
	f.NextStep = strings.TrimSpace(f.NextStep)
	if f.Component == "" {
		f.Component = "startup"
	}
	if f.Summary == "" {
		f.Summary = "Приложение не смогло завершить запуск."
	}
	if f.NextStep == "" {
		f.NextStep = "Откройте docs/diagnostics_runbook.md и приложите технический лог к заявке."
	}
	return f
}

// Write prints an operator-readable diagnostic block to the supplied writer.
func Write(w io.Writer, failure Failure) {
	f := failure.normalized()
	fmt.Fprintln(w, "DocFlow startup diagnostics")
	fmt.Fprintf(w, "Component: %s\n", f.Component)
	if f.ConfigPath != "" {
		fmt.Fprintf(w, "Config path: %s\n", f.ConfigPath)
	}
	fmt.Fprintf(w, "Problem: %s\n", f.Summary)
	fmt.Fprintf(w, "Next step: %s\n", f.NextStep)
	if f.Err != nil {
		fmt.Fprintf(w, "Technical details: %v\n", f.Err)
	}
}

// Log writes the same failure as structured startup diagnostics.
func Log(logger *slog.Logger, failure Failure) {
	f := failure.normalized()
	if logger == nil {
		logger = slog.Default()
	}

	attrs := []any{
		"type", "startup_diagnostics",
		"component", f.Component,
		"summary", f.Summary,
		"next_step", f.NextStep,
	}
	if f.ConfigPath != "" {
		attrs = append(attrs, "config_path", f.ConfigPath)
	}
	if f.Err != nil {
		attrs = append(attrs, "error", f.Err)
	}
	logger.Error("Startup diagnostics", attrs...)
}
