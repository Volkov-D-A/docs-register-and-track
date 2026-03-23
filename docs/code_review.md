# Код-ревью проекта docs-register-and-track

> **Дата ревью:** 21 марта 2026  
> **Ревьюер:** AI Code Reviewer  
> **Область:** Backend (Go), инфраструктура (Docker, Makefile), конфигурация

---

## Общая оценка

Проект представляет собой **десктопное приложение** (Wails + Go + PostgreSQL + MinIO) для регистрации и отслеживания документов. Архитектура чистая: классическая трёхслойная структура *Models → Repository → Service* с DTO-слоем и маппинг-функциями. Код хорошо документирован, есть тесты, используются транзакции и параметризованные запросы.

**Ниже приведены найденные проблемы** — от критических до незначительных.

---

## 🔴 Критические проблемы

### ✅ 1. Захардкоженный ключ шифрования по умолчанию (ИСПРАВЛЕНО)

**Файл:** [crypto.go](file:///home/dimas/projects/docs-register-and-track/internal/config/crypto.go#L29)

```go
const defaultDevKey = "dOcFl0wApp-S3cR3t-K3y!AES256ok!!"
```

Если приложение собирается **без `-ldflags`** (забыли, CI-ошибка и т.д.), используется этот ключ. Все пароли в `config.json` могут быть расшифрованы любым, кто знает этот ключ (который лежит в открытом исходном коде).

**Рекомендация:** При продакшн-сборке **не запускать** приложение без явного ключа. Добавить проверку при старте:

```go
func init() {
    if rawEncryptionKey == "" {
        log.Println("WARNING: using default dev encryption key. Set ENCRYPTION_KEY for production!")
    }
    // ...
}
```

---

### ✅ 2. Секреты в `.env` под контролем версий (НЕАКТУАЛЬНО - УЖЕ В .GITIGNORE)

**Файл:** [.env](file:///home/dimas/projects/docs-register-and-track/.env#L5)

```
ENCRYPTION_KEY=5DSn65Q5wRz9Fwue8DV3UBZ/PUzZ/m1F
MINIO_ROOT_PASSWORD=docflow_password
POSTGRES_PASSWORD=docflow_password
SMB_PASSWORD=1234
```

Файл `.env` содержит **реальные секреты** и при этом не указан в [.gitignore](file:///home/dimas/projects/docs-register-and-track/.gitignore). Если репозиторий публичный или попадёт в чужие руки — все пароли скомпрометированы.

**Рекомендация:** Добавить `.env` в `.gitignore`, оставив только `.envExample` в репозитории.

---

### ✅ 3. Приложение стартует при недоступной БД — nil pointer panics (ИСПРАВЛЕНО)

**Файл:** [main.go](file:///home/dimas/projects/docs-register-and-track/main.go#L53-L56)

```go
db, err := database.Connect(cfg.Database)
if err != nil {
    log.Printf("Warning: Failed to establish database connection pool: %v", err)
}
// db может быть nil, но все репозитории создаются с db = nil
userRepo := repository.NewUserRepository(db)
```

Если подключение не удалось, `db` будет `nil`. Репозитории создаются с `nil` указателем, и при первом обращении к БД — **panic: nil pointer dereference**. Аналогичная проблема с MinIO (строка 88–91).

**Рекомендация:** Либо завершать работу при критической ошибке подключения, либо добавить nil-проверки во все методы репозиториев, либо реализовать отложенное подключение (lazy connect).

---

### ✅ 4. Журналирование ПЕРЕД удалением (Delete подряд может потерять данные) (ИСПРАВЛЕНО)

**Файл:** [incoming_doc_service.go](file:///home/dimas/projects/docs-register-and-track/internal/services/incoming_doc_service.go#L299-L308)

```go
func (s *IncomingDocumentService) Delete(id string) error {
    // ... журналирование вызывается ДО удаления:
    s.journal.LogAction(...)
    return s.repo.Delete(uid) // если Delete упадёт, запись в журнале останется ложной
}
```

**Аналогично в:** `outgoing_doc_service.go` Delete.

Журналирование записывает «Документ удален», но само удаление может **не пройти**. В результате журнал содержит запись о несуществующем событии.

**Рекомендация:** Логировать **после** успешного удаления (как сделано в `assignment_service.go` Delete — `if err == nil { s.journal.LogAction(...) }`).

---

## 🟠 Серьёзные проблемы

### ✅ 5. `AuthService` хранит текущего пользователя in-memory — не работает для многопользовательского варианта (ИСПРАВЛЕНО)

**Файл:** [auth_service.go](file:///home/dimas/projects/docs-register-and-track/internal/services/auth_service.go#L27-L32)

```go
type AuthService struct {
    db          *database.DB
    userRepo    UserStore
    currentUser *models.User   // ← единственный пользователь на весь процесс
    mu          sync.RWMutex
}
```

В десктопном Wails-приложении это допустимо (один пользователь на инстанс). Однако **если когда-нибудь проект будет расширен** до веб-варианта или multi-window — этот подход полностью сломается. Стоит задокументировать это ограничение.

**Также:** При обновлении профиля (`UpdateProfile`) кешированный `currentUser` обновляется вручную — при ошибке может возникнуть рассинхронизация между состоянием в памяти и в БД.

---

### 6. Непоследовательная обработка ошибок авторизации [ИСПРАВЛЕНО]

Код использует разные ошибки для одних и тех же ситуаций:

| Ситуация                    | Файл                                                         | Ошибка                                 |
| --------------------------- | ------------------------------------------------------------ | -------------------------------------- |
| Нет прав (admin only)       | `user_service.go` CreateUser                                 | `ErrNotAuthenticated` (401)            |
| Нет прав (admin only)       | `department_service.go` CreateDepartment                     | `ErrForbidden` (403)                   |
| Нет прав (admin only)       | `reference_service.go` DeleteOrganization                    | `fmt.Errorf("недостаточно прав")` (500) |

В `user_service.go` возвращается `ErrNotAuthenticated` (код 401), хотя пользователь **аутентифицирован**, но не имеет нужной роли. Семантически это должен быть `ErrForbidden` (403).

В `reference_service.go:137` используется обычная `fmt.Errorf`, а не `models.ErrForbidden`. Фронт не сможет корректно обработать эту ошибку.

**Рекомендация:** Унифицировать — всегда использовать `models.ErrForbidden` для отказа в доступе, `ErrNotAuthenticated` только если пользователь не залогинен.

---

### 7. Ошибки от `uuid.Parse` в `GetCurrentUserID()` молча игнорируются [ИСПРАВЛЕНО]

**Многие сервисы:**

```go
currentUserID, _ := uuid.Parse(s.auth.GetCurrentUserID())
```

Результат `GetCurrentUserID()` может быть пустой строкой (если `currentUser == nil`). В этом случае `uuid.Parse` вернёт ошибку, а `currentUserID` будет `uuid.Nil`. Журнальная запись будет создана с `UserID = uuid.Nil` — бессмысленное значение.

**Рекомендация:** Использовать `getCurrentUserIDSafe()` (уже есть, но не используется в сервисах) или хотя бы проверять ошибку.

---

### 8. Миграции ищутся по относительному пути

**Файл:** [auth_service.go](file:///home/dimas/projects/docs-register-and-track/internal/services/auth_service.go#L17), [settings.go](file:///home/dimas/projects/docs-register-and-track/internal/services/settings.go#L11)

```go
const migrationsPathAuth = "internal/database/migrations"
const migrationsPath = "internal/database/migrations"
```

Две отдельные константы для одного и того же пути. При запуске из другой директории (или из скомпилированного бинарника) путь не будет найден.

**Рекомендация:**
1. Объединить в одну константу/переменную
2. Либо встраивать миграции через `embed.FS`, либо определять путь относительно бинарника

---

### 9. ContentType вложения записывается неверно [ИСПРАВЛЕНО]

**Файл:** [attachment.go](file:///home/dimas/projects/docs-register-and-track/internal/services/attachment.go#L105)

```go
ContentType:  ext, // упрощённый тип содержимого, например ".pdf"
```

В поле `ContentType` записывается **расширение файла** (`.pdf`), а не MIME-тип (`application/pdf`). Это нарушает семантику поля и может привести к проблемам при скачивании или отображении файлов.

**Рекомендация:** Использовать `mime.TypeByExtension(ext)` из стандартной библиотеки Go.

---

### 10. `Upload` передаёт расширение файла вместо MIME-типа в хранилище [ИСПРАВЛЕНО]

**Файл:** [attachment.go](file:///home/dimas/projects/docs-register-and-track/internal/services/attachment.go#L90)

```go
if err := s.fileStorage.UploadFile(context.Background(), objectName, data, ext); err != nil {
```

В MinIO `PutObject` ожидает `Content-Type`, а передаётся расширение вроде `.pdf`. Это приводит к некорректным заголовкам при скачивании файлов через MinIO API.

---

## 🟡 Средние проблемы

### 11. Дупликация кода в слайс-маперах

**Файл:** [mapper.go](file:///home/dimas/projects/docs-register-and-track/internal/dto/mapper.go)

Все `Map*s` (MapUsers, MapDepartments, MapNomenclatures и т.д.) — это идентичный код, различающийся только типом. Go 1.18+ позволяет использовать дженерики:

```go
func MapSlice[T any, R any](items []T, mapFn func(*T) *R) []R {
    if items == nil { return nil }
    res := make([]R, len(items))
    for i, v := range items {
        if mapped := mapFn(&v); mapped != nil {
            res[i] = *mapped
        }
    }
    return res
}
```

Это уменьшит `mapper.go` с ~530 строк до ~200 строк.

---

### 12. Дубликация логики получения списка пользователей (N-запросов)

**Файл:** [user_repo.go](file:///home/dimas/projects/docs-register-and-track/internal/repository/user_repo.go)

Методы `GetAll()` (~65 строк, L85–L151) и `GetExecutors()` (~65 строк, L281–L349) содержат **почти идентичный код** — сканирование строк, обработка department, batch-загрузка ролей и номенклатур. Разница только в SQL-запросе.

**Рекомендация:** Выделить общую функцию `fetchUsersWithDetails(query string, args ...interface{})`.

---

### 13. Отсутствие валидации входных данных на уровне сервисов

В сервисах `Register` (документы) не проверяются:
- Пустые строки (`subject`, `content`, `senderSignatory`)
- Валидность `pagesCount` (может быть 0 или отрицательным)
- Длина текстовых полей

**Рекомендация:** Добавить слой валидации для бизнес-правил (хотя бы `subject` не пустой, `pagesCount > 0`).

---

### 14. Сервисы принимают слишком много отдельных параметров

**Файл:** [incoming_doc_service.go](file:///home/dimas/projects/docs-register-and-track/internal/services/incoming_doc_service.go#L94-L103)

```go
func (s *IncomingDocumentService) Register(
    nomenclatureID, documentTypeID string,
    senderOrgName, recipientOrgName string,
    incomingDate, outgoingDateSender string,
    outgoingNumberSender string,
    intermediateNumber, intermediateDateStr string,
    subject, content string, pagesCount int,
    senderSignatory, senderExecutor, addressee string,
    resolution string,
) (*dto.IncomingDocument, error) {
```

**15 параметров** — крайне неудобно для вызова и поддержки. Порядок легко перепутать (все `string`).

**Рекомендация:** Создать request-структуру (DTO) для таких методов:

```go
type RegisterIncomingDocRequest struct {
    NomenclatureID string `json:"nomenclatureId"`
    // ...
}
```

---

### 15. Отсутствует таймаут при подключении к БД и MinIO

**Файл:** [postgres.go](file:///home/dimas/projects/docs-register-and-track/internal/database/postgres.go#L33)

```go
db, err := sql.Open("postgres", cfg.ConnectionString())
```

Нет настройки `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`. При проблемах с сетью соединения могут накапливаться бесконтрольно.

**Рекомендация:**

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

### 16. Не закрывается MinIO при shutdown

**Файл:** [main.go](file:///home/dimas/projects/docs-register-and-track/main.go#L108-L111)

```go
OnShutdown: func(ctx context.Context) {
    log.Println("Gracefully shutting down database connection...")
    db.Close()
},
```

Соединение с БД закрывается, но MinIO-клиент — нет. Хотя для MinIO Go SDK это не критично (нет пула), стоит быть последовательным.

---

### 17. Дублированные комментарии в `link_service.go`

**Файл:** [link_service.go](file:///home/dimas/projects/docs-register-and-track/internal/services/link_service.go#L79-L82)

```go
// Логирование создания связи (для обоих документов)
// Логирование создания связи (для обоих документов)
// Упрощенный лог (без номеров, просто факт)
// Упрощенный лог (без номеров, просто факт)
```

Повторяющиеся комментарии — артефакт копирования.

---

### 18. `BulkDeleteOlderThan` теряет файлы при частичном сбое

**Файл:** [attachment.go](file:///home/dimas/projects/docs-register-and-track/internal/services/attachment.go#L369-L383)

```go
for _, att := range attachments {
    if err := s.fileStorage.DeleteFile(...); err != nil {
        fmt.Printf("Failed to delete file %s from MinIO: %v\n", ...)
        continue // Файл не удалён из MinIO, но и не удален из БД — ок
    }
    successfulIDs = append(successfulIDs, att.ID) // Удалён из MinIO
}
// Если DeleteMultiple упадёт — файлы удалены из MinIO, но записи в БД остались
if len(successfulIDs) > 0 {
    if err := s.repo.DeleteMultiple(successfulIDs); err != nil {
        return 0, fmt.Errorf("failed to delete records from db: %v", err) // ПОТЕРЯ ФАЙЛОВ
    }
}
```

Если `DeleteMultiple` падает — файлы уже удалены из MinIO, но записи в БД остались. «Осиротевшие» записи указывают на несуществующие файлы.

**Рекомендация:** Удалять из БД **первоначально**, а из MinIO — потом. Или оборачивать в транзакцию + undo-список.

---

## 🔵 Незначительные замечания

### 19. Смешение языков в сообщениях об ошибках

Часть ошибок на **русском**, часть на **английском**:

```go
// auth_service.go
return nil, fmt.Errorf("ошибка создания администратора: %w", err)

// incoming_doc_service.go
return nil, fmt.Errorf("invalid ID: %w", err)
```

**Рекомендация:** Выбрать один язык и придерживаться его. Для внутренних ошибок (логи) — предпочтительнее английский; для пользовательских — русский через `AppError`.

---

### 20. Отсутствует `context.Context` в большинстве интерфейсов

Интерфейсы `UserStore`, `IncomingDocStore`, `OutgoingDocStore` и др. не принимают `context.Context`, в то время как `LinkStore` и `JournalStore` — принимают. Это несогласованность, которая затруднит добавление таймаутов и трассировки в будущем.

---

### 21. Отсутствие `Startup` callback в большинстве сервисов

Только `SystemService` реализует `Startup(ctx)` для получения Wails-контекста. Другие сервисы (`AttachmentService`, `LinkService`) используют `context.Background()`. Если понадобится доступ к Wails runtime — придётся рефакторить.

---

### 22. Неиспользуемое поле `Filepath` в Attachment

**Файл:** [attachment.go (models)](file:///home/dimas/projects/docs-register-and-track/internal/models/attachment.go#L15)

```go
Filepath  string `json:"filepath"` // внутренний путь
```

Поле `Filepath` определено в модели и DTO, но нигде не заполняется при создании вложения (в сервисе `Upload`). Вероятно, осталось от старой реализации с файловой системой.

---

### 23. `SystemService.ctx` сохраняется, но не используется

**Файл:** [system_service.go](file:///home/dimas/projects/docs-register-and-track/internal/services/system_service.go#L11)

```go
type SystemService struct {
    ctx context.Context   // присваивается в Startup, но нигде не используется
    db  *database.DB
}
```

---

### 24. Нет `PageSize` cap в IncomingDocumentService

В `OutgoingDocumentService.GetList()` есть дефолты для `Page` и `PageSize`, но в `IncomingDocumentService.GetList()` — нет. Пользователь может запросить `PageSize = 1000000`.

---

## ✅ Что сделано хорошо

| Аспект                        | Детали                                                                                            |
| ----------------------------- | ------------------------------------------------------------------------------------------------- |
| **Слоистая архитектура**      | Чёткое разделение Models → Repository → Service → DTO. Интерфейсы для всех хранилищ.              |
| **Тесты**                     | Тесты для всех репозиториев, сервисов, конфигураций и безопасности. Используются моки.              |
| **Batch-загрузка**            | `batchLoadUserRoles` и `batchLoadDepartmentNomenclatures` решают проблему N+1 запросов.            |
| **Параметризованные запросы** | Все SQL-запросы используют `$1, $2, ...` — защита от SQL-инъекций.                                |
| **Транзакции**                | Создание/обновление пользователей оборачивается в `tx.Begin()` / `tx.Commit()` с `defer Rollback`. |
| **Шифрование конфига**        | AES-256-GCM для паролей в `config.json`. `ldflags` для инъекции ключа.                          |
| **Аудит-лог**                 | `AdminAuditLogService` фиксирует все административные действия. Nil-safe `LogAction`.              |
| **Дженерики**                 | `PagedResult[T any]` — хороший пример использования дженериков Go.                                |
| **Хелперы**                   | `filterNomenclaturesByDepartment` — вынесенная общая логика фильтрации с хорошей документацией.    |
| **Docker Compose**            | PostgreSQL + MinIO в одной конфигурации с alpine-образами и именованными томами.                   |
| **Makefile**                  | Удобные цели: `dev`, `build-linux`, `build-windows`, `storage-up/down/reset`.                     |

---

## Приоритетный план исправлений

| Приоритет  | Задача                                                  | Сложность |
| ---------- | ------------------------------------------------------- | --------- |
| 🔴 Высокий | ~~Добавить `.env` в `.gitignore`~~ (Сделано)                | Тривиально |
| 🔴 Высокий | ~~Исправить порядок journal/delete (п.4)~~ (Сделано)        | Лёгкая     |
| 🔴 Высокий | ~~Обработать `db == nil` в main.go (п.3)~~ (Сделано)         | Средняя    |
| 🟠 Средний | Унифицировать ошибки авторизации (п.6)                  | Средняя    |
| 🟠 Средний | Исправить ContentType вложений (п.9–10)                 | Лёгкая     |
| 🟠 Средний | Настроить пул соединений БД (п.15)                      | Лёгкая     |
| 🟡 Низкий  | Рефакторинг: request-структуры вместо 15 параметров (п.14) | Средняя |
| 🟡 Низкий  | Рефакторинг: дженерик-маперы (п.11)                     | Средняя    |
| 🟡 Низкий  | Добавить валидацию входных данных (п.13)                 | Средняя    |
| 🔵 Мелочи  | Удалить дубликаты комментариев (п.17)                   | Тривиально |
| 🔵 Мелочи  | Унифицировать язык ошибок (п.19)                        | Лёгкая     |
