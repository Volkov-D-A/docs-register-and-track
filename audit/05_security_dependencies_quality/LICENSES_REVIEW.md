# Licenses Review

Дата аудита: 2026-05-28
Этап: F.02.185

## npm License Snapshot

Local package-lock scan:

- MIT: 176 packages.
- ISC: 27 packages.
- MIT AND ISC: 1 package.
- BSD-3-Clause: 4 packages.
- Apache-2.0: 2 packages.
- MPL-2.0: 12 packages.
- 0BSD: 1 package.
- UNKNOWN: 1 package: `node_modules/@antv/g2-extension-plot` `0.2.2`.
- No GPL/AGPL/LGPL license strings found in `package-lock.json`.

## Go License Status

No Go license inventory tool is configured in the repository. `go.mod` includes common permissive ecosystem dependencies, but a formal license report for all direct and transitive Go modules has not been generated.

## Вывод

No obvious copyleft blocker was found in npm lockfile, but production redistribution is not fully cleared until:

- unknown npm license is resolved;
- Go transitive license report is generated;
- release artifact includes required notices if applicable.

Связанные issues: `ISSUE-037`.
