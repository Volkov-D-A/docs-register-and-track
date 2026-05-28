# Frontend Tests Review

Дата аудита: 2026-05-28
Этап: G.02.196-G.02.199

## Вывод

Frontend has TypeScript build coverage through `npm run build`, but no unit/component test framework was found:

- no Vitest/Jest config;
- no React Testing Library tests;
- no frontend `test` script;
- no component specs for document forms, errors, empty states or navigation.

## Missing Coverage

- Registration forms for incoming/outgoing/citizen appeals/orders.
- Validation and backend error display.
- Empty table/list states.
- Access-based navigation and hidden actions.
- Submit loading/dirty confirmation after planned remediation.
- Structured error code handling after `DECISION-009`.

## Рекомендации

- Add small Vitest/React Testing Library setup.
- Start with pure helpers and high-risk UI: error adapter, document form payload builders, access navigation.
- Keep e2e for full Wails flows; do not over-mock Wails internals in component tests.

Связанные issues: `ISSUE-038`, `ISSUE-019`, `ISSUE-020`, `ISSUE-021`.
