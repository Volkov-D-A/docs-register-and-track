# Security Review

Дата аудита: 2026-05-28
Этап: F.01, F.02. Security and vulnerability checks

## Общий Вывод

Фактические проверки:

- `npm audit --audit-level=critical --json`: 0 vulnerabilities.
- After remediation `ISSUE-032`, `govulncheck ./...`: 0 reachable vulnerabilities.
- `go test ./...`: passed.
- `go vet ./...`: passed.
- `npm run build`: TypeScript and Vite production build passed.

Главный blocker этапа F был закрыт после audit: `go.mod` requires `go1.26.3`, `golang.org/x/net@v0.53.0`; `govulncheck` clean.

## Контрольные Пункты

| Код | Статус | Severity | Area | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- |
| F.01.181 | ok | none | Go stdlib / Go modules | `go1.26.3`, `golang.org/x/net@v0.53.0`; `govulncheck` reports 0 reachable vulnerabilities. | Keep `govulncheck` in release gate. |
| F.01.182 | ok | none | npm | `npm audit`: total 0 vulnerabilities across 224 dependencies. | npm advisory scan clean на момент проверки. |
| F.01.183 | ok | none | lock files | `go.sum` и `frontend/package-lock.json` присутствуют. | Lock files зафиксированы. |
| F.01.184 | issue | minor | dependency policy | `package.json` использует caret ranges; Go modules exact in `go.mod`; release build uses `npm install` in `wails.json`. | Lock file mitigates npm drift, но release gate должен использовать `npm ci` and dependency update policy. |

## Security Risks Already Tracked

- `ISSUE-016`: logging PII minimization.
- `ISSUE-024`: release build secret injection.
- `ISSUE-029`: production secret policy.
- `ISSUE-032`: Go reachable vulnerabilities fixed after remediation.

## Required Before Production

- Keep Go toolchain at `go1.26.3` or newer compatible approved version.
- Keep `golang.org/x/net` at least `v0.53.0`.
- Repeat `govulncheck ./...`, `go test ./...`, `go vet ./...`, `npm run build` from clean checkout.
