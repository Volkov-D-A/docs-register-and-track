# Known Issues

Дата: 2026-06-02
Статус: release-blocking list for current production candidate

## Critical Blockers

None currently tracked. Release tag, clean-clone release gate and artifact evidence remain required release checklist steps.

## Fixed Critical Issues

| Issue | Status | Evidence |
| --- | --- | --- |
| `ISSUE-007` | fixed | Runtime rollback now requires backend-enforced backup confirmation, backup reference, data-loss acknowledgment and control phrase; frontend uses the same guardrails; audit entries include backup reference; `docs/migration_rollback_runbook.md`, `go test ./...` and `npm run build` completed. |
| `ISSUE-032` | fixed | `go.mod` requires Go `1.26.3`; `golang.org/x/net` upgraded to `v0.53.0`; `govulncheck ./...` reports 0 reachable vulnerabilities. |
| `ISSUE-050` | fixed | Root `README.md` now links maintained release, rollback, backup/restore and diagnostics runbooks; `docs/release_runbook.md` and `docs/diagnostics_runbook.md` were added. |
| `ISSUE-052` | fixed | Audit/remediation changes committed in `1592e9b`; `git status --short` is clean after the commit. |

## Major Issues To Fix Or Explicitly Accept

| Area | Issues |
| --- | --- |
| Production config and secrets | none |
| Backup/restore and filesystem safety | none |
| Backend lifecycle/logging | `ISSUE-015` |
| Frontend safety and errors | none |
| Build/install/update | none |
| Security/license/static analysis | none |
| Test/performance evidence | `ISSUE-041`, `ISSUE-042`, `ISSUE-043` |
| UX terminology | none |
| Documentation/release readiness | none |

## Postponable Minor Issues

`ISSUE-009`, `ISSUE-022`.

These are not production blockers by themselves, but several are good follow-ups after critical/major remediation.

## Already Fixed During Audit

`ISSUE-001`, `ISSUE-002`, `ISSUE-003`, `ISSUE-004`, `ISSUE-005`, `ISSUE-006`, `ISSUE-008`, `ISSUE-010`, `ISSUE-012`, `ISSUE-013`, `ISSUE-014`, `ISSUE-016`, `ISSUE-017`, `ISSUE-018`, `ISSUE-019`, `ISSUE-020`, `ISSUE-021`, `ISSUE-023`, `ISSUE-024`, `ISSUE-027`, `ISSUE-028`, `ISSUE-030`, `ISSUE-031`, `ISSUE-032`, `ISSUE-033`, `ISSUE-034`, `ISSUE-035`, `ISSUE-036`, `ISSUE-038`, `ISSUE-039`, `ISSUE-040`, `ISSUE-044`, `ISSUE-045`, `ISSUE-050`, `ISSUE-051`, `ISSUE-052`, `ISSUE-053`.

Key fixed areas: no-gaps/idempotent document registration, retention-safe journal/audit FK strategy, structured backend/Wails/frontend error handling, strict document command decoding and consistent backend `NOT_FOUND` responses.
