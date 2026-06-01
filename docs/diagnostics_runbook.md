# Diagnostics Runbook

Дата обновления: 2026-06-01

## Purpose

This runbook gives operators and release testers a consistent path for startup, configuration, database, storage, logging and migration failures.

Current limitation: some startup errors still exit before an in-app diagnostics screen. That is tracked by `ISSUE-028`.

## First Checks

Collect:

- application version and artifact name;
- exact launch path and current working directory;
- config path used for launch;
- PostgreSQL, MinIO and Seq availability;
- latest technical logs from Seq or console output;
- current migration status if the app opens.

Never paste production passwords, tokens or full encrypted secret material into tickets.

Follow `docs/secret_policy.md` for secret delivery, config permissions and rotation.

## Config Failures

Config lookup order:

```text
DOCFLOW_CONFIG_PATH
<executable directory>/config/config.json
<current working directory>/config/config.json
```

The current working directory fallback is intended for local development.

Check:

- file exists at the expected resolved path;
- file is valid JSON;
- operator account can read the file;
- values are production-approved, not copied from local examples;
- encrypted `ENC:` values can be decrypted with the approved `ENCRYPTION_KEY`.

Common actions:

- set `DOCFLOW_CONFIG_PATH` to the approved production config or place it next to the executable under `config/config.json`;
- restore the approved production config from secure storage;
- check file permissions;
- rebuild/redeploy only if the approved encryption key path changed.

## PostgreSQL Failures

Check:

- host and port are reachable from the workstation;
- database name, user and SSL mode match production documentation;
- password decrypts correctly;
- PostgreSQL service is running;
- `schema_migrations` status is not dirty.

If migration status is dirty:

1. Stop normal application use.
2. Preserve logs and DB state.
3. Do not retry destructive migration operations blindly.
4. Use latest PostgreSQL+MinIO backup if recovery requires restore.
5. Record migration version and error before remediation.

## Newer Schema Or Downgrade Failure

If the app reports that DB schema is newer than the embedded migrations:

- stop using this binary with that DB;
- install the compatible newer binary, or follow an approved downgrade procedure on a disposable/restored database;
- do not run rollback as a substitute for version compatibility unless a release owner approves it and a fresh backup exists.

The backend blocks login and migration operations for newer/dirty schema states.

## MinIO Failures

Check:

- endpoint and TLS mode;
- access key and secret key;
- bucket name;
- network route from workstation/server;
- object store capacity and permissions.

For attachment inconsistencies between PostgreSQL metadata and MinIO objects, restore PostgreSQL and MinIO from the same backup set. Do not restore only one side unless a recovery plan explicitly allows it.

## Seq Failures

Seq is used for technical logs and is not included in the PostgreSQL+MinIO backup set.

Check:

- Seq URL;
- service availability;
- network route;
- credentials or first-run admin password if applicable.

If Seq is unavailable, application behavior should continue where possible, but release smoke must record whether logs were delivered.

## Backup And Restore Failures

Use `docs/backup_restore_runbook.md`.

Important rules:

- restore must fail fast on PostgreSQL errors;
- MinIO mirror must not start until PostgreSQL restore and DB validation succeed;
- temp directories and temporary credentials files must be removed on interruption;
- restore report must be kept with release evidence.

## Migration Rollback Failures

Use `docs/migration_rollback_runbook.md`.

If rollback fails:

- stop application use;
- preserve logs;
- record backup reference, migration version and dirty state;
- prefer restoring PostgreSQL+MinIO from a consistent backup set over manual partial fixes.

## Release Diagnostics Smoke

Before release, verify and record behavior for:

- missing config file;
- invalid JSON config;
- unreadable config;
- wrong encryption key;
- PostgreSQL unavailable;
- MinIO unavailable;
- Seq unavailable;
- dirty migration state;
- newer-schema/downgrade attempt;
- failed restore before MinIO mirror.
