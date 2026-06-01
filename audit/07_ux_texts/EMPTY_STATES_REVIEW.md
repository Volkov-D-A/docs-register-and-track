# Empty States Review

Дата аудита: 2026-05-28
Этап: H.03.238

## Вывод

Original audit finding: empty states existed, but were mostly passive:

- `Нет прикрепленных файлов`;
- chart cards use AntD `Empty` without custom description;
- `Нет доступа к статистике`;
- tables often rely on default AntD empty state.

## Recommendations

| Component | Current | Proposed |
| --- | --- | --- |
| Files | `Нет прикрепленных файлов` | `Файлы не прикреплены. Нажмите "Загрузить файл", чтобы добавить вложение.` |
| Statistics charts | default Empty | `Нет данных за выбранный период. Измените фильтры или период отчёта.` |
| No statistics access | `Нет доступа к статистике` | `Нет доступа к статистике. Обратитесь к администратору, если доступ нужен для работы.` |
| Document lists | default table empty | `Документы не найдены. Измените фильтры или зарегистрируйте новый документ.` |
| Assignments | default table empty | `Поручения не найдены. Измените фильтры или создайте поручение из карточки документа.` |

## Current Status

`ISSUE-047` is fixed for the currently identified empty states: dashboard cards, file attachments, document links, statistics charts, statistics access denial and statistics report/rating tables now include a next step based on permissions, filters or access.

Remaining validation belongs to frontend/e2e smoke coverage under `ISSUE-043`; broader table/list empty states can be revisited with the table component test pass.

Связанные issues: fixed `ISSUE-047`; open `ISSUE-043`.
