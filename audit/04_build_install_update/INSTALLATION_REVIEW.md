# Installation Review

Дата аудита: 2026-05-28
Этап: E.02. Установка, запуск и права

## Общий Вывод

Для Windows есть NSIS installer template. По умолчанию он запрашивает admin privileges, ставит приложение в `$PROGRAMFILES64`, пишет uninstall info в HKLM и создает shortcuts для всех пользователей. Для Linux есть portable binary в `build/bin`, отдельного installer/service package не найдено.

Главный installation риск после remediation: target OS smoke. Runtime config lookup now supports `DOCFLOW_CONFIG_PATH`, executable-relative config and local cwd fallback; Windows per-machine admin install policy is documented.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.02.160 | needs_info | major | install/first run | Windows/Linux target OS | Локального installer smoke на целевых ОС в этом проходе не было. | Выполнить запуск после установки в Program Files/portable path и проверить config resolution. |
| E.02.161 | needs_info | major | install/runtime | target OS paths | Код использует `filepath`, но запуск из пути с пробелами/кириллицей не проверен. | Добавить smoke: `C:\Program Files\...`, каталог с кириллицей, Linux path с пробелом. |
| E.02.162 | ok | none | install | `build/windows/installer/project.nsi`, `docs/install_policy.md` | Per-machine Windows install policy accepted: installer requires admin, installs under Program Files, writes HKLM uninstall metadata and creates all-user shortcuts. | Target OS smoke must verify the installed app runs for ordinary users without elevation. |
| E.02.163 | ok | none | first run/runtime | `config.GetDefaultConfigPath()` | Config lookup order: `DOCFLOW_CONFIG_PATH`, executable-relative `config/config.json`, cwd fallback for local development. Startup diagnostics include resolved path. | Target OS smoke must verify missing/invalid config. |

## Issues

Связанные новые issues: none; fixed: `ISSUE-025`, `ISSUE-026`.

## Проверки После Исправлений

- Установка/запуск Windows без elevated app process после install.
- Запуск из shortcut, прямой запуск exe, запуск с другой рабочей директорией.
- Запуск из пути с пробелами и кириллицей.
- Поведение при отсутствующем, недоступном или поврежденном `config.json`.
