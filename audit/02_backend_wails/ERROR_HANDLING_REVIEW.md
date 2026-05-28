# Error Handling Review

Дата аудита: 2026-05-27

## Summary

В проекте есть `models.AppError` и предопределенные ошибки авторизации/прав, но error boundary не доведен до production-контракта: frontend получает строку, а не объект. Ошибки validation/not-found/DB/resource смешаны между `models.NewBadRequest`, `models.ErrForbidden` и plain `fmt.Errorf`.

## Issues

### ISSUE-012: no stable structured error codes

Severity: major
Пункты: C.01.079-C.01.081, C.01.080, C.03.090, C.04.091-C.04.093

`AppError.Code` является числом, а не domain code; Wails formatter возвращает `err.Error()`. Поэтому frontend не может надежно отличить validation, forbidden, not found, conflict/idempotency и internal failure.

Рекомендация: `DECISION-007` - ввести structured backend error envelope и стабильные коды.

### Related ISSUE-012: raw internal errors can cross UI boundary

Severity: major
Пункты: C.01.079, C.01.081

Многие repository/service методы возвращают wrapped internal errors вроде `failed to create document root`, `failed to get file content`, `invalid date format, expected RFC3339`. Это полезно для логов, но не для пользователя.

Рекомендация: на service/Wails boundary маппить internal errors в safe messages; логировать original error с context.

### ISSUE-018: inconsistent not-found behavior

Severity: minor
Пункты: C.04.093

Статус: fixed after audit.

Отсутствие сущности больше не возвращается как validation/silent nil/plain not-found на исправленных production paths. Document access, assignment, admin access target user, administrative-order acknowledgment row, registration nomenclature and department update/delete paths return structured `NOT_FOUND`/404.

Проверка: `go test ./internal/services`, `go test ./internal/repository`; search по services/repositories не находит production `fmt.Errorf("... not found")` или `NewBadRequest("...не найден")` paths.

## Error Families To Standardize

| Family | Target code |
| --- | --- |
| Authentication required | `UNAUTHORIZED` |
| Permission denied | `FORBIDDEN` |
| Validation failure | `VALIDATION_ERROR` |
| Entity missing | `NOT_FOUND` |
| Duplicate/idempotency conflict | `CONFLICT` / `IDEMPOTENCY_CONFLICT` |
| Storage unavailable | `STORAGE_ERROR` |
| Database unavailable/internal | `INTERNAL_ERROR` |
