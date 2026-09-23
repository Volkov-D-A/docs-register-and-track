# Тестовая инфраструктура

Здесь находятся вспомогательные файлы запуска тестов. Сами Go-тесты остаются
в соответствующих пакетах, frontend-тесты — в `frontend/test`.

- `compose/integration.yaml` — изолированные PostgreSQL, SeaweedFS и сервер для интеграционных проверок.
- `compose/backup-test.yaml` — тестовый Samba-сервер.
- `compose/backup-integration.yaml` и `scripts/backup-integration.sh` — сквозная проверка резервирования и восстановления с покрытием серверного бинарника.
- `compose/prod-smoke.override.yaml` и `scripts/prod-compose-smoke.sh` — тестовые образ, секреты и порты поверх production Compose.
- `images/samba/` — Dockerfile и конфигурация тестового Samba.

Запуск из корня репозитория:

```bash
make integration-test
make storage-smoke-test
```

`make integration-test` сначала запускает пакетные Go-тесты с именем `Integration`
на PostgreSQL и S3, затем проверяет backup/restore через инструментированный
сервер и Samba. Профили этих интеграционных сценариев и их объединение сохраняются
в `build/release-evidence/`; обычные unit-тесты в этот профиль не входят.
Для ручной работы с первым
стеком доступны `make integration-db-up` и `make integration-db-down`.

`make storage-smoke-test` запускает `docs/examples/docker-compose.prod.example.yaml`
с тестовым override. Он проверяет Caddy и сохранность вложений при пересоздании
SeaweedFS и перезапуске сервера. Отчёты находятся в
`build/transition-evidence/prod-compose-smoke/`.

Тестовые команды удаляют собственные тестовые тома при завершении. Не используйте
тестовые учётные данные и Compose override для рабочего окружения.
