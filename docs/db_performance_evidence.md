# Database Performance Evidence Checklist

Дата обновления: 2026-06-02
Статус: maintained release evidence checklist

Use this checklist on a disposable production-like PostgreSQL dataset before adding performance indexes or approving a production candidate. Do not add composite, partial or trigram indexes unless the captured plan and latency evidence shows a real degradation against the release budgets.

## Dataset

- [ ] [DB-PERF-DATASET-SIZE] Record document, assignment, acknowledgment, attachment, journal and audit-log row counts.
- [ ] [DB-PERF-DATASET-ANALYZE] Run `ANALYZE` after loading production-like data and before collecting plans.
- [ ] [DB-PERF-DATASET-VOLUME] Confirm dataset volume is equal to or larger than the expected first-year production volume, or document acceptance.

## Required EXPLAIN ANALYZE Set

- [ ] [DB-PERF-DOC-LIST-KIND] Capture list by document kind, registration date and created-at ordering.
- [ ] [DB-PERF-DOC-LIST-ACCESS] Capture restricted document list with access-scope `EXISTS` filters.
- [ ] [DB-PERF-DOC-SEARCH] Capture document search/filter with representative text term and date filters.
- [ ] [DB-PERF-ASSIGNMENT-DASHBOARD] Capture active/overdue assignment dashboard query by executor/status/deadline.
- [ ] [DB-PERF-ACKNOWLEDGMENT-PENDING] Capture pending acknowledgment query by user/document/confirmed state.
- [ ] [DB-PERF-JOURNAL-DOCUMENT] Capture document journal by document id and created-at ordering.
- [ ] [DB-PERF-ADMIN-AUDIT-LIST] Capture admin audit list ordered by created-at.
- [ ] [DB-PERF-STATISTICS] Capture statistics/report queries for the heaviest expected date range.

## Decision Rules

- [ ] [DB-PERF-LATENCY-BUDGET] Compare list/search latency with the release budget: main list open/search/filter <= 2 seconds.
- [ ] [DB-PERF-SEQ-SCAN-REVIEW] Review sequential scans only when row counts make them risky; do not reject small-table seq scans by default.
- [ ] [DB-PERF-INDEX-CANDIDATES] If latency or plan evidence fails, document candidate composite/partial/trigram indexes and expected query families.
- [ ] [DB-PERF-BEFORE-AFTER] For every added index, attach before/after `EXPLAIN (ANALYZE, BUFFERS)` and write-latency impact notes.
- [ ] [DB-PERF-NO-INDEX-ACCEPTANCE] If no index is added, record the reason and attach passing plans.

## Evidence

- [ ] Store plans and notes under release evidence.
- [ ] Link accepted missing evidence or failed budgets to `docs/known_issues.md`.
