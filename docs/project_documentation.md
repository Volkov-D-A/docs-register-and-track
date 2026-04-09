# Документация проекта: Docflow

## 1. Назначение проекта

**Docflow** — настольная система регистрации и отслеживания документов на базе **Wails**. Приложение предназначено для ведения входящих и исходящих документов, работы с поручениями, ознакомлениями, вложениями, номенклатурой дел и административными справочниками.

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

- `AuthService` — логин, logout, текущий пользователь, активная роль
- `DocumentAccessService` — централизованная проверка доступа к document-domain по роли, корневому документу и связанным сущностям доступа
- `UserService` — пользователи и пароли
- `DepartmentService` — подразделения
- `NomenclatureService` — дела номенклатуры
- `ReferenceService` — справочники типов документов, организаций и исполнителей резолюции
- `IncomingDocumentService` — входящие документы
- `OutgoingDocumentService` — исходящие документы
- `AttachmentService` — вложения и файловые операции
- `LinkService` — связи документов
- `AssignmentService` — поручения
- `AcknowledgmentService` — ознакомления
- `DashboardService` — дашборд по роли
- `SettingsService` — системные настройки и миграции
- `AdminAuditLogService` — административный аудит

### Document-domain модель

Текущая document-архитектура построена вокруг общей корневой сущности документа:

- `documents` — общий корень документа с полями `id`, `kind`, `nomenclature_id`, `document_type_id`, `content`, `pages_count`, `created_by`, `created_at`, `updated_at`
- `incoming_document_details` — типоспецифичные поля входящего документа
- `outgoing_document_details` — типоспецифичные поля исходящего документа

Связанные сущности больше не используют полиморфную пару `document_id + document_type` в БД:

- `assignments`
- `acknowledgments`
- `attachments`
- `document_journal`
- `document_links`

Все они ссылаются на единый `documents(id)`, а тип документа определяется через `documents.kind`.

Это дает:

- нормальную ссылочную целостность на уровне БД
- более простой сервисный код без дублирующего `switch` по типу документа
- единый путь расширения document-domain для будущих типов документов

## 4. Модель авторизации

### Активная роль

Пользователь может иметь несколько ролей, но backend применяет не весь набор ролей сразу, а **активную роль**.

Текущая реализация:

- активная роль хранится в `AuthService`
- после логина выставляется роль по умолчанию
- frontend синхронизирует переключение роли с backend через binding
- backend-проверки используют `RequireActiveRole(...)` и `RequireAnyActiveRole(...)`

Это нужно для того, чтобы пользователь с ролями, например, `admin + clerk` не сохранял административный доступ в document-domain, если сейчас работает в режиме `clerk`, и наоборот.

### Общие правила

- UI-ограничения не считаются механизмом безопасности
- document-domain и admin-domain проверяются на backend
- `admin` не является суперпользователем document-domain
- доступ `executor` к документам ограничен номенклатурами его подразделения
- централизованные проверки доступа к document-domain вынесены в `DocumentAccessService`
- сервисы документов, вложений и журнала не должны реализовывать собственные разрозненные policy-проверки, если можно переиспользовать `DocumentAccessService`

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

- входящие документы: `list`, `get`, `create`, `update`
- исходящие документы: `list`, `get`, `create`, `update`
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

- входящие документы: `list`, `get` только в пределах разрешенных номенклатур
- исходящие документы: `list`, `get` только в пределах разрешенных номенклатур
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
| Incoming documents | no | list/get/create/update | list/get in allowed nomenclatures |
| Outgoing documents | no | list/get/create/update | list/get in allowed nomenclatures |
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

- backend-поддержка `activeRole`
- синхронизация активной роли между frontend и backend
- вынос `admin` из document-domain
- backend-проверки доступа для документов, вложений, поручений, ознакомлений и связей
- централизация document-domain access policy через `DocumentAccessService`
- защита `JournalService` проверкой доступа к документу перед выдачей журнала
- перевод admin-domain сервисов на `activeRole`
- тесты для мульти-ролевых сценариев

Кодовые опорные точки:

- [auth_service.go](/home/dimas/projects/docs-register-and-track/internal/services/auth_service.go)
- [document_access_service.go](/home/dimas/projects/docs-register-and-track/internal/services/document_access_service.go)
- [access_policy.go](/home/dimas/projects/docs-register-and-track/internal/services/access_policy.go)
- [journal_service.go](/home/dimas/projects/docs-register-and-track/internal/services/journal_service.go)
- [useAuthStore.ts](/home/dimas/projects/docs-register-and-track/frontend/src/store/useAuthStore.ts)

## 7. Запуск и разработка

### Требования

- Go 1.24+
- Node.js и npm
- Wails CLI
- PostgreSQL
- целевая среда: Windows desktop

### Локальный запуск

Если проект запускается в Windows-окружении разработки:

```bash
cmd /c docker compose up -d
cmd /c wails dev
```

### Production-сборка

```bash
cmd /c wails build
```

Готовый бинарный файл попадает в `build/bin/`.

### Отдельная разработка frontend

```bash
cd frontend
cmd /c npm run dev
```
