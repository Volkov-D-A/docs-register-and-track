# Static Analysis Review

Дата аудита: 2026-05-28
Этап: F.02.186-F.02.188

## Выполненные Проверки

- `go vet ./...`: passed.
- `go test ./...`: passed.
- `npm run build`: TypeScript compile + Vite production build passed.
- Search for suppressions found several `// @ts-ignore` comments in frontend code.
- After remediation `ISSUE-034`, ESLint flat config and `npm run lint` exist.

## Findings

### Go

`go vet` passed. Formatting drift from the original audit was fixed after remediation; `gofmt -l main.go internal tools` now returns empty.

Previously affected files were:

- `internal/logger/seq_writer.go`
- `internal/mocks/ReferenceStore.go`
- `internal/models/journal.go`
- `internal/repository/acknowledgment_repo_test.go`
- `internal/repository/attachment.go`
- `internal/services/system_service.go`

`internal/mocks/*` are generated and should normally be regenerated/formatted through the mock generation path; this remediation only applied gofmt-compatible formatting.

### TypeScript

TypeScript strict mode is enabled and build passes. The `// @ts-ignore` comments found in the original audit were obsolete generated Wails service import suppressions and were removed after remediation:

- `AcknowledgmentModal.tsx`
- `DocumentViewModal.tsx`
- `AcknowledgmentList.tsx`
- `JournalList.tsx`
- `DashboardPage.tsx`

`rg -n '@ts-ignore|@ts-expect-error' frontend/src` now returns no matches. Broad `any` usage in frontend document forms, pages and generated-adjacent types remains a separate gradual typing concern.

### ESLint

ESLint flat config exists and is included in `make release-gate`. Initial scope is intentionally high-signal: hooks correctness as error, exhaustive-deps/unused-vars as warnings, and basic JS/TS rules. `npm run lint` currently passes with warnings that should be reduced gradually during frontend remediation.

Связанные issues: fixed `ISSUE-034`, `ISSUE-035`, `ISSUE-036`.
