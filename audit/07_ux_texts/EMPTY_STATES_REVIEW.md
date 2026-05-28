# Empty States Review

Дата аудита: 2026-05-28
Этап: H.03.238

## Вывод

Empty states exist, but are mostly passive:

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

Risk: low for wording, medium for action-specific empty states because available actions depend on permissions.

Связанные issues: `ISSUE-047`.
