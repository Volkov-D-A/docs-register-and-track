# Production Runtime Review

Дата аудита: 2026-05-28
Этап: E. Production runtime, first run, shutdown

## Общий Вывод

Runtime startup initializes config, logger, DB pool, repositories, services, MinIO, release notes, theme and then starts Wails. Frontend has first-run organization setup for admins when required settings are empty. Local user state is stored under user config dir.

After `ISSUE-028`, failures before UI exists (`config.Load`, DB/MinIO/release/theme/Wails init) write structured startup diagnostics and an operator-readable stderr block with component, resolved config path when relevant, next step and technical details. Seq startup is asynchronous/non-fatal and is treated as degraded logging if unavailable.

## Runtime Strengths

- Embedded frontend assets and migrations are used.
- Release notes are embedded and version is visible in About.
- Local release/theme state uses user config dir, not install directory.
- Backend error formatter now returns safe structured errors for Wails bindings.

## Runtime Gaps

- Startup config failure exits before UI but now includes actionable diagnostics.
- MinIO init failure exits before UI but now includes actionable diagnostics.
- DB ping failure now stops startup with PostgreSQL diagnostics.
- Shutdown now cancels/waits active lifecycle-aware backend operations before closing DB/logger after `ISSUE-015`; logger close is deterministic and idempotent.
- Context propagation for MinIO/file/statistics/link operations remains open.

## Связанные Issues

- `ISSUE-015`: fixed context/shutdown lifecycle; long-running smoke remains under `ISSUE-042`.
- `ISSUE-017`: fixed deterministic logger shutdown.
- `ISSUE-019`: fixed frontend structured error handling.
- `ISSUE-025`: fixed config lookup; target OS config smoke remains in release evidence.
- `ISSUE-028`: fixed startup diagnostics for fatal pre-UI startup failures.

## Smoke Tests To Add

- Start with missing config.
- Start with invalid JSON config.
- Start with encrypted config and missing/wrong encryption key.
- Start with PostgreSQL down.
- Start with MinIO down.
- Login and run migration status/action.
- Close app during upload/download/statistics/link graph operation.
