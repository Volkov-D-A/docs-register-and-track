# Component Structure Review

Дата аудита: 2026-05-28
Этап: D.01

## Вывод

Компонентная структура в документных разделах стала лучше после выделения `DocumentListPage`, `DocumentKindPage`, document-kind forms/filters/columns и `useDocumentListPage`. Это снижает дублирование между входящими, исходящими, обращениями и приказами.

Основной structural debt находится в крупных страницах:

- `frontend/src/pages/SettingsPage.tsx` — 1296 строк;
- `frontend/src/pages/StatisticsPage.tsx` — 622 строки;
- `frontend/src/components/DocumentViewModal.tsx` — 569 строк;
- `frontend/src/pages/AssignmentsPage.tsx` — 466 строк.

## Риски

Крупные страницы смешивают состояние, service calls, формы, таблицы, modal lifecycle и UI layout. Это повышает вероятность регрессий при точечных изменениях, особенно в settings/migrations/users и document view flows.

## Рекомендации

- При следующей функциональной правке разделить `SettingsPage` по вкладкам/feature blocks.
- Вынести service-call orchestration в hooks там, где уже есть repeating CRUD pattern.
- Не делать большой рефактор отдельно от пользовательской задачи; фиксировать поведение тестами/smoke checklist.

Связанные issues: `ISSUE-022`; fixed: `ISSUE-003`.
