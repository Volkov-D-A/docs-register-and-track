# Long Running Review

Дата аудита: 2026-05-28
Этап: G.04.211-G.04.220

## Вывод

No long-running/memory/cancellation test suite was found. This aligns with open lifecycle issues from C/E:

- `context.Background()` in MinIO/link/statistics/journal paths;
- shutdown closes DB/logger without active request coordination;
- no cancellation/progress UX for long operations.

## Missing Scenarios

- 4-8 hour desktop session memory trend.
- Repeated open/close document view modal.
- Repeated open/close registration/edit modals.
- Repeated file upload/download/delete.
- Statistics/report refresh loop.
- Link graph open/close loop.
- Closing app during upload/download/statistics.
- MinIO/network timeout during file operations.
- DB disconnect/reconnect during list/save.
- Destructive actions confirmation and rollback guardrails.

## Recommendations

- Add manual long-running smoke before release, then automate the highest-risk parts.
- Add context cancellation first; tests should assert cancellation behavior after implementation.
- Track memory via OS tools on both target platforms.

Связанные issues: `ISSUE-015`, `ISSUE-028`, `ISSUE-041`, `ISSUE-043`; fixed after audit: `ISSUE-017`, `ISSUE-020`.
