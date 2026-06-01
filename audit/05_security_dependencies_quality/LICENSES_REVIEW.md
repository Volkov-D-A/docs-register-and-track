# Licenses Review

Дата аудита: 2026-05-28
Этап: F.02.185

## License Gate Status

After remediation, `make release-gate` runs `node tools/license-report.js` through `npm-license-check` and `license-inventory`.

The tool:

- scans `frontend/package-lock.json`;
- resolves the `@antv/g2-extension-plot@0.2.2` missing metadata through an explicit MIT override backed by the package `LICENSE` file;
- downloads and scans all Go modules from `go list -m all`;
- blocks unknown licenses unless explicitly overridden with evidence;
- blocks GPL/LGPL/AGPL family licenses unless a documented exception is approved;
- writes the release notice/inventory to `build/release-evidence/LICENSE_REPORT.md`.

Latest local result:

- npm packages checked: 349.
- Go modules checked: 316.
- Unknown licenses: 0.
- Disallowed licenses: 0.

## Вывод

No unknown or disallowed dependency licenses remain in the current inventory. The generated `build/release-evidence/LICENSE_REPORT.md` must be archived with release evidence and included with release notices/artifacts as applicable.

Связанные issues: fixed `ISSUE-037`.
