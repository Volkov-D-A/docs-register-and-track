# Performance Review

Дата аудита: 2026-05-28
Этап: G.03.204-G.03.210

## Existing Baseline

Database stage B created synthetic dataset:

- 1000 documents;
- 20 users;
- 500 assignments;
- 200 acknowledgments;
- 333 attachment metadata rows;
- 1000 journal rows;
- 500 admin audit rows.

Representative `EXPLAIN ANALYZE` showed fast plans on this dataset, e.g. document list/search and assignment/acknowledgment paths within milliseconds.

## Missing Baseline

No measured baseline was found for:

- Wails app startup to login screen;
- login to dashboard;
- opening main document lists;
- opening document view with files/journal/links;
- saving registration form;
- search/filter latency from UI perspective;
- statistics/report screens;
- frontend render cost of large tables/graphs;
- memory usage under normal work.

## Performance Metrics To Add

- startup to login: target <= 5s;
- main lists open/search/filter: target <= 2s;
- document registration save: target <= 2s typical, no numbering gaps;
- heavy statistics/report: target <= 5s;
- memory normal work: <= 512 MB;
- Wails binary size warning threshold: 100 MB;
- frontend chunk size and load time trend.

Связанные issues: `ISSUE-004`, `ISSUE-009`, `ISSUE-041`.
