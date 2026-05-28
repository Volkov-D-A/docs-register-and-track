# Security Review

Дата аудита: 2026-05-28
Этап: F.01, F.02. Security and vulnerability checks

## Общий Вывод

Фактические проверки:

- `npm audit --audit-level=critical --json`: 0 vulnerabilities.
- `govulncheck ./...`: found 2 reachable vulnerabilities affecting the current build/runtime.
- `go test ./...`: passed.
- `go vet ./...`: passed.
- `npm run build`: TypeScript and Vite production build passed.

Главный blocker этапа F: текущая локальная сборочная среда использует `go1.26.2`, а `govulncheck` нашел достижимые уязвимости, исправленные в `go1.26.3`, плюс `golang.org/x/net@v0.52.0` нужно поднять до `v0.53.0`.

## Контрольные Пункты

| Код | Статус | Severity | Area | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- |
| F.01.181 | issue | critical | Go stdlib / Go modules | `govulncheck` нашел `GO-2026-4971` in `net@go1.26.2`, fixed in `go1.26.3`; `GO-2026-4918` in `net/http@go1.26.2` and `golang.org/x/net@v0.52.0`, fixed in `go1.26.3` / `x/net@v0.53.0`. | Перед production обновить Go toolchain and `golang.org/x/net`, затем повторить `govulncheck`. |
| F.01.182 | ok | none | npm | `npm audit`: total 0 vulnerabilities across 224 dependencies. | npm advisory scan clean на момент проверки. |
| F.01.183 | ok | none | lock files | `go.sum` и `frontend/package-lock.json` присутствуют. | Lock files зафиксированы. |
| F.01.184 | issue | minor | dependency policy | `package.json` использует caret ranges; Go modules exact in `go.mod`; release build uses `npm install` in `wails.json`. | Lock file mitigates npm drift, но release gate должен использовать `npm ci` and dependency update policy. |

## Security Risks Already Tracked

- `ISSUE-016`: logging PII minimization.
- `ISSUE-024`: release build secret injection.
- `ISSUE-029`: production secret policy.
- `ISSUE-032`: Go reachable vulnerabilities.

## Required Before Production

- Upgrade Go toolchain to `go1.26.3` or newer compatible approved version.
- Upgrade `golang.org/x/net` to at least `v0.53.0`.
- Repeat `govulncheck ./...`, `go test ./...`, `go vet ./...`, `npm run build`.
