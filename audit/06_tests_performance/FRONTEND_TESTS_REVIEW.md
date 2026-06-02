# Frontend Tests Review

Дата аудита: 2026-05-28
Этап: G.02.196-G.02.199

## Вывод

Frontend has TypeScript build coverage through `npm run build`. After `ISSUE-038`, it also has a dependency-free TypeScript + Node test layer:

- `npm test` compiles focused frontend tests and runs them through Node's built-in test runner;
- `appError` safe copy and serialized Wails envelope handling are covered;
- `dirtyForm` touched-state and discard confirmation behavior are covered;
- `npm run smoke:prod` verifies production `dist/index.html` and referenced JS/CSS assets.

## Missing Coverage

- Full registration form component tests for incoming/outgoing/citizen appeals/orders.
- Empty table/list component tests.
- Access-based navigation and hidden actions.
- Browser/Wails lifecycle smoke for login, navigation and document flows.

## Рекомендации

- Keep dependency-free helper tests in release gate.
- Add browser/jsdom component tests only when the dependency/tooling tradeoff is accepted.
- Keep full Wails flows in manual/target OS release smoke through `docs/ux_safety_smoke.md`; do not over-mock Wails internals in component tests.

Связанные issues: fixed after audit: `ISSUE-019`, `ISSUE-020`, `ISSUE-021`, `ISSUE-038`, `ISSUE-043`.
