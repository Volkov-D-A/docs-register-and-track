# Microcopy Review

Дата аудита: 2026-05-28
Этап: H.01, H.03, H.05

## Destructive Actions

Original examples before remediation:

- `Удалить?`
- `Удалить файл?`
- `Откатить последнюю`
- `Массовое удаление файлов`

Recommendations:

| Current | Proposed |
| --- | --- |
| `Удалить?` | `Удалить поручение? Это действие нельзя отменить.` |
| `Удалить файл?` | `Удалить файл "{filename}"? Это действие нельзя отменить.` |
| `Откатить последнюю` | `Откатить последнюю миграцию` |
| Rollback confirmation | Add: `Перед откатом должен быть свежий backup PostgreSQL и MinIO.` |
| Bulk delete confirmation | Keep strong warning, include count preview if available. |

Current status: destructive confirmation wording is fixed for file delete, document link delete, assignment delete, acknowledgment delete, reference deletes, migration rollback and bulk file delete. Generic success-message/style consistency is fixed under `ISSUE-049`; smoke coverage is maintained in `docs/ux_safety_smoke.md` after `ISSUE-043`.

## Success Messages

Original generic messages before remediation:

- `Удалено`;
- `Статус обновлен`;
- `Файл скачан`;
- `Задача создана`.

Proposed:

- `Поручение удалено`;
- `Статус поручения обновлён`;
- `Файл сохранён в папку "Загрузки"`;
- `Задача на ознакомление создана`.

Current status: fixed in `ISSUE-049` for the audited examples and nearby flows. Success messages now name the affected entity/action, visible `ё` usage is normalized for updated/deleted/finished strings touched by the remediation, `dirty` is no longer shown to users, and statistics uses `Нет данных` instead of `N/A`.

## Buttons

Recommended verb-first/action-specific labels:

- `Отметить ознакомление` instead of `Отметить`;
- `Отметить исполненным` instead of `Исполнено`;
- `Создать поручение` instead of generic `Создать` where space allows;
- `Вернуть на доработку` instead of `Вернуть` in modal action.

Связанные issues: fixed `ISSUE-046`, `ISSUE-049`, `ISSUE-043`.
