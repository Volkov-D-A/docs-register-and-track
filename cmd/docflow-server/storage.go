package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
)

func runStorage(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("storage requires check, bucket-check, export or import")
	}
	operation := args[0]
	switch operation {
	case "check", "bucket-check":
		if len(args) != 1 {
			return fmt.Errorf("unexpected storage arguments")
		}
	case "export", "import":
		if len(args) != 2 {
			return fmt.Errorf("storage %s requires a directory", operation)
		}
	default:
		return fmt.Errorf("unknown storage operation %q", operation)
	}
	cfg, err := config.LoadServer()
	if err != nil {
		return err
	}
	if operation != "check" && cfg.S3.BucketName == "" {
		return fmt.Errorf("S3_BUCKET is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	switch operation {
	case "check":
		limited, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		err = storage.ProbeS3(limited, cfg.S3)
	case "bucket-check":
		err = storage.CheckS3(ctx, cfg.S3)
	case "export":
		err = storage.ExportDirectory(ctx, cfg.S3, args[1])
	case "import":
		err = storage.ImportDirectory(ctx, cfg.S3, args[1])
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "storage operation completed")
	return nil
}
