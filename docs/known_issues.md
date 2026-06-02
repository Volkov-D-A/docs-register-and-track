# Known Issues For Release Candidate

Дата обновления: 2026-06-02
Статус: maintained release-facing list

This document summarizes open blockers and accepted follow-up work for the current production candidate. It must match `audit/REVIEW_LOG.md` before a release tag is created.

## Critical Blockers

None currently tracked. Release must still be built from a clean committed/tagged checkout with release gate output summaries and artifact checksums attached as evidence.

## Major Issues Requiring Fix Or Explicit Acceptance

| Issue | Area | Owner | Mitigation / required action |
| --- | --- | --- | --- |
| `ISSUE-015` | Backend lifecycle | Backend owner | Add app/request context propagation and timeout/cancel policy for MinIO, journal, link graph and statistics operations. |
| `ISSUE-041` | Performance baseline | QA owner | Record startup, list, dashboard/statistics and memory baselines on production-like data. |
| `ISSUE-042` | Cancellation tests | QA/backend owner | Test long-running upload/download/list/statistics/link operations and shutdown cancellation. |
| `ISSUE-043` | UX safety tests | QA/frontend owner | Smoke destructive confirmations, dirty confirmations, empty states and error messages. |

## Postponable Minor Issues

| Issue | Area | Acceptance note |
| --- | --- | --- |
| `ISSUE-009` | Database performance | Current baseline plans are fast for expected initial volume; repeat EXPLAIN on production-like data before adding indexes. |
| `ISSUE-022` | Frontend structure | Large components are accepted short-term; decompose gradually during feature work with smoke coverage. |
