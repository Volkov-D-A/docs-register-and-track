# Terminology Review

Дата аудита: 2026-05-28
Этап: H.02, H.04

## Terminology Decisions Proposed

| Entity | Preferred UI Term | Notes |
| --- | --- | --- |
| incoming/outgoing/appeal/order | `Вид документа` | Registry/document family. |
| letter/contract/act/etc. | `Тип документа` | Value selected in document form. |
| nomenclature item | `Дело` | User-facing forms and filters. |
| nomenclature admin area | `Номенклатура дел` | Admin/settings context. |
| assignment executor | `Ответственный исполнитель` | Pоручения. |
| outgoing letter executor | `Исполнитель письма` | Avoid conflict with assignment executor. |
| resolution executor | `Исполнитель резолюции` | Keep current, but use consistently. |
| content | `Содержание` | Avoid `Краткое содержание` unless field is truly short summary. |
| inactive order | `Недействующий` | One word. |
| overdue | `Просроченные` | Avoid `Проср.` in controls. |

## Style Rules

- Use `ё` consistently in visible UI where words are standard: `обновлён`, `завершён`, `удалён`.
- Avoid English technical words: `dirty`, `N/A`.
- Dates in UI: `DD.MM.YYYY`; date-time: `DD.MM.YYYY HH:mm`.
- Avoid internal IDs in user-facing messages. IDs may remain in admin audit details only if explicitly considered technical/audit context.

## Needs Business Agreement

- Whether public UI should say `Дело` or official `Номенклатура`.
- Business confirmation for broader terminology; `ПОС` expansion is fixed in UI by `ISSUE-048`.
- Whether `Краткое содержание` is still a valid business field after backend/frontend consolidation to `content`.

Связанные issues: open `ISSUE-045`; fixed `ISSUE-048`.
