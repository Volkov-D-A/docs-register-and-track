# Secret Policy

Дата обновления: 2026-06-02

## Scope

This policy covers production handling for:

- `ENCRYPTION_KEY`;
- PostgreSQL password in `config.json`;
- MinIO secret access key in `config.json`;
- SMB credentials used by backup/restore scripts;
- generated release evidence and technical logs.

## Allowed Secret Delivery

Production secrets must not be committed to Git, pasted into issue trackers or stored in release evidence.

Approved delivery methods:

- `ENCRYPTION_KEY` is supplied by the release operator through the approved release environment or a local `.env` file with restricted permissions.
- PostgreSQL and MinIO secrets in production config should use `ENC:` values encrypted with the approved `ENCRYPTION_KEY`.
- Plaintext PostgreSQL/MinIO secrets are allowed only as a documented break-glass exception for a closed LAN contour and must be accepted by the release owner.
- SMB credentials must use `SMB_CREDENTIALS_FILE` where possible. Temporary credentials files created by scripts must be restricted and removed by cleanup traps.

The current production build embeds `ENCRYPTION_KEY` through Go ldflags. Treat built binaries and installers as sensitive release artifacts: restrict access, store only in the approved artifact location and rotate the key if an artifact is exposed outside the approved contour.

## File Permissions

Linux/portable contour:

- `.env`: `0600`, owned by the release/operator account.
- `config.json`: `0600` where possible, or stricter ACL equivalent if shared by service/desktop accounts.
- CIFS credentials file: `0600`.
- backup/restore temporary directories: `0700`.

Windows contour:

- production config and artifact directories must be readable only by approved users/operators and local administrators;
- ordinary users may launch the installed app but must not be able to modify production config unless this is explicitly approved.

## Rotation

Rotate `ENCRYPTION_KEY` when:

- a release artifact containing the key is exposed outside the approved contour;
- an operator with access leaves the release role;
- production config secrets are suspected to be exposed;
- the release owner requests periodic rotation.

Rotation procedure:

1. Generate a new approved 32-byte key.
2. Re-encrypt PostgreSQL and MinIO config values with the new key.
3. Rebuild artifacts through `make release-gate` and production build targets.
4. Replace production config and artifacts in the approved maintenance window.
5. Archive evidence that old artifacts/configs are retired.

## Release Checks

Before publishing a production candidate:

- confirm `.env`, `config.json` and CIFS credentials file permissions;
- run backup/restore smoke and confirm SMB password is not visible in process arguments;
- inspect release evidence and technical logs for passwords, tokens, full encrypted secret material and unexpected plaintext config values;
- confirm `docs/known_issues.md` records any accepted plaintext-secret exception.

