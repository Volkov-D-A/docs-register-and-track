# Production Runtime Review

Дата аудита: 2026-05-28
Этап: E. Production runtime, first run, shutdown

## Общий Вывод

Runtime startup initializes config, logger, DB pool, repositories, services, MinIO, release notes, theme and then starts Wails. Frontend has first-run organization setup for admins when required settings are empty. Local user state is stored under user config dir.

Главный runtime gap: часть failures occurs before UI exists (`config.Load`, DB/MinIO/release/theme init) and exits process via `log.Fatalf`/`os.Exit`. For production desktop this should become an operator-visible startup diagnostics screen or a documented launcher/runbook behavior.

## Runtime Strengths

- Embedded frontend assets and migrations are used.
- Release notes are embedded and version is visible in About.
- Local release/theme state uses user config dir, not install directory.
- Backend error formatter now returns safe structured errors for Wails bindings.

## Runtime Gaps

- Startup config failure exits before UI.
- MinIO init failure exits before UI.
- DB ping failure is warning-only, but later DB operations may fail in UI.
- Shutdown still closes DB without active request coordination (`ISSUE-015`); logger close is now deterministic and idempotent.
- Context propagation for MinIO/file/statistics/link operations remains open.

## Связанные Issues

- `ISSUE-015`: context/shutdown lifecycle.
- `ISSUE-017`: fixed deterministic logger shutdown.
- `ISSUE-019`: fixed frontend structured error handling.
- `ISSUE-025`: fixed config lookup; target OS config smoke remains in release evidence.
- `ISSUE-028`: startup/update failure UX.

## Smoke Tests To Add

- Start with missing config.
- Start with invalid JSON config.
- Start with encrypted config and missing/wrong encryption key.
- Start with PostgreSQL down.
- Start with MinIO down.
- Login and run migration status/action.
- Close app during upload/download/statistics/link graph operation.
