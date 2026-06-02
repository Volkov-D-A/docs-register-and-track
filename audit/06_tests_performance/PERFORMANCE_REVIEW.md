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

## Baseline Evidence After Remediation

`ISSUE-041` fixed after audit by adding `make performance-baseline` and `tools/performance-report.js`.
The generated report is written to `build/release-evidence/PERFORMANCE_BASELINE.md`.
It records automated static metrics and a required target OS manual timings table.

Current generated static baseline:

- frontend dist total: 3.05 MiB;
- largest JS asset: `StatisticsPage-4iz6lzg0.js`, 1.40 MiB, below 1.6 MiB route chunk budget;
- Linux binary: 17.48 MiB, below 100 MiB warning threshold;
- Windows binary: 18.66 MiB, below 100 MiB warning threshold.

Target OS manual timing rows remain required release evidence for startup, login/dashboard, lists/search, registration save, statistics and memory.

## Performance Metrics To Add

- startup to login: target <= 5s;
- main lists open/search/filter: target <= 2s;
- document registration save: target <= 2s typical, no numbering gaps;
- heavy statistics/report: target <= 5s;
- memory normal work: <= 512 MB;
- Wails binary size warning threshold: 100 MB;
- frontend chunk size and load time trend.

Связанные issues: fixed `ISSUE-004`, `ISSUE-009`, `ISSUE-041`.
