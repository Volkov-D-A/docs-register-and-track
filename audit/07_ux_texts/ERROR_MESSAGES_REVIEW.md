# Error Messages Review

Дата аудита: 2026-05-28
Этап: H.03

## Вывод

User-facing error copy is centralized in the shared frontend `appError` adapter. Structured backend/Wails error codes now map to safe UX copy that says what happened and gives the next step; unstructured raw `Error.message` / string errors are not displayed directly. Manual smoke for full failure paths remains tracked by `ISSUE-043`.

## Examples

| Current | Problem | Proposed Direction |
| --- | --- | --- |
| `Ошибка загрузки документа: ${err}` | Can include technical text; closes modal. | `Не удалось открыть документ. Проверьте доступ и попробуйте снова.` |
| `Ошибка загрузки журнала: ${err}` | Technical raw suffix. | `Не удалось загрузить журнал документа. Попробуйте обновить карточку.` |
| `Ошибка миграции (dirty)` | English technical state. | `Миграция завершилась с ошибкой. Работа со схемой БД требует восстановления по инструкции.` |
| `Не удалось получить статус` | No next step. | `Не удалось получить статус миграций. Проверьте подключение к БД и повторите запрос.` |
| `err?.message || String(err)` | Leaks backend/internal messages. | Fixed in shared `appError` adapter: structured `code` maps to safe copy and action; unknown/raw errors use generic recovery text. |

## Error Copy Rules

- Avoid raw DB/storage/Go text in user UI.
- Do not show internal UUIDs unless operator explicitly needs them.
- Include next step for recoverable cases.
- Use neutral tone: "Не удалось..." instead of blame.
- Keep details in technical logs and audit trail.

Связанные issues: `ISSUE-019`, `ISSUE-043`, `ISSUE-028`.
