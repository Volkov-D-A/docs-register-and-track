# Dependencies Review

Дата аудита: 2026-05-28
Этап: F.01

## Go Dependencies

`go.mod` uses exact module versions. Important runtime dependencies include Wails, PostgreSQL driver, MinIO client, golang-migrate, `golang.org/x/crypto`, `golang.org/x/net`.

Findings:

- After remediation `ISSUE-032`, `govulncheck` reports 0 reachable vulnerabilities with `go1.26.3` and `x/net@v0.53.0`.
- `go list -m -json all` without network failed in sandbox because module metadata lookup tried to reach `proxy.golang.org`; this is not a code failure, but release dependency checks need network/proxy access.
- `go test ./...` and `go vet ./...` passed locally.

## npm Dependencies

Direct dependencies:

- React 18.3.1, React DOM 18.3.1.
- Ant Design 6.3.5, icons 6.1.1, plots 2.6.8.
- Vite 8.0.8, TypeScript 6.0.2.
- Zustand 5.0.12, dayjs 1.11.20, `@xyflow/react` 12.10.2.

Findings:

- `package-lock.json` exists and `npm audit` returned 0 vulnerabilities.
- `package.json` direct versions use caret ranges, acceptable only with lockfile and `npm ci` in release builds.
- `frontend` package version is `0.0.0`; application version is handled elsewhere and already tracked by `ISSUE-023`.

## Recommendations

- Use maintained `make release-gate`, which runs `govulncheck`, `npm audit --audit-level=critical`, `go test`, `go vet`, `npm run build`, npm license check and dependency inventory generation.
- Use `npm ci` in release build path.
- Define dependency update cadence and emergency security patch process.

Связанные issues: fixed `ISSUE-032`, `ISSUE-033`; open `ISSUE-024`.
