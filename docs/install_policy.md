# Install Policy

Дата обновления: 2026-06-02

## Windows

Production Windows installation is per-machine and requires administrator rights.

The NSIS installer:

- requests administrator execution level;
- installs under `Program Files`;
- writes uninstall metadata under `HKLM`;
- creates shortcuts for all users.

The installed application itself must run for ordinary users without an elevated process. Target OS smoke must verify:

- installer prompts for elevation;
- install completes under the approved administrator account;
- ordinary user can launch the app from Start menu/desktop shortcut;
- config is resolved through `DOCFLOW_CONFIG_PATH` or executable-relative `config/config.json`;
- uninstall/update behavior does not remove or overwrite the approved production config unless the release owner explicitly approves it.

## Linux

Current Linux delivery is a portable binary artifact. Target OS smoke must verify:

- artifact runs from the approved installation directory;
- launch from a path with spaces and Cyrillic characters;
- ordinary user can launch without elevated privileges;
- config is resolved through `DOCFLOW_CONFIG_PATH` or executable-relative `config/config.json`.

