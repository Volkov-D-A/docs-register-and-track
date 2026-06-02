# Long Running Review

Дата аудита: 2026-05-28
Этап: G.04.211-G.04.220

## Вывод

Long-running/memory/cancellation coverage is now maintained as release smoke after `ISSUE-042`. It follows lifecycle remediation from `ISSUE-015`:

- lifecycle-aware MinIO/link/statistics/journal paths;
- shutdown cancel/wait coordination before DB/logger close;
- maintained release evidence checklist for memory, repeated workflows, shutdown and outages.

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

- Execute maintained long-running smoke before release, then automate the highest-risk parts.
- Keep context cancellation tests after implementation and expand outage/shutdown coverage.
- Track memory via OS tools on both target platforms.

Связанные issues: fixed after audit: `ISSUE-015`, `ISSUE-017`, `ISSUE-020`, `ISSUE-028`, `ISSUE-041`, `ISSUE-042`, `ISSUE-043`.
