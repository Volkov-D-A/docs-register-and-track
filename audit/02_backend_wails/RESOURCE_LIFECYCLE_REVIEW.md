# Resource Lifecycle Review

Дата аудита: 2026-05-27

## Summary

DB connection pool закрывается в `OnShutdown`; MinIO client не требует явного close. Файловые операции используют `os.WriteFile`, external opener запускается через `exec.Command(...).Start()`. Основная проблема не в незакрытых файлах, а в отсутствии request/app context для отмены долгих операций.

## Issues

### ISSUE-015: no propagated cancellation context

Severity: major
Пункты: C.05.094, C.05.096, C.05.097

MinIO upload/download/delete, journal writes, link graph, statistics storage info и document command journal calls используют `context.Background()`. Закрытие окна не отменяет текущие операции; Wails `OnShutdown` закрывает DB/logger, но не координирует active requests.

Рекомендация: хранить app root context из Wails startup, прокидывать context в service/repository/storage, добавлять timeout для MinIO/DB-heavy operations.

### ISSUE-017: Seq writer shutdown is timing-based

Severity: minor
Пункты: C.05.095, C.05.097

Seq writer запускает goroutine и при `Close` закрывает `done`, затем ждет `100ms`. Это не deterministic flush. Также `closeLogger` вызывается через `defer` и в `OnShutdown`; `Close` idempotent, но lifecycle стоит сделать единым.

Рекомендация: `sync.WaitGroup`/ack channel для graceful logger shutdown; один shutdown path.

## Point Status

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| C.05.094 | issue | major | Закрытие окна не отменяет операции, которые стартовали с `context.Background()`. |
| C.05.095 | issue | minor | Явный goroutine leak не доказан, но Seq shutdown timing-based. |
| C.05.096 | issue | major | DB закрывается, но MinIO/file/process operations не имеют lifecycle/cancel policy. |
| C.05.097 | issue | major | Shutdown закрывает ресурсы частично, без active-request coordination. |
