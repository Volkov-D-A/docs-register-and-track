# Build Review

Дата аудита: 2026-05-28
Этап: E.01. Production build и версия приложения

## Общий Вывод

Wails production build описан через `wails.json` и `Makefile`: frontend собирается командой `npm run build`, release assets генерируются через `go generate ./internal/releaseassets`, Linux/Windows binaries собираются Wails targets. Embedded frontend assets и embedded migrations снижают зависимость runtime от исходных директорий.

Основные build risks:

- production build still requires approved external `ENCRYPTION_KEY` delivery and target OS artifact smoke;
- target OS install/update smoke remains open.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.01.154 | ok | none | build | `Makefile`, `wails.json` | `release-gate` validates `ENCRYPTION_KEY`, generated release assets, Go checks, `govulncheck`, `npm ci`, lint/build/audit/license inventory; Wails `frontend:install` uses `npm ci`. | Clean build contract is maintained; target OS artifact smoke remains separate. |
| E.01.155 | ok | none | build/update | `docs/releases.yaml`, `internal/releaseassets/current_release.yaml`, Wails metadata | `docs/releases.yaml` генерирует `current_release.yaml` и `wails.json` `info.productVersion`; `TestWailsProductVersionMatchesCurrentRelease` ловит рассинхрон. | About/release notes и Wails product metadata используют одну release version. |
| E.01.156 | ok | none | runtime | `AboutProgramModal.tsx`, `ReleaseNoteService` | About modal показывает `release?.version` и дату релиза из embedded release note; Wails metadata синхронизируется тем же release source. | Версия видна пользователю и связана с binary/installer metadata через release generation. |
| E.01.157 | issue | minor | build/ops | backup/restore scripts | Build itself uses relative paths, but ops scripts embed `/home/dimas/projects/docs-register-and-track/.env`. | Для build blocker не подтвержден; для release ops path должен быть зафиксирован отдельно. |
| E.01.158 | ok | none | build | `.env`, `internal/releaseassets/current_release.yaml` | `build-linux`/`build-windows` fail fast without `ENCRYPTION_KEY`; `release-assets-check` verifies generated asset and Wails version freshness without rewriting files. | Release key remains externally supplied under `docs/secret_policy.md`. |
| E.01.159 | needs_info | major | install/build | `build/bin`, Windows installer | Embedded assets/migrations есть; installer содержит binary, WebView2 bootstrapper path зависит от Wails NSIS build. Нет целевой ОС install smoke. | Проверить Windows installer и Linux portable на целевых ОС: ресурсы, иконка, WebView2, config path, запуск из standard path. |

## Issues

Связанные новые issues: none; fixed: `ISSUE-023`, `ISSUE-024`.

## Проверки После Исправлений

- Clean checkout build на Linux и Windows builder.
- `npm ci`, `npm run build`, `go test ./...`, `wails build` в одном release script.
- Проверка, что About version, binary metadata и installer DisplayVersion совпадают.
- Проверка отсутствия незакоммиченного diff в generated release assets после build.
