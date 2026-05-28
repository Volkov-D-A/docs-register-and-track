# Known Issues

Дата: 2026-05-28
Статус: release-blocking list for current production candidate

## Critical Blockers

| Issue | Status | Why it blocks release | Required action |
| --- | --- | --- | --- |
| `ISSUE-032` | open | `govulncheck` reports reachable vulnerabilities in Go `1.26.2` and `golang.org/x/net v0.52.0`. | Upgrade Go to `1.26.3+`, `x/net` to `v0.53.0+`, repeat `govulncheck`. |
| `ISSUE-050` | open | Final production candidate is missing release-grade README/runbooks for build, migrations, backup/restore and diagnostics. | Add and validate root release/ops documentation. |
| `ISSUE-052` | open | Release evidence is not clean/reproducible: worktree has many uncommitted audit/remediation changes. | Review, test, commit/tag or exclude non-release changes; release from clean status. |

## Fixed Critical Issues

| Issue | Status | Evidence |
| --- | --- | --- |
| `ISSUE-007` | fixed | Runtime rollback now requires backend-enforced backup confirmation, backup reference, data-loss acknowledgment and control phrase; frontend uses the same guardrails; audit entries include backup reference; `docs/migration_rollback_runbook.md`, `go test ./...` and `npm run build` completed. |

## Major Issues To Fix Or Explicitly Accept

| Area | Issues |
| --- | --- |
| Production config and secrets | `ISSUE-001`, `ISSUE-002`, `ISSUE-025`, `ISSUE-028`, `ISSUE-029` |
| Backup/restore and filesystem safety | `ISSUE-010`, `ISSUE-030`, `ISSUE-031` |
| Backend lifecycle/logging | `ISSUE-015`, `ISSUE-016` |
| Frontend safety and errors | `ISSUE-019`, `ISSUE-020`, `ISSUE-021`, `ISSUE-044`, `ISSUE-046` |
| Build/install/update | `ISSUE-023`, `ISSUE-024`, `ISSUE-026`, `ISSUE-027` |
| Security/license/static analysis | `ISSUE-033`, `ISSUE-034`, `ISSUE-037` |
| Test/performance evidence | `ISSUE-038`, `ISSUE-039`, `ISSUE-040`, `ISSUE-041`, `ISSUE-042`, `ISSUE-043` |
| UX terminology | `ISSUE-045` |
| Documentation/release readiness | `ISSUE-051`, `ISSUE-053` |

## Postponable Minor Issues

`ISSUE-003`, `ISSUE-004`, `ISSUE-005`, `ISSUE-009`, `ISSUE-011`, `ISSUE-017`, `ISSUE-018`, `ISSUE-022`, `ISSUE-035`, `ISSUE-036`, `ISSUE-047`, `ISSUE-048`, `ISSUE-049`.

These are not production blockers by themselves, but several are good follow-ups after critical/major remediation.

## Already Fixed During Audit

`ISSUE-006`, `ISSUE-008`, `ISSUE-012`, `ISSUE-013`, `ISSUE-014`.

Key fixed areas: no-gaps/idempotent document registration, retention-safe journal/audit FK strategy, structured backend/Wails error envelope, strict document command decoding.
