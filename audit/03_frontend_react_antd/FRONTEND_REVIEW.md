# Frontend Review

Дата аудита: 2026-05-28
Этап: D. React, Ant Design, состояние, формы, таблицы, доступность, производительность

## Общий Вывод

Frontend построен предсказуемо: страницы React вызывают generated Wails services, UI собирается на Ant Design, краткоживущее состояние вынесено в Zustand hooks/stores, документные списки используют общий `useDocumentListPage` и reusable page/table shell. Прямого доступа frontend к PostgreSQL/MinIO не найдено.

После исправлений этапа C формы регистрации всех 4 видов документов отправляют `idempotencyKey`, а backend защищает повторную регистрацию транзакционно. После исправления `ISSUE-019` frontend читает Wails structured error envelope через единый adapter `formatAppError`/`normalizeAppError`, а auth lockout распознается по code `USER_LOCKED`. После исправления `ISSUE-020` критичные submit/actions имеют локальные loading guards.

Главные production-риски этапа D:

- закрытие регистрационных/редактирующих модалок не защищено dirty/unsaved guard;
- часть страниц стала крупной и смешивает табличный UI, формы, service calls и состояние;
- production build уже дает Vite warning о большом основном чанке, а route-level lazy loading отсутствует.

## Critical/Major Issues

| Issue | Severity | Кратко | Ответственный этап |
| --- | --- | --- | --- |
| ISSUE-019 | fixed | Frontend uses `formatAppError`/`normalizeAppError` for Wails `{code,message,status}` and auth lockout uses code `USER_LOCKED`. | D/F/H |
| ISSUE-020 | fixed | Critical document, assignment, file, settings, migration and storage actions have local submitting/loading guards. | D/F |
| ISSUE-021 | major | Формы в модалках можно закрыть без предупреждения о несохраненных изменениях. | D/H |
| ISSUE-022 | minor | Крупные page-компоненты смешивают UI, состояние и интеграции, повышая риск регрессий. | D/F |
| ISSUE-004 | fixed | Main frontend chunk split by lazy page loading; remaining largest lazy chunk is StatisticsPage and is covered by explicit Wails desktop budget/performance baseline. | D/F |

## Контрольные Пункты D

| Код | Статус | Severity | Место | Доказательство | Вывод / рекомендация |
| --- | --- | --- | --- | --- | --- |
| D.01.101 | ok | none | `frontend/src/App.tsx`, pages/components | App переключает страницы, MainLayout держит shell, document pages используют общий `DocumentKindPage`. | Базовая структура понятна. |
| D.01.102 | issue | minor | `SettingsPage.tsx`, `StatisticsPage.tsx`, `AssignmentsPage.tsx`, `DocumentViewModal.tsx` | Размеры: 1296, 622, 466, 569 строк. | Разделить на feature hooks/subcomponents при следующем изменении этих зон. |
| D.01.103 | ok | none | `frontend/src/modules/documentKinds/*` | Document-kind forms/filters/columns частично вынесены в modules. | Хорошее направление; распространить на settings/statistics. |
| D.02.108 | ok | none | `frontend/src/utils/appError.ts`, catch handlers, `useAuthStore` | Raw `err?.message || String(err)` frontend patterns removed; auth lockout reads stable `USER_LOCKED` code. | Keep new catch handlers on `formatAppError`; verify end-to-end in smoke/e2e work. |
| D.02.110 | ok | none | document modals, settings/actions | Incoming/outgoing/orders/citizen appeal modals use local submit guards; assignment/acknowledgment/file/settings/migration/storage actions block repeated clicks. | Keep double-click smoke in release checklist. |
| D.02.113 | issue | major | `DocumentKindPage` modals | `onCancel` закрывает модалки сразу; dirty state формы не проверяется. | Добавить unsaved changes confirmation для registration/edit и важных settings forms. |
| D.03.116 | ok | none | Zustand stores | `useAuthStore`, `useDraftLinkStore`, `useRegisterDocumentStore` содержат ограниченное UI/session state. | Глобальное состояние не выглядит избыточным. |
| D.03.121 | ok | none | `resolveUserProfile` | Backend dashboard no longer duplicates UX profile labels; frontend profile classification remains UI-only. | Keep access summary/permissions as source of access truth; do not use UX profile labels for authorization. |
| D.04.123 | ok | none | document forms | AntD `Form` validation закрывает основные required/date/number поля; `InputNumber min` используется для листов. | Базовая client validation есть, backend остается источником истины. |
| D.04.126 | issue | minor | forms/rules | Часть required rules без `whitespace` и без явных сообщений; null/empty-string policy неодинакова. | Унифицировать form rules и payload normalization вместе с typed DTO contract. |
| D.05.134 | ok | none | notifications/forms | UI uses structured `code/message/status` adapter with safe code-based defaults for validation/forbidden/not_found/conflict/internal. Backend field-level `details` are not currently part of the production envelope. | Add field-level mapping only if backend starts returning validation details. |
| D.06.142 | issue | minor | modals/graph/file flows | Нет явной focus recovery после ошибок/закрытия некоторых modal flows; graph interaction в основном pointer-oriented. | Проверить keyboard/screen-reader smoke, особенно модалки и link graph. |
| D.07.149 | ok | none | Vite build | Route-level lazy loading reduced main app chunk to ~34 kB; `npm run build` passes without Vite chunk warning under explicit 1600 kB Wails desktop route-chunk budget. | Performance baseline still needs target Wails startup/memory measurements. |

## Что Передать На F/H

- Проверить frontend error contract end-to-end in release smoke/e2e: login, forbidden, validation, not found, duplicate/idempotency conflict, internal error.
- UI smoke для повторного submit: регистрация, редактирование, settings CRUD, миграции, поручения, файлы.
- Accessibility smoke: keyboard navigation, focus after modal close/error, AntD table/filter controls, link graph fallback.
- Performance smoke: старт до login, открытие списков, statistics, размер bundle и memory на production-like данных.

## Closure

Статус этапа D: completed with open frontend remediation issues.

Все обязательные артефакты D созданы: `FRONTEND_REVIEW.md`, `COMPONENT_STRUCTURE_REVIEW.md`, `STATE_MANAGEMENT_REVIEW.md`, `ANTD_FORMS_REVIEW.md`, `TABLES_FILTERS_PAGINATION_REVIEW.md`, `ACCESSIBILITY_REVIEW.md`, `FRONTEND_PERFORMANCE_REVIEW.md`.
