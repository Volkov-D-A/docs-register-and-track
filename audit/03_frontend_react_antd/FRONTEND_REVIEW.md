# Frontend Review

Дата аудита: 2026-05-28
Этап: D. React, Ant Design, состояние, формы, таблицы, доступность, производительность

## Общий Вывод

Frontend построен предсказуемо: страницы React вызывают generated Wails services, UI собирается на Ant Design, краткоживущее состояние вынесено в Zustand hooks/stores, документные списки используют общий `useDocumentListPage` и reusable page/table shell. Прямого доступа frontend к PostgreSQL/MinIO не найдено.

После исправлений этапа C формы регистрации всех 4 видов документов отправляют `idempotencyKey`, а backend защищает повторную регистрацию транзакционно. Но frontend еще не завершил переход на новый structured error contract: ошибки почти везде показываются как `err?.message || String(err)`, а auth flow продолжает распознавать lockout по тексту. Это оставляет часть `ISSUE-012` на стороне D как отдельную frontend-задачу.

Главные production-риски этапа D:

- отсутствует единый frontend error adapter для Wails `{code,message,status}`, поэтому UI не умеет стабильно различать validation/forbidden/not_found/conflict/internal errors;
- в части модалок регистрации/редактирования нет собственного submit/loading guard: document idempotency снижает риск дублей, но UX и недокументные операции остаются чувствительны к повторным действиям;
- закрытие регистрационных/редактирующих модалок не защищено dirty/unsaved guard;
- часть страниц стала крупной и смешивает табличный UI, формы, service calls и состояние;
- production build уже дает Vite warning о большом основном чанке, а route-level lazy loading отсутствует.

## Critical/Major Issues

| Issue | Severity | Кратко | Ответственный этап |
| --- | --- | --- | --- |
| ISSUE-019 | major | Frontend не использует structured backend error codes и продолжает показывать/разбирать raw error strings. | D/F/H |
| ISSUE-020 | major | Не все destructive/submit actions имеют локальный submitting/loading guard от повторного клика. | D/F |
| ISSUE-021 | major | Формы в модалках можно закрыть без предупреждения о несохраненных изменениях. | D/H |
| ISSUE-022 | minor | Крупные page-компоненты смешивают UI, состояние и интеграции, повышая риск регрессий. | D/F |
| ISSUE-004 | minor | Большой основной frontend chunk остается открытым performance risk. | D/F |

## Контрольные Пункты D

| Код | Статус | Severity | Место | Доказательство | Вывод / рекомендация |
| --- | --- | --- | --- | --- | --- |
| D.01.101 | ok | none | `frontend/src/App.tsx`, pages/components | App переключает страницы, MainLayout держит shell, document pages используют общий `DocumentKindPage`. | Базовая структура понятна. |
| D.01.102 | issue | minor | `SettingsPage.tsx`, `StatisticsPage.tsx`, `AssignmentsPage.tsx`, `DocumentViewModal.tsx` | Размеры: 1296, 622, 466, 569 строк. | Разделить на feature hooks/subcomponents при следующем изменении этих зон. |
| D.01.103 | ok | none | `frontend/src/modules/documentKinds/*` | Document-kind forms/filters/columns частично вынесены в modules. | Хорошее направление; распространить на settings/statistics. |
| D.02.108 | issue | major | `message.error(err?.message || String(err))`, `useAuthStore.formatAuthError` | Raw error strings используются в большинстве catch handlers; lockout ищется через `includes(...)`. | Ввести `formatAppError(err)`/`useAppError()` и читать stable `code/message`. |
| D.02.110 | issue | major | document modals, settings/actions | `IncomingPage` register/edit modals не передают `confirmLoading`; часть операций в settings имеет loading, часть нет. | У критичных действий должен быть локальный submitting guard; document idempotency не заменяет UX-blocking. |
| D.02.113 | issue | major | `DocumentKindPage` modals | `onCancel` закрывает модалки сразу; dirty state формы не проверяется. | Добавить unsaved changes confirmation для registration/edit и важных settings forms. |
| D.03.116 | ok | none | Zustand stores | `useAuthStore`, `useDraftLinkStore`, `useRegisterDocumentStore` содержат ограниченное UI/session state. | Глобальное состояние не выглядит избыточным. |
| D.03.121 | issue | minor | `resolveUserProfile` | Frontend profile classification дублирует backend behavior. | Оставить как `ISSUE-003`; access summary должен быть источником истины. |
| D.04.123 | ok | none | document forms | AntD `Form` validation закрывает основные required/date/number поля; `InputNumber min` используется для листов. | Базовая client validation есть, backend остается источником истины. |
| D.04.126 | issue | minor | forms/rules | Часть required rules без `whitespace` и без явных сообщений; null/empty-string policy неодинакова. | Унифицировать form rules и payload normalization вместе с typed DTO contract. |
| D.05.134 | issue | major | notifications/forms | Ошибки не маршрутизируются по `code`, field-level errors не мапятся из backend details. | После `DECISION-007` UI должен различать forbidden/not_found/conflict/validation. |
| D.06.142 | issue | minor | modals/graph/file flows | Нет явной focus recovery после ошибок/закрытия некоторых modal flows; graph interaction в основном pointer-oriented. | Проверить keyboard/screen-reader smoke, особенно модалки и link graph. |
| D.07.149 | issue | minor | Vite build | `npm run build` ранее прошел с предупреждением: основной chunk около 3001 kB, gzip около 872 kB. | На F измерить startup и решить route/code splitting или зафиксировать budget. |

## Что Передать На F/H

- Проверить frontend error contract end-to-end: login, forbidden, validation, not found, duplicate/idempotency conflict, internal error.
- UI smoke для повторного submit: регистрация, редактирование, settings CRUD, миграции, поручения, файлы.
- Accessibility smoke: keyboard navigation, focus after modal close/error, AntD table/filter controls, link graph fallback.
- Performance smoke: старт до login, открытие списков, statistics, размер bundle и memory на production-like данных.

## Closure

Статус этапа D: completed with open frontend remediation issues.

Все обязательные артефакты D созданы: `FRONTEND_REVIEW.md`, `COMPONENT_STRUCTURE_REVIEW.md`, `STATE_MANAGEMENT_REVIEW.md`, `ANTD_FORMS_REVIEW.md`, `TABLES_FILTERS_PAGINATION_REVIEW.md`, `ACCESSIBILITY_REVIEW.md`, `FRONTEND_PERFORMANCE_REVIEW.md`.
