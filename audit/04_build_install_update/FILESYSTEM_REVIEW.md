# Filesystem Review

Дата аудита: 2026-05-28
Этап: E.04, E.05

## Общий Вывод

Runtime filesystem usage is limited and mostly predictable:

- local release/theme state is stored in `os.UserConfigDir()/docflow/*.json` with `0600` files;
- attachment downloads are saved to user's `Downloads`;
- `OpenFile`/`OpenFolder` validates that paths stay under Downloads;
- attachment storage object names are UUID-based, reducing path traversal risk in MinIO.

Open risks remain around config placement and secret handling. Temp cleanup in backup/restore scripts and download filename collisions were fixed after the audit.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.04.169 | ok | none | runtime/install | `config.GetDefaultConfigPath()` | Main resolves config via `DOCFLOW_CONFIG_PATH`, executable-relative `config/config.json`, then cwd fallback for local development. | App no longer depends only on cwd-specific config; missing/invalid config now produces startup diagnostics. |
| E.04.170 | ok | none | ops | `backup_smb_tar.sh`, `restore_smb_tar.sh` | Temp dirs are created via `mktemp -d`, restricted with `chmod 700`, and removed by cleanup trap on any exit. | Manual interruption smoke remains part of release-gate validation on the target contour. |
| E.04.171 | ok | none | runtime | `DownloadToDisk`, local state | State files use `0600`; download files use `0644`; `docs/secret_policy.md` requires production `.env`/`config.json`/CIFS credentials permission checks. | File permission enforcement is release policy/smoke; attachment downloads remain ordinary user files. |
| E.04.172 | ok | none | build/runtime | `config.example.json`, `crypto.go`, Makefile | `docs/secret_policy.md` defines encrypted config preference, plaintext break-glass acceptance, key delivery, artifact sensitivity and rotation. | Keep release checklist evidence for permissions, process-list and logs/artifact scans. |
| E.04.173 | ok | none | runtime/ops | logs/scripts | Technical logs now use `app_user_id` without full names; binding error logs do not include full error text; SMB password mount exposure fixed via CIFS credentials file support. | Keep process-list/log secret exposure smoke in release gate. |
| E.04.174 | fixed | none | runtime | startup/file/storage errors | Startup config/DB/MinIO/release/theme/Wails failures use `startupdiag` with structured logs and operator-readable diagnostics; runtime binding errors are mapped through the structured error envelope. | Keep target OS smoke evidence for startup diagnostics and file operations. |
| E.05.175 | not_applicable | none | runtime | application export | Dedicated domain data export not found; attachment download returns original file content. | Reassess if report/export feature appears. |
| E.05.176 | ok | none | runtime | `AttachmentService.Upload` | Upload validates UUID document, access, base64 decode, max size, extension allowlist. | Backend validates core attachment input. |
| E.05.177 | ok | none | runtime | `Upload` | Max file size defaults to 15 MB and setting-driven. | Large-file risk bounded by configured max. |
| E.05.178 | issue | minor | runtime | `Upload` filename | Filename encoding/invalid characters are not normalized beyond extension/base path on download. | Normalize/sanitize filename for download display and OS compatibility. |
| E.05.179 | ok | none | runtime | MinIO object names | Storage object name is `uuid + ext`, not user path. | Import/upload does not write user-controlled paths into storage path. |
| E.05.180 | ok | none | runtime | `DownloadToDisk`, `OpenFile`, `OpenFolder` | `filepath.Base` prevents traversal on write; open path validates Downloads; `DownloadToDisk` now writes with `O_EXCL` and adds suffixes like `file (1).ext` on collision. | Covered by `TestWriteDownloadFileWithoutOverwrite`; manual file open/folder smoke remains useful. |

## Issues

Связанные новые issues: none; fixed: `ISSUE-028`, `ISSUE-029`, `ISSUE-030`, `ISSUE-031`.
Связанные ранее открытые: none; fixed: `ISSUE-002`, `ISSUE-016`.

## Проверки После Исправлений

- Upload filename with `../`, absolute path, reserved Windows names, long Unicode name.
- Download same filename twice; ensure no silent overwrite. Helper-level regression is covered by `TestWriteDownloadFileWithoutOverwrite`.
- OpenFile/OpenFolder with symlink, `..`, path outside Downloads.
- Backup/restore interruption leaves no stale temp data; script cleanup is implemented, target-contour interruption smoke remains.
- Verify `config.json` permissions and absence of secrets in logs/process args.
