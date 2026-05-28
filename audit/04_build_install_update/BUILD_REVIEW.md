# Build Review

Дата аудита: 2026-05-28
Этап: E.01. Production build и версия приложения

## Общий Вывод

Wails production build описан через `wails.json` и `Makefile`: frontend собирается командой `npm run build`, release assets генерируются через `go generate ./internal/releaseassets`, Linux/Windows binaries собираются Wails targets. Embedded frontend assets и embedded migrations снижают зависимость runtime от исходных директорий.

Основные build risks:

- версия приложения не имеет единого источника истины: About modal берет `docs/releases.yaml`/`current_release.yaml`, а Wails metadata использует `{{.Info.ProductVersion}}`;
- production build зависит от untracked `.env`/`ENCRYPTION_KEY` и встраивает encryption key через `ldflags`;
- `frontend:install` использует `npm install`, а не более воспроизводимый `npm ci`;
- build/release gate не зафиксирован как единый script с clean-machine checks.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.01.154 | issue | major | build | `Makefile`, `wails.json` | `Makefile` читает `.env`; `LDFLAGS` включает `ENCRYPTION_KEY`; `wails.json` делает `frontend:install: npm install`. | Clean build требует внешнего секрета и не имеет documented release script. Нужен release build contract: required env, `npm ci`, clean checkout, generated release assets, `go test`, frontend build, Wails build. |
| E.01.155 | issue | major | build/update | `docs/releases.yaml`, `internal/releaseassets/current_release.yaml`, Wails metadata | About version сейчас `1.0.4`, Wails metadata version идет из `{{.Info.ProductVersion}}`, но `wails.json` не фиксирует product version. | Ввести единый version source и генерацию About/Wails metadata из него. |
| E.01.156 | ok | none | runtime | `AboutProgramModal.tsx`, `ReleaseNoteService` | About modal показывает `release?.version` и дату релиза из embedded release note. | Версия видна пользователю, но связана с release notes, а не гарантированно с binary metadata. |
| E.01.157 | issue | minor | build/ops | backup/restore scripts | Build itself uses relative paths, but ops scripts embed `/home/dimas/projects/docs-register-and-track/.env`. | Для build blocker не подтвержден; для release ops path должен быть зафиксирован отдельно. |
| E.01.158 | issue | major | build | `.env`, `internal/releaseassets/current_release.yaml` | `.env` не tracked; current release генерируется перед build, но может быть stale без `make release-assets`. | Release command должен fail-fast при отсутствующем `ENCRYPTION_KEY` и проверять generated release asset freshness. |
| E.01.159 | needs_info | major | install/build | `build/bin`, Windows installer | Embedded assets/migrations есть; installer содержит binary, WebView2 bootstrapper path зависит от Wails NSIS build. Нет целевой ОС install smoke. | Проверить Windows installer и Linux portable на целевых ОС: ресурсы, иконка, WebView2, config path, запуск из standard path. |

## Issues

Связанные новые issues: `ISSUE-023`, `ISSUE-024`.

## Проверки После Исправлений

- Clean checkout build на Linux и Windows builder.
- `npm ci`, `npm run build`, `go test ./...`, `wails build` в одном release script.
- Проверка, что About version, binary metadata и installer DisplayVersion совпадают.
- Проверка отсутствия незакоммиченного diff в generated release assets после build.
