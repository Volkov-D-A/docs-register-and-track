# Filesystem Review

Дата аудита: 2026-05-28
Этап: E.04, E.05

## Общий Вывод

Runtime filesystem usage is limited and mostly predictable:

- local release/theme state is stored in `os.UserConfigDir()/docflow/*.json` with `0600` files;
- attachment downloads are saved to user's `Downloads`;
- `OpenFile`/`OpenFolder` validates that paths stay under Downloads;
- attachment storage object names are UUID-based, reducing path traversal risk in MinIO.

Open risks remain around config placement, download filename collisions, temp cleanup in backup/restore scripts and secret handling.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.04.169 | issue | major | runtime/install | `config.GetDefaultConfigPath()` | Main loads `config/config.json` relative cwd; production policy says manual config path. | App should not require writable install dir or cwd-specific config. Define system/user config path or executable-relative read-only config intentionally. |
| E.04.170 | issue | minor | ops | `backup_smb_tar.sh`, `restore_smb_tar.sh` | Temp dirs in `/tmp`; cleanup happens at end, but trap only unmounts SMB. | Add cleanup trap for temp dirs on failure/interruption. |
| E.04.171 | issue | minor | runtime | `DownloadToDisk`, local state | State files use `0600`; download files use `0644`; config permissions not checked. | Enforce/check `config.json` permissions if it contains plaintext or encrypted secrets. |
| E.04.172 | issue | major | build/runtime | `config.example.json`, `crypto.go`, Makefile | Plaintext secrets are accepted; encrypted secrets require key embedded in binary or env. | Define production secret policy: plaintext forbidden/allowed only by exception, key delivery, rotation and no key in repo/artifacts. |
| E.04.173 | issue | major | runtime/ops | logs/scripts | Stage C found PII/business identifiers in logs; SMB password passed to `mount` options. | Continue `ISSUE-016`; use credentials file for CIFS and redaction/minimization. |
| E.04.174 | issue | major | runtime | startup/file/storage errors | Startup config/DB failures use `log.Fatalf`/`os.Exit`; file errors include internal paths/details. | Map startup/runtime errors to safe operator messages and logs. |
| E.05.175 | not_applicable | none | runtime | application export | Dedicated domain data export not found; attachment download returns original file content. | Reassess if report/export feature appears. |
| E.05.176 | ok | none | runtime | `AttachmentService.Upload` | Upload validates UUID document, access, base64 decode, max size, extension allowlist. | Backend validates core attachment input. |
| E.05.177 | ok | none | runtime | `Upload` | Max file size defaults to 15 MB and setting-driven. | Large-file risk bounded by configured max. |
| E.05.178 | issue | minor | runtime | `Upload` filename | Filename encoding/invalid characters are not normalized beyond extension/base path on download. | Normalize/sanitize filename for download display and OS compatibility. |
| E.05.179 | ok | none | runtime | MinIO object names | Storage object name is `uuid + ext`, not user path. | Import/upload does not write user-controlled paths into storage path. |
| E.05.180 | issue | major | runtime | `DownloadToDisk`, `OpenFile`, `OpenFolder` | `filepath.Base` prevents traversal on write; open path validates Downloads. But download overwrites same filename in Downloads. | Add collision-safe save names or explicit overwrite confirmation. |

## Issues

Связанные новые issues: `ISSUE-029`, `ISSUE-030`, `ISSUE-031`.
Связанные ранее открытые: `ISSUE-002`, `ISSUE-016`.

## Проверки После Исправлений

- Upload filename with `../`, absolute path, reserved Windows names, long Unicode name.
- Download same filename twice; ensure no silent overwrite.
- OpenFile/OpenFolder with symlink, `..`, path outside Downloads.
- Backup/restore interruption leaves no stale temp data.
- Verify `config.json` permissions and absence of secrets in logs/process args.
