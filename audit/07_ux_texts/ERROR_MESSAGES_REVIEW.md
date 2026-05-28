# Error Messages Review

Дата аудита: 2026-05-28
Этап: H.03

## Вывод

Error copy remains the most important UX text risk. Many handlers still render `err?.message || String(err)`. This overlaps with `ISSUE-019`, but H adds the UX requirement: messages should say what happened, what the user can do, and avoid internal jargon/IDs.

## Examples

| Current | Problem | Proposed Direction |
| --- | --- | --- |
| `Ошибка загрузки документа: ${err}` | Can include technical text; closes modal. | `Не удалось открыть документ. Проверьте доступ и попробуйте снова.` |
| `Ошибка загрузки журнала: ${err}` | Technical raw suffix. | `Не удалось загрузить журнал документа. Попробуйте обновить карточку.` |
| `Ошибка миграции (dirty)` | English technical state. | `Миграция завершилась с ошибкой. Работа со схемой БД требует восстановления по инструкции.` |
| `Не удалось получить статус` | No next step. | `Не удалось получить статус миграций. Проверьте подключение к БД и повторите запрос.` |
| `err?.message || String(err)` | Leaks backend/internal messages. | Map structured `code` to safe copy and action. |

## Error Copy Rules

- Avoid raw DB/storage/Go text in user UI.
- Do not show internal UUIDs unless operator explicitly needs them.
- Include next step for recoverable cases.
- Use neutral tone: "Не удалось..." instead of blame.
- Keep details in technical logs and audit trail.

Связанные issues: `ISSUE-019`, `ISSUE-044`, `ISSUE-028`.
