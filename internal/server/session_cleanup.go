package server

import (
	"context"
	"log/slog"
	"time"
)

type expiredSessionStore interface {
	DeleteExpired(time.Time) (int64, error)
}

type sessionCleaner struct {
	store expiredSessionStore
}

func (c *sessionCleaner) Run(ctx context.Context) error {
	if c == nil || c.store == nil {
		return nil
	}
	if err := c.cleanup(); err != nil {
		slog.Warn("failed to clean expired server sessions", "error", err)
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.cleanup(); err != nil {
				slog.Warn("failed to clean expired server sessions", "error", err)
			}
		}
	}
}

func (c *sessionCleaner) cleanup() error {
	deleted, err := c.store.DeleteExpired(time.Now().UTC())
	if err != nil {
		return err
	}
	if deleted > 0 {
		slog.Info("expired server sessions cleaned", "deleted", deleted)
	}
	return nil
}
