# Known Issues For Release Candidate

Дата обновления: 2026-06-02
Статус: maintained release-facing list

This document summarizes open blockers and accepted follow-up work for the current production candidate. It must match `audit/REVIEW_LOG.md` before a release tag is created.

## Critical Blockers

| Issue | Owner | Mitigation / required action |
| --- | --- | --- |
| `ISSUE-052` Release reproducibility | Release owner | Build only from a clean committed/tagged worktree. Attach clean `git status --short`, release gate output summaries and artifact checksums to release evidence. |

## Major Issues Requiring Fix Or Explicit Acceptance

| Issue | Area | Owner | Mitigation / required action |
| --- | --- | --- | --- |
| `ISSUE-015` | Backend lifecycle | Backend owner | Add app/request context propagation and timeout/cancel policy for MinIO, journal, link graph and statistics operations. |
| `ISSUE-021` | Frontend UX | Frontend owner | Add dirty-state confirmations for long registration/edit and important settings forms. |
| `ISSUE-028` | Startup diagnostics | Backend/release owner | Replace startup fatal exits with actionable diagnostics and verify unavailable DB/MinIO/Seq cases. |
| `ISSUE-038` | Frontend/e2e tests | QA owner | Add or manually execute e2e smoke for login, document flows, permissions and release-critical UX. |
| `ISSUE-039` | Integration safety | QA/backend owner | Gate integration tests on disposable services only and document required DSN/environment. |
| `ISSUE-040` | Database constraints | Backend owner | Add release-gated database constraint checks for registration, retention and migration safety. |
| `ISSUE-041` | Performance baseline | QA owner | Record startup, list, dashboard/statistics and memory baselines on production-like data. |
| `ISSUE-042` | Cancellation tests | QA/backend owner | Test long-running upload/download/list/statistics/link operations and shutdown cancellation. |
| `ISSUE-043` | UX safety tests | QA/frontend owner | Smoke destructive confirmations, dirty confirmations, empty states and error messages. |
| `ISSUE-045` | UX terminology | Product/frontend owner | Confirm and apply glossary terminology for document type, nomenclature/case, executor roles and content fields. |

## Postponable Minor Issues

| Issue | Area | Acceptance note |
| --- | --- | --- |
| `ISSUE-009` | Database performance | Current baseline plans are fast for expected initial volume; repeat EXPLAIN on production-like data before adding indexes. |
| `ISSUE-022` | Frontend structure | Large components are accepted short-term; decompose gradually during feature work with smoke coverage. |
