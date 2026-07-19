# PostgreSQL performance evidence

`make db-performance-check` creates a local, disposable PostgreSQL 18 baseline.
It writes the complete benchmark output, including `EXPLAIN (ANALYZE, BUFFERS,
FORMAT JSON)` and connection-pool state, to `build/performance/db-performance.txt`.
It then prints a compact median/p95/plan/buffer summary and stores a JSON
snapshot in `build/performance/latest.json` plus a timestamped local history.
The summary compares median latency with the preceding local snapshot.
The detailed Go benchmark and plan log is retained in the artifact file rather
than repeated on the console; it is printed only if the benchmark fails.

The command is intentionally not part of `release-gate` and has no latency
threshold: Docker, WSL and developer hardware are not stable benchmark hosts.
Run it before and after changes to repository SQL, indexes, pagination or
access predicates. Compare planning/execution time, actual rows, shared-buffer
activity, allocations and benchmark throughput; copy a reviewed before/after
summary below when a change is accepted.

| Date | Change | Dataset / workload | Before | After | Notes |
| --- | --- | --- | --- | --- | --- |
| — | Initial harness | 250 document list; 500 document monthly statistics | Run locally | Run locally | No baseline committed yet |

The current harness is deliberately small and deterministic. Add production-like
sizes only together with a documented workload and a reason to keep its runtime.
