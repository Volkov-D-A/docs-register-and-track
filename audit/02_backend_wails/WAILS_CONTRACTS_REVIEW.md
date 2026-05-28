# Wails Contracts Review

Дата аудита: 2026-05-27

## Public Bindings

`main.go` публикует через Wails следующие сервисы: `AuthService`, `UserService`, `NomenclatureService`, `ReferenceService`, `DocumentAccessAdminService`, `DocumentKindService`, `DocumentQueryService`, `DocumentRegistrationService`, `AdministrativeOrderService`, `AssignmentService`, `DashboardService`, `StatisticsService`, `DepartmentService`, `SettingsService`, `AttachmentService`, `LinkService`, `AcknowledgmentService`, `SystemService`, `ReleaseNoteService`, `ThemeService`, `JournalService`, `AdminAuditLogService`.

Контракт сейчас implicit: frontend получает методы из Wails bindings по exported Go methods. Это рабочая схема, но production-аудит фиксирует риск: без явного contract file/schema изменения сигнатур, ошибки и nullable/date behavior трудно контролировать между Go и TypeScript.

## Issues

### ISSUE-012: string-only error boundary

Severity: major
Пункты: C.01.079-C.01.081, C.03.090, C.04.091-C.04.093
Место: `main.go` `ErrorFormatter`, `internal/models/errors.go`

`ErrorFormatter` логирует `err.Error()` и возвращает во frontend plain string. `AppError` содержит `Code int` и `Message`, но этот объект не доходит до frontend как структура. Обычные `fmt.Errorf` также становятся пользовательскими строками.

Рекомендация: возвращать structured envelope `{ code, message, details? }`; `details` только для dev/logs, не для UI.

### ISSUE-014: document command API accepts any

Severity: major
Пункты: C.02.083-C.02.085, C.03.086
Место: `DocumentRegistrationService.Register(kindCode string, req any)`, `Update(kindCode string, req any)`

Payload декодируется в typed request через `json.Marshal`/`json.Unmarshal`. Неизвестные поля не отклоняются, обязательный `idempotencyKey` отсутствует, contract для frontend неочевиден.

Рекомендация: typed public methods или strict decoder with unknown-field rejection; добавить `idempotencyKey` в create DTO всех 4 видов документов.

## Contract Findings

| Группа | Статус | Вывод |
| --- | --- | --- |
| Auth/session | issue | Методы понятны, но errors string-only; login failure безопасен. |
| Users/settings/references | issue | CRUD contracts понятны, но validation/not-found errors неоднородны. |
| Documents registration/update | issue | Главный риск: `any` payload, no idempotency key, unknown fields ignored. |
| Documents query/access | ok | Query methods имеют достаточно ясные signatures; access проверяется backend-side. |
| Attachments | issue | Base64 upload/download понятен, но errors и context/cancel требуют hardening. |
| Migration management | issue | Runtime rollback сохранен по `DECISION-003`; нужны guardrails и structured errors. |
