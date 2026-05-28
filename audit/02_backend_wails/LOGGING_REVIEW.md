# Logging Review

Дата аудита: 2026-05-27

## Summary

Логирование построено на `slog`, Seq CLEF writer и Wails adapter. Уровни `debug/info/warn/error` технически доступны через `slog`. Startup failures и Wails binding errors логируются.

## Issues

### ISSUE-016: PII in production logs

Severity: major
Пункты: C.06.099

`logger.GetAppUser` добавляет `app_user` во все логи; значение обычно ФИО текущего пользователя. Admin audit/journal details содержат имена пользователей, названия файлов и номера документов. Для PostgreSQL audit trail это ожидаемо, но для Seq production logs нужна отдельная minimization policy.

Рекомендация: для technical logs использовать user ID или псевдоним; ФИО и business details оставлять в `admin_audit_log`/`document_journal`, где это доменный audit trail.

### Related ISSUE-016: insufficient critical-path logging context

Severity: minor
Пункты: C.06.100

Регистрация документов, migration rollback, file operations и idempotency/concurrency после исправлений требуют correlation fields: action, document kind, document ID, request/idempotency key hash, user ID, outcome.

Рекомендация: добавить structured fields без payload/PII и не логировать raw file content/base64.

## Point Status

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| C.06.098 | ok | none | `slog` поддерживает уровни; Wails logs ограничены ERROR. |
| C.06.099 | issue | major | Нужна PII minimization для Seq/technical logs. |
| C.06.100 | issue | minor | Critical path logs требуют correlation context. |
