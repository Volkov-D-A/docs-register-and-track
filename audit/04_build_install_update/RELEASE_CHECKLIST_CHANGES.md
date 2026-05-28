# Release Checklist Changes From Stage E

Дата аудита: 2026-05-28

В проекте пока нет отдельного `RELEASE_CHECKLIST.md`; эти пункты нужно перенести в этап I / финальный release checklist.

## Build Gate

- Clean checkout.
- Required release secrets present and validated before build.
- `npm ci`.
- `npm run build`.
- `go generate ./internal/releaseassets`.
- Generated release assets have no unexpected diff.
- `go test ./...`.
- Wails Linux/Windows build.
- Artifact version check: About, binary metadata, installer DisplayVersion match.

## Install / First Run

- Windows install into standard path.
- Launch from shortcut as non-admin user.
- Launch from path with spaces and Cyrillic.
- Launch from different working directory.
- Missing/invalid/unreadable config produces actionable diagnostics.
- Encrypted config with missing/wrong key produces actionable diagnostics.

## Update / Migration

- Current binary upgrades old DB safely.
- Older binary refuses newer DB schema.
- Dirty migration state blocks unsafe use and points to runbook.
- Rollback requires warning, confirmation, fresh backup and audit entry.
- Restore test PostgreSQL+MinIO completed before release.

## Filesystem / Secrets

- Duplicate attachment filenames do not overwrite local files silently.
- OpenFile/OpenFolder cannot escape Downloads.
- Backup/restore interrupted on each stage leaves no temp dump/files.
- SMB credentials are not exposed in process list/logs.
- Seq technical logs do not contain secrets or unnecessary PII.
