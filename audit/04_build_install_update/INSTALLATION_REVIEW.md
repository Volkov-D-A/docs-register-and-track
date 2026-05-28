# Installation Review

Дата аудита: 2026-05-28
Этап: E.02. Установка, запуск и права

## Общий Вывод

Для Windows есть NSIS installer template. По умолчанию он запрашивает admin privileges, ставит приложение в `$PROGRAMFILES64`, пишет uninstall info в HKLM и создает shortcuts для всех пользователей. Для Linux есть portable binary в `build/bin`, отдельного installer/service package не найдено.

Главный installation риск: runtime config ищется как относительный путь `config/config.json`. Это связывает запуск с текущей рабочей директорией и вручную подготовленным каталогом рядом с приложением.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.02.160 | needs_info | major | install/first run | Windows/Linux target OS | Локального installer smoke на целевых ОС в этом проходе не было. | Выполнить запуск после установки в Program Files/portable path и проверить config resolution. |
| E.02.161 | needs_info | major | install/runtime | target OS paths | Код использует `filepath`, но запуск из пути с пробелами/кириллицей не проверен. | Добавить smoke: `C:\Program Files\...`, каталог с кириллицей, Linux path с пробелом. |
| E.02.162 | issue | major | install | `build/windows/installer/wails_tools.nsh` | Default `REQUEST_EXECUTION_LEVEL` = `admin`; installer пишет HKLM и Program Files. | Нужна явная production policy: per-machine admin install или per-user install. App после установки должен работать без admin. |
| E.02.163 | issue | major | first run/runtime | `config.GetDefaultConfigPath()` | Config path — `filepath.Join("config","config.json")` относительно cwd. Ошибка загрузки вызывает `log.Fatalf`. | При отсутствии доступа/файла приложение аварийно завершится до UI. Нужен predictable config lookup и user-facing startup error. |

## Issues

Связанные новые issues: `ISSUE-025`, `ISSUE-026`.

## Проверки После Исправлений

- Установка/запуск Windows без elevated app process после install.
- Запуск из shortcut, прямой запуск exe, запуск с другой рабочей директорией.
- Запуск из пути с пробелами и кириллицей.
- Поведение при отсутствующем, недоступном или поврежденном `config.json`.
