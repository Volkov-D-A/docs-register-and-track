package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/logger"
	"github.com/Volkov-D-A/docs-register-and-track/internal/releaseassets"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server"
	"github.com/Volkov-D-A/docs-register-and-track/internal/startupdiag"
)

const usage = `Docflow server

Usage:
  docflow-server run
  docflow-server check-config
  docflow-server healthcheck
  docflow-server version
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var failure *startupdiag.Failure
		if errors.As(err, &failure) {
			startupdiag.Log(slog.Default(), *failure)
			startupdiag.Write(os.Stderr, *failure)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return fmt.Errorf("command is required")
	}
	if args[0] == "version" {
		if len(args) != 1 {
			return fmt.Errorf("version command does not accept arguments")
		}
		version, err := releaseassets.CurrentVersion()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, version)
		return nil
	}

	command := args[0]
	if command != "run" && command != "check-config" && command != "healthcheck" {
		fmt.Fprint(stderr, usage)
		return fmt.Errorf("unknown command %q", command)
	}

	if len(args) != 1 {
		return fmt.Errorf("%s command does not accept arguments", command)
	}

	cfg, err := config.LoadServer()
	if err != nil {
		return &startupdiag.Failure{
			Component: "configuration",
			Summary:   "Не удалось загрузить конфигурацию docflow-server.",
			NextStep:  "Проверьте переменные PostgreSQL, MinIO, Seq, outbox и ENCRYPTION_KEY.",
			Err:       err,
		}
	}
	if err := server.ValidateConfig(cfg); err != nil {
		return &startupdiag.Failure{
			Component: "configuration",
			Summary:   "Конфигурация docflow-server не прошла проверку.",
			NextStep:  "Исправьте указанные переменные окружения и повторите проверку.",
			Err:       err,
		}
	}
	if command == "check-config" {
		fmt.Fprintln(stdout, "configuration is valid")
		return nil
	}

	_, closeLogger := logger.Init(cfg.Seq)
	var closeLoggerOnce sync.Once
	closeLoggerSafely := func() { closeLoggerOnce.Do(closeLogger) }
	defer closeLoggerSafely()

	if command == "healthcheck" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.ProbeLive(ctx, cfg.Server.ListenAddress); err != nil {
			return &startupdiag.Failure{
				Component: "server health",
				Summary:   "Management API docflow-server не отвечает.",
				NextStep:  "Проверьте процесс, DOCFLOW_SERVER_LISTEN_ADDRESS и технический лог.",
				Err:       err,
			}
		}
		fmt.Fprintln(stdout, "healthy")
		return nil
	}

	application, err := server.New(cfg)
	if err != nil {
		return &startupdiag.Failure{
			Component: "server startup",
			Summary:   "Не удалось создать docflow-server.",
			NextStep:  "Проверьте PostgreSQL, миграции, MinIO, environment и технический лог.",
			Err:       err,
		}
	}
	defer application.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := application.Run(ctx); err != nil {
		return &startupdiag.Failure{
			Component: "server runtime",
			Summary:   "docflow-server остановлен из-за ошибки.",
			NextStep:  "Проверьте состояние схемы, PostgreSQL, MinIO и технический лог.",
			Err:       err,
		}
	}
	return nil
}
