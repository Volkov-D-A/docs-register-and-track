# UX Safety Smoke Checklist

Дата обновления: 2026-06-02
Статус: maintained release smoke checklist

Run these checks on the built Wails application in the same target OS smoke session as `docs/smoke_test.md`. Use disposable PostgreSQL/MinIO data and attach the completed checklist to release evidence.

## Preconditions

- Operator has one admin user and one limited user.
- Test data includes at least one document with an attachment, assignment, acknowledgment and document link.
- Release candidate is built from a clean checkout and frontend production build smoke has passed.

## Error Recovery

- [ ] [UX-SAFE-ERROR-VALIDATION] Trigger a required-field or invalid-value save error and verify the message is user-safe, names the action and leaves the form editable.
- [ ] [UX-SAFE-ERROR-FORBIDDEN] Attempt an action as a user without permission and verify the error explains lack of access without exposing backend internals.
- [ ] [UX-SAFE-ERROR-NOT-FOUND] Open or refresh a deleted/missing document scenario and verify the UI offers a clear way back to the list.
- [ ] [UX-SAFE-ERROR-CONFLICT] Repeat a document registration submit or duplicate conflict scenario and verify the copy explains duplicate/conflict behavior without consuming a new number.
- [ ] [UX-SAFE-ERROR-INTERNAL] Simulate DB or MinIO outage on a disposable contour and verify the UI gives a safe operator action and can recover after services return.

## Destructive Confirmations

- [ ] [UX-SAFE-DEST-ROLLBACK] Open migration rollback and verify the confirmation requires backup reference, data-loss acknowledgment and control phrase before rollback is allowed.
- [ ] [UX-SAFE-DEST-FILE] Delete an attachment and verify the confirmation names the file and consequence.
- [ ] [UX-SAFE-DEST-LINK] Delete a document link and verify the confirmation names the relationship and consequence.
- [ ] [UX-SAFE-DEST-ASSIGNMENT] Delete or return/finish a critical assignment action and verify the confirmation names the assignment action.
- [ ] [UX-SAFE-DEST-ACK] Delete or complete an acknowledgment and verify the confirmation names the acknowledgment action.
- [ ] [UX-SAFE-DEST-REFERENCE] Delete a reference item in settings and verify the confirmation names the item and warns about historical data impact where applicable.

## Dirty Forms And Repeat Submit

- [ ] [UX-SAFE-DIRTY-INCOMING] Change an incoming document registration/edit form, close it, cancel the warning and verify changes remain.
- [ ] [UX-SAFE-DIRTY-OUTGOING] Change an outgoing letter registration/edit form, close it, confirm discard and verify the form resets.
- [ ] [UX-SAFE-DIRTY-ORDER] Change an administrative order registration/edit form and verify clean close does not warn while dirty close does.
- [ ] [UX-SAFE-DIRTY-APPEAL] Change a citizen appeal registration/edit form and verify the dirty warning appears before discard.
- [ ] [UX-SAFE-DIRTY-SETTINGS] Change an important settings form and verify discard confirmation protects unsaved changes.
- [ ] [UX-SAFE-REPEAT-SUBMIT] Double-click critical submit/save/upload/delete buttons and verify loading guards prevent duplicate operations or return the same idempotent result.

## Empty States And Microcopy

- [ ] [UX-SAFE-EMPTY-LISTS] Open empty document lists and verify copy explains whether to create data, change filters or request access.
- [ ] [UX-SAFE-EMPTY-DASHBOARD] Open empty assignment/acknowledgment dashboard cards and verify next-step copy is action-aware.
- [ ] [UX-SAFE-EMPTY-FILES] Open an empty files tab with and without upload permission and verify the copy matches the allowed action.
- [ ] [UX-SAFE-EMPTY-STATS] Open statistics with no data/no access and verify tables/charts use `Нет данных` or actionable copy.
- [ ] [UX-SAFE-MICROCOPY-TERMS] Spot-check visible terminology: `Тип документа`, `Дело`, `Ответственный исполнитель`, no user-visible `dirty` or `N/A`.

## Evidence

- [ ] Record target OS, artifact name, operator, date and data contour.
- [ ] Attach screenshots or notes for each failed scenario.
- [ ] Link accepted failures to `docs/known_issues.md`.
