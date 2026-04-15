# Документация проекта: Docflow

## 1. Назначение проекта

**Docflow** — настольная система регистрации и отслеживания документов на базе **Wails**. Приложение предназначено для ведения регистров документов, работы с поручениями, ознакомлениями, вложениями, номенклатурой дел и административными справочниками.

Ключевая особенность текущей версии: backend является источником истины для авторизации. Ограничения доступа не должны обеспечиваться только скрытием экранов во frontend.

## 2. Технологический стек

### Backend

- Go 1.24
- Wails v2
- PostgreSQL
- `github.com/lib/pq`
- миграции БД через `github.com/golang-migrate/migrate/v4`
- `github.com/google/uuid`
- `golang.org/x/crypto`
- `stretchr/testify`, `go-sqlmock`

### Frontend

- React 18
- TypeScript
- Vite
- Ant Design
- Zustand
- Day.js
- `@xyflow/react`

## 3. Архитектура

Проект построен как desktop-приложение с Go backend и React frontend.

Основные директории:

- `internal/database` — подключение к БД и миграции
- `internal/repository` — слой доступа к данным
- `internal/services` — бизнес-логика и авторизация
- `internal/models` — доменные модели
- `internal/dto` — DTO и маппинг
- `frontend/src` — UI и client-side state
- `docs` — проектная документация
- `main.go` — сборка зависимостей и binding сервисов в Wails

### Основные сервисы

- `AuthService` — логин, logout, текущий пользователь и его роли
- `DocumentAccessService` — централизованная проверка доступа к document-domain по матрице прав, корневому документу и связанным сущностям доступа
- `DocumentKindService` — системные метаданные видов документов и доступных пользователю сценариев регистрации
- `UserService` — пользователи и пароли
- `DepartmentService` — подразделения
- `NomenclatureService` — дела номенклатуры
- `ReferenceService` — справочники типов документов, организаций и исполнителей резолюции
- `DocumentQueryService` — общий read-only API документов по `kindCode`
- `DocumentRegistrationService` — общий command API регистрации и обновления документов по `kindCode`
- `AttachmentService` — вложения и файловые операции
- `LinkService` — связи документов
- `AssignmentService` — поручения
- `AcknowledgmentService` — ознакомления
- `DashboardService` — дашборд по роли
- `SettingsService` — системные настройки и миграции
- `AdminAuditLogService` — административный аудит

### Document-domain модель

Текущая document-архитектура построена вокруг общей корневой сущности документа и системных видов документов:

- системные виды документов пока захардкожены в коде: `incoming_letter`, `outgoing_letter`
- backend-реестр видов документов хранится в `internal/models/document_kind.go`
- frontend использует тот же системный словарь через `frontend/src/constants/documentKinds.ts` и `DocumentKindService`
- `documents` — общий корень документа с полями `id`, `kind`, `nomenclature_id`, `registration_number`, `registration_date`, `document_type_id`, `content`, `pages_count`, `created_by`, `created_at`, `updated_at`
- `incoming_document_details` — типоспецифичные поля входящего документа
- `outgoing_document_details` — типоспецифичные поля исходящего документа

Связанные сущности больше не используют полиморфную пару `document_id + document_type` в БД:

- `assignments`
- `acknowledgments`
- `attachments`
- `document_journal`
- `document_links`

Все они ссылаются на единый `documents(id)`, а тип документа определяется через `documents.kind`.

На уровне DTO и UI связанные сущности тоже переведены на `documentKind`:

- `assignments.documentKind`
- `acknowledgments.documentKind`
- `document_links.sourceKind / targetKind`
- `graph.nodes.kindCode`

Это дает:

- нормальную ссылочную целостность на уровне БД
- более простой сервисный код без дублирующего `switch` по типу документа
- единый путь расширения document-domain для будущих типов документов

## 4. Модель авторизации

### Общие правила

- UI-ограничения не считаются механизмом безопасности
- document-domain и admin-domain проверяются на backend
- `admin` не является суперпользователем document-domain
- эффективные права вычисляются по реальным ролям пользователя без переключаемой `activeRole`
- доступ `executor` к документам ограничен каналами видимости: номенклатура подразделения, поручения, ознакомления
- централизованные проверки доступа к document-domain вынесены в `DocumentAccessService`
- document read/write flow проходит через `DocumentQueryService` и `DocumentRegistrationService`
- legacy-значения `incoming/outgoing` нормализуются в системные `incoming_letter/outgoing_letter` через `NormalizeDocumentKind(...)`
- сервисы вложений, связей, поручений, ознакомлений и журнала не должны реализовывать собственные разрозненные policy-проверки, если можно переиспользовать `DocumentAccessService`

## 5. Матрица доступа

### Роли

#### `admin`

Сервисный администратор. Работает только с системными и административными сущностями.

Доступ:

- пользователи и роли
- подразделения
- системные настройки
- миграции БД
- административный аудит
- системный дашборд
- административные справочники и оргструктура
- batch/system операции административного характера

Нет доступа:

- входящие документы
- исходящие документы
- связи документов
- вложения документов в рамках document-domain
- поручения
- ознакомления

#### `clerk`

Делопроизводитель. Имеет полный доступ к документному контуру.

Доступ:

- входящие письма: `list`, `get`, `create`, `update`
- исходящие письма: `list`, `get`, `create`, `update`
- связи документов: `list`, `get-graph`, `create`, `delete`
- вложения: `list`, `upload`, `download`, `delete`
- поручения: `list`, `get`, `create`, `update`, `delete`
- review-переходы статусов поручений: `completed -> finished/returned`
- ознакомления: `create`, `list`, `delete`
- рабочие справочники и номенклатура: чтение и изменение там, где это разрешено бизнес-процессом
- операционный дашборд

#### `executor`

Исполнитель. Работает только со своими задачами и доступными ему документами.

Доступ:

- входящие письма: `list`, `get` в пределах разрешенных каналов видимости
- исходящие письма: `list`, `get` в пределах разрешенных каналов видимости
- вложения: `list`, `download` только если доступен связанный документ
- поручения: `list`, `get` только свои
- смена статуса поручений: только своих
- ознакомления: просмотр только своих, отметки `viewed` и `confirmed`
- минимально необходимые справочники: только чтение
- персональный дашборд

Нет доступа:

- создание, редактирование и удаление документов
- связи документов
- загрузка и удаление вложений
- создание, редактирование и удаление поручений
- создание и удаление ознакомлений

### Матрица по сущностям

| Сущность | admin | clerk | executor |
| --- | --- | --- | --- |
| Incoming letters | no | list/get/create/update | list/get in allowed scopes |
| Outgoing letters | no | list/get/create/update | list/get in allowed scopes |
| Document links | no | list/get-graph/create/delete | no |
| Attachments | no | list/upload/download/delete | list/download if document is accessible |
| Assignments | no | list/get/create/update/delete/status review | list/get own only, own status changes only |
| Acknowledgments | no | create/list/delete | own only, viewed/confirmed |
| Users and roles | full | limited if explicitly allowed | no |
| Departments | full | limited if explicitly allowed | read only if needed |
| Settings | full | no unless explicitly granted | no |
| Migrations | full | no | no |
| Admin audit log | full | no | no |
| Reference data | full | read and modify where business process requires | read only minimal |
| Dashboard | system | operational | personal |

## 6. Реализованные изменения по безопасности доступа

На текущем этапе уже внедрено:

- отказ от `activeRole` в backend и frontend
- вычисление рабочего профиля пользователя без ручного переключения роли
- вынос `admin` из document-domain
- матрица действий по виду документа и visibility-каналы
- backend-проверки доступа для документов, вложений, поручений, ознакомлений и связей
- централизация document-domain access policy через `DocumentAccessService`
- защита `JournalService` проверкой доступа к документу перед выдачей журнала
- общий `DocumentQueryService` для карточек и списков документов
- общий `DocumentRegistrationService` для регистрации и обновления документов по `kindCode`
- тесты для мульти-ролевых и матричных сценариев

## 7. Frontend-архитектура document-domain

Frontend больше не строится вокруг двух отдельных приложений "входящие" и "исходящие". Сейчас используется общий document-kind каркас:

- `DocumentKindPage` — общий orchestration-контейнер страницы вида документа
- `DocumentListPage` / `DocumentListTable` — общий список документов
- `DocumentViewModal` — общая карточка документа с kind-specific detail-блоком
- `useDocumentListPage` — общий hook списков
- `useDocumentKindModals` — общий hook register/edit modal flow
- `modules/documentKinds/incomingLetter` и `modules/documentKinds/outgoingLetter` — видоспецифичные формы, фильтры, config и helper-логика

Это означает, что добавление нового вида документа теперь требует в первую очередь:

- нового backend handler-а
- нового frontend модуля вида документа
- регистрации вида в системном kind-registry

а не копирования целой параллельной страницы и набора сервисов.

## 8. Регистрация документов

Регистрация документа в UI теперь проходит через единый сценарий:

- плавающая кнопка `Зарегистрировать` в `MainLayout`
- загрузка доступных видов документов через `DocumentKindService.GetAvailableForRegistration()`
- выбор вида документа
- открытие видоспецифичной формы
- сохранение через `DocumentRegistrationService.Register(kindCode, payload)`

Редактирование построено так же:

- вид документа определяет форму
- сохранение выполняется через `DocumentRegistrationService.Update(kindCode, payload)`

Это делает backend единым входом для command-side document-domain.

## 9. Дашборд

Дашборд больше не зависит от ручного переключения роли.

Текущая схема:

- backend вычисляет рабочий профиль пользователя
- поддерживаются профили `admin`, `clerk`, `executor`, `mixed`
- frontend рендерит соответствующий сценарий без `activeRole`

Для document-domain это важно, потому что:

- делопроизводитель видит операционные метрики по регистрам
- исполнитель видит персональные поручения и ознакомления
- mixed-профиль получает объединенный сценарий

## 10. Текущее состояние платформы

На данный момент в проекте уже зафиксированы следующие архитектурные опоры:

- системные виды документов `incoming_letter` и `outgoing_letter`
- общая корневая сущность `documents`
- kind-aware access layer
- kind-aware query layer
- kind-aware registration/update layer
- общая карточка документа
- общий список документов
- общая регистрация через FAB и выбор вида документа
- `documentKind` как основной транспортный формат вокруг поручений, ознакомлений и связей

Это означает, что следующий новый вид документа можно добавлять уже как расширение платформы, а не как отдельную ветку логики приложения.

Кодовые опорные точки:

- [auth_service.go](/home/dimas/projects/docs-register-and-track/internal/services/auth_service.go)
- [document_kind.go](/home/dimas/projects/docs-register-and-track/internal/models/document_kind.go)
- [document_kind_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_kind_service.go)
- [document_access_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_access_service.go)
- [document_query_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_query_service.go)
- [document_kind_query_handler.go](/home/dimas/projects/docs-register-and-track/internal/services/document_kind_query_handler.go)
- [document_kind_command_handler.go](/home/dimas/projects/docs-register-and-track/internal/services/document_kind_command_handler.go)
- [journal_service.go](/home/dimas/projects/docs-register-and-track/internal/services/journal_service.go)
- [MainLayout.tsx](/home/dimas/projects/docs-register-and-track/frontend/src/components/MainLayout.tsx)
- [DocumentKindPage.tsx](/home/dimas/projects/docs-register-and-track/frontend/src/components/DocumentKindPage.tsx)
- [DocumentViewModal.tsx](/home/dimas/projects/docs-register-and-track/frontend/src/components/DocumentViewModal.tsx)
- [documentKinds.ts](/home/dimas/projects/docs-register-and-track/frontend/src/constants/documentKinds.ts)
- [useAuthStore.ts](/home/dimas/projects/docs-register-and-track/frontend/src/store/useAuthStore.ts)
