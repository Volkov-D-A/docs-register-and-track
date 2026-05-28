# Static Analysis Review

Дата аудита: 2026-05-28
Этап: F.02.186-F.02.188

## Выполненные Проверки

- `go vet ./...`: passed.
- `go test ./...`: passed.
- `npm run build`: TypeScript compile + Vite production build passed.
- Search for suppressions found several `// @ts-ignore` comments in frontend code.
- No ESLint or Prettier config/script found.

## Findings

### Go

`go vet` passed. `gofmt -l` reports formatting drift in:

- `internal/logger/seq_writer.go`
- `internal/mocks/ReferenceStore.go`
- `internal/models/journal.go`
- `internal/repository/acknowledgment_repo_test.go`
- `internal/repository/attachment.go`
- `internal/services/system_service.go`

`internal/mocks/*` are generated and should be regenerated/formatted through the mock generation path, not hand-edited.

### TypeScript

TypeScript strict mode is enabled and build passes, but multiple files use `// @ts-ignore` to call generated Wails services or handle model gaps:

- `AcknowledgmentModal.tsx`
- `DocumentViewModal.tsx`
- `AcknowledgmentList.tsx`
- `JournalList.tsx`
- `DashboardPage.tsx`

There is also broad `any` usage in frontend document forms, pages and generated-adjacent types.

### ESLint

No ESLint config or lint script exists. Therefore F.02.188 cannot prove that disabled rules are controlled; linting is simply absent.

Связанные issues: `ISSUE-034`, `ISSUE-035`, `ISSUE-036`.
