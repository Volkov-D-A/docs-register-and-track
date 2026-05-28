# Smoke Test Changes From Stage G

Дата аудита: 2026-05-28

В проекте пока нет отдельного `SMOKE_TEST.md`; эти пункты нужно перенести в этап I / финальный smoke-test.

## Backend / DB

- `go test ./...`
- Disposable PostgreSQL integration tests with safe DSN guard.
- Fresh migration from empty DB.
- Document registration idempotency/concurrency/no-gaps.
- Journal/admin audit retention FK.

## Frontend / E2E

- Login/logout.
- Admin first-run organization setup.
- Register all 4 document kinds.
- Edit document.
- Upload/download/delete attachment.
- Assignment lifecycle.
- Acknowledgment lifecycle.
- Link documents and view graph.
- Forbidden/validation/not-found/conflict/internal error cases.

## Performance / Long Running

- Startup to login.
- Open each main list.
- Search/filter.
- Save registration form.
- Open statistics.
- Memory after repeated modals and 4-hour session.
- Close app during long file operation.
