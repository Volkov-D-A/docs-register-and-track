# Production Smoke Test

Дата обновления: 2026-06-02
Статус: maintained minimal smoke

Run this smoke test against the release artifact produced from a clean checkout.

## Preconditions

- PostgreSQL, MinIO and Seq are available in the approved closed LAN/test contour.
- Test database and MinIO bucket are disposable or freshly restored from release test backup.
- Operator has one admin user and at least one document participant user.
- Config is supplied through `DOCFLOW_CONFIG_PATH` or executable-relative `config/config.json`.

## Startup

- [ ] Start app from target install path.
- [ ] Start app from shortcut/default working directory.
- [ ] Start app from a path with spaces and Cyrillic characters.
- [ ] Login screen appears within performance budget.
- [ ] Missing/invalid config scenario gives actionable diagnostics.

## Authentication And Settings

- [ ] Login as admin.
- [ ] Open settings.
- [ ] Verify organization settings are visible.
- [ ] Verify migration status screen opens without error.
- [ ] Verify non-admin user cannot access admin-only settings.

## Document Registration

- [ ] Register one incoming letter.
- [ ] Register one outgoing letter.
- [ ] Register one citizen appeal.
- [ ] Register one administrative order.
- [ ] Repeat submit/double click is blocked or returns the same document without number gap.
- [ ] Verify registration numbers are contiguous for the used nomenclature.

## Lists, Search And Access

- [ ] Open each document registry.
- [ ] Search/filter by number/date/text where available.
- [ ] Open document card from list.
- [ ] Login as limited user and verify access scope is not wider than expected.

## Files

- [ ] Upload allowed attachment under configured max size.
- [ ] Download attachment.
- [ ] Try duplicate filename download and verify overwrite/collision behavior is acceptable.
- [ ] Delete attachment only where user has permission.
- [ ] Try forbidden file action and verify safe user error text.

## Assignments And Acknowledgments

- [ ] Create assignment for a document.
- [ ] Complete or return assignment according to available role.
- [ ] Create acknowledgment for administrative order.
- [ ] Confirm acknowledgment as target user.

## Errors And Safety

- [ ] Trigger validation error and verify user-safe message.
- [ ] Trigger forbidden action and verify user-safe message.
- [ ] Trigger not-found/missing document scenario if possible.
- [ ] Verify destructive confirmations name affected entity and consequence.
- [ ] Try closing dirty registration/edit modal and verify warning or accepted known issue.

## Backup/Restore Evidence

- [ ] Backup archive exists for this test contour.
- [ ] Manual test restore PostgreSQL+MinIO completed.
- [ ] Restore report and smoke result saved with release evidence.

## Final

- [ ] App exits cleanly.
- [ ] No unexpected panic/fatal exit.
- [ ] Seq/technical logs do not expose secrets or unnecessary PII.
- [ ] `document_journal` and `admin_audit_log` contain expected domain audit entries.

