# Long Running And Cancellation Smoke Checklist

Дата обновления: 2026-06-02
Статус: maintained release smoke checklist

Run this checklist on Linux and Windows release artifacts with disposable PostgreSQL/MinIO data. Attach completed results to release evidence. For the full endurance pass, keep the app open for 4-8 hours; for release candidate triage, complete at least the fast loops and outage/shutdown scenarios.

## Preconditions

- Release artifact is built from a clean checkout.
- Disposable PostgreSQL database and MinIO bucket are available.
- Test data includes documents with files, links, assignments and statistics-visible records.
- Operator can stop/start PostgreSQL and MinIO on the disposable contour.
- Memory is tracked with OS tools, for example Task Manager, Resource Monitor, `ps`, `top` or equivalent.

## Endurance And Memory

- [ ] [LR-SMOKE-MEMORY-BASELINE] Record memory after login and dashboard load.
- [ ] [LR-SMOKE-MEMORY-SESSION] Keep the app open for 4-8 hours and record memory at start, hourly checkpoints and finish.
- [ ] [LR-SMOKE-MEMORY-RECOVERY] After repeated workflows, return to dashboard and confirm memory stabilizes or any growth is accepted with evidence.

## Repeated UI Workflows

- [ ] [LR-SMOKE-DOC-VIEW-LOOP] Open and close document view modal at least 100 times across document kinds.
- [ ] [LR-SMOKE-REGISTRATION-LOOP] Open/cancel registration and edit modals at least 50 times, including dirty discard and clean close paths.
- [ ] [LR-SMOKE-FILE-LOOP] Upload/download/delete attachments in a loop with small and configured-near-limit files on disposable data.
- [ ] [LR-SMOKE-STATS-LOOP] Refresh document, assignment and system statistics repeatedly across filters and periods.
- [ ] [LR-SMOKE-LINK-GRAPH-LOOP] Open link graph repeatedly for documents with empty, simple and multi-node links.

## Shutdown Cancellation

- [ ] [LR-SMOKE-SHUTDOWN-UPLOAD] Close the app during a file upload and verify shutdown finishes, no panic appears and partial data is recoverable or absent.
- [ ] [LR-SMOKE-SHUTDOWN-DOWNLOAD] Close the app during file download/download-to-disk and verify the operation cancels or completes without corrupting local files.
- [ ] [LR-SMOKE-SHUTDOWN-STATS] Close the app while statistics/storage info is loading and verify backend shutdown completes before DB/logger close.
- [ ] [LR-SMOKE-SHUTDOWN-LINK-GRAPH] Close the app while link graph is loading and verify no stuck process remains.

## Outage Recovery

- [ ] [LR-SMOKE-MINIO-UPLOAD-OUTAGE] Stop MinIO during upload and verify safe error text, no stuck loading state and retry works after MinIO returns.
- [ ] [LR-SMOKE-MINIO-DOWNLOAD-OUTAGE] Stop MinIO during download and verify safe error text and recovery after restart.
- [ ] [LR-SMOKE-MINIO-STATS-OUTAGE] Stop MinIO while system statistics loads and verify the app remains usable.
- [ ] [LR-SMOKE-DB-LIST-OUTAGE] Stop PostgreSQL during list/search refresh and verify safe error behavior, then restart and refresh successfully.
- [ ] [LR-SMOKE-DB-SAVE-OUTAGE] Stop PostgreSQL during disposable document save/update and verify no duplicate/partial user-visible result is accepted.

## Evidence

- [ ] Record target OS, artifact name, operator, date and data contour.
- [ ] Attach memory checkpoints and process-exit observations.
- [ ] Attach notes/screenshots for failed scenarios.
- [ ] Link accepted failures to `docs/known_issues.md`.
