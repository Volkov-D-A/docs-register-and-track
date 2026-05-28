# Microcopy Review

Дата аудита: 2026-05-28
Этап: H.01, H.03, H.05

## Destructive Actions

Current examples:

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

## Success Messages

Current generic messages:

- `Удалено`;
- `Статус обновлен`;
- `Файл скачан`;
- `Задача создана`.

Proposed:

- `Поручение удалено`;
- `Статус поручения обновлён`;
- `Файл сохранён в папку "Загрузки"`;
- `Задача на ознакомление создана`.

## Buttons

Recommended verb-first/action-specific labels:

- `Отметить ознакомление` instead of `Отметить`;
- `Отметить исполненным` instead of `Исполнено`;
- `Создать поручение` instead of generic `Создать` where space allows;
- `Вернуть на доработку` instead of `Вернуть` in modal action.

Связанные issues: `ISSUE-046`, `ISSUE-049`, `ISSUE-043`.
