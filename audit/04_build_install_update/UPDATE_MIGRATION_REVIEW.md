# Update Migration Review

Дата аудита: 2026-05-28
Этап: E.03. Обновления, миграции и crash safety

## Общий Вывод

Миграции embedded в binary и запускаются через `SettingsService.RunMigrations()` вручную из admin UI. Это делает установленный binary самодостаточным относительно migration files. Downgrade/newer-schema guard добавлен после remediation `ISSUE-027`, но lifecycle обновления приложения еще не полностью закрыт: startup/admin migration policy, dirty-state runbook и target-contour smoke остаются обязательными перед релизом.

Runtime rollback по решению `DECISION-003` остается production feature; guardrails for rollback and fatal restore handling are fixed as `ISSUE-007`/`ISSUE-010`.

## Контрольные Пункты

| Код | Статус | Severity | Lifecycle | Место | Доказательство | Проблема / рекомендация |
| --- | --- | --- | --- | --- | --- | --- |
| E.03.164 | issue | major | update | installer/runtime config | Update process не описан; config рядом с install/cwd может быть перезаписан или потерян в зависимости от install procedure. | Зафиксировать, где живет production `config.json`, и проверить update без потери config/local state. |
| E.03.165 | ok | none | update | `database.GetMigrationStatus`, `AuthService.Login`, migration UI | `MigrationStatus` distinguishes `schemaTooNew`; login and migration operations block newer/dirty schema with structured `CONFLICT`; UI shows `Схема новее приложения`. | Target-contour downgrade smoke remains a release check. |
| E.03.166 | issue | major | update | `SettingsService.RunMigrations` | Миграции запускаются вручную admin action; fresh/update startup gate не определен. | Нужна policy: auto-migrate on startup или explicit admin run before use, плюс backup/runbook. |
| E.03.167 | issue | major | update/runtime | frontend settings migration UI, `ErrorFormatter` | Backend теперь safe-maps errors, но startup config/DB failures still exit before UI. Migration failure UX требует smoke. | Проверить понятность ошибки migration failure и dirty DB state. |
| E.03.168 | issue | major | crash/update | migrations, registration, restore | Document registration transaction hardened; migration crash safety relies on golang-migrate dirty flag. Dirty handling/runbook не оформлены. | Добавить dirty migration recovery runbook and smoke; не продолжать работу при dirty schema. |

## Issues

Связанные новые issues: `ISSUE-028`.
Связанные ранее открытые: `ISSUE-015`; fixed after audit: `ISSUE-007`, `ISSUE-010`, `ISSUE-017`, `ISSUE-027`.

## Проверки После Исправлений

- Update old version -> current with existing DB and files.
- Attempt downgrade current DB -> older binary.
- Failed migration leaves dirty state; app blocks unsafe use and shows clear operator action.
- Rollback warning/confirmation/backup/audit entry.
- Crash during document registration does not break no-gaps/idempotency invariants.
