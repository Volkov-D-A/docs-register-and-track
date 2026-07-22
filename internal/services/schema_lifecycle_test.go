package services

import (
	"errors"
	"sync"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
)

type fakeSchemaLifecycle struct {
	mu                 sync.Mutex
	reconcileCalls     int
	prepareCalls       int
	completeResults    []bool
	checkReadyErr      error
	prepareRollbackErr error
}

func (l *fakeSchemaLifecycle) ReconcileSchema() {
	l.mu.Lock()
	l.reconcileCalls++
	l.mu.Unlock()
}

func (l *fakeSchemaLifecycle) PrepareRollback() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prepareCalls++
	return l.prepareRollbackErr
}

func (l *fakeSchemaLifecycle) CompleteRollback(success bool) {
	l.mu.Lock()
	l.completeResults = append(l.completeResults, success)
	l.mu.Unlock()
}

func (l *fakeSchemaLifecycle) CheckReady() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.checkReadyErr
}

type fakeMigrationDatabase struct {
	runErr        error
	rollbackErr   error
	status        *database.MigrationStatus
	statusErr     error
	runCalls      int
	rollbackCalls int
	statusCalls   int
}

func (db *fakeMigrationDatabase) RunMigrations(string) error {
	db.runCalls++
	return db.runErr
}

func (db *fakeMigrationDatabase) GetMigrationStatus(string) (*database.MigrationStatus, error) {
	db.statusCalls++
	if db.statusErr != nil {
		return nil, db.statusErr
	}
	if db.status == nil {
		return nil, errors.New("migration status is not configured")
	}
	status := *db.status
	return &status, nil
}

func (db *fakeMigrationDatabase) RollbackMigration(string) error {
	db.rollbackCalls++
	return db.rollbackErr
}
