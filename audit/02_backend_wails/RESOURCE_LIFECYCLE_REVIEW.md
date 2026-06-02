# Resource Lifecycle Review

Дата аудита: 2026-05-27

## Summary

DB connection pool закрывается в `OnShutdown`; MinIO client не требует явного close. После remediation `ISSUE-015` долгие backend operations use shared app operation lifecycle with timeout and shutdown cancel/wait coordination.

## Issues

### ISSUE-015: no propagated cancellation context

Severity: major
Статус: fixed
Пункты: C.05.094, C.05.096, C.05.097

Было: MinIO upload/download/delete, journal writes, link graph, statistics storage info и document command journal calls используют `context.Background()`. Закрытие окна не отменяет текущие операции; Wails `OnShutdown` закрывает DB/logger, но не координирует active requests.

Исправлено: added `OperationLifecycle`, Wails shutdown cancel/wait coordination and lifecycle contexts for attachment, link, journal, storage statistics and document registration wrapper. MinIO startup bucket check has timeout.

### ISSUE-017: Seq writer shutdown is timing-based

Severity: minor
Статус: fixed
Пункты: C.05.095, C.05.097

Было: Seq writer запускал goroutine и при `Close` закрывал `done`, затем ждал `100ms`. Это не deterministic flush. Также `closeLogger` вызывается через `defer` и в `OnShutdown`.

Исправлено: `SeqAsyncWriter` использует `sync.WaitGroup` для graceful shutdown, `Close` idempotent через `sync.Once`, repeated close safe. Общий active-request shutdown lifecycle fixed in `ISSUE-015`.

## Point Status

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| C.05.094 | fixed | none | Wails shutdown cancels shared operation lifecycle before DB/logger close. |
| C.05.095 | ok | none | Seq writer shutdown deterministic через `sync.WaitGroup`; flush буфера и repeated close покрыты тестом. |
| C.05.096 | fixed | none | MinIO/file/link/journal/statistics operations use lifecycle contexts and timeout policy where supported. |
| C.05.097 | fixed | none | Shutdown waits for lifecycle-aware active operations or logs timeout before resource close. |
