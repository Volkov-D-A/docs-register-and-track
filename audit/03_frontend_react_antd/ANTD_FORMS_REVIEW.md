# AntD Forms Review

Дата аудита: 2026-05-28
Этап: D.04

## Вывод

Основные формы построены на Ant Design `Form`, `DatePicker`, `Select`, `InputNumber`, `Switch`. Для документных форм есть required validation и numeric min constraints для количества листов. После исправления этапа C формы регистрации отправляют `idempotencyKey`; after remediation, critical submit/actions have local loading guards.

## Найденные Риски

### Повторный submit не везде заблокирован UI-состоянием

Fixed after audit: document registration/edit modals now use local register/edit submitting guards and `confirmLoading`; assignment/acknowledgment completion, file upload/delete, settings CRUD, migration and storage actions have repeat-click guards.

Связано: `ISSUE-020` fixed; manual double-click smoke remains under `ISSUE-043`.

### Unsaved changes guard

Fixed after audit: document registration/edit and important settings modals use shared dirty-form confirmation before discarding touched fields.

Связано: `ISSUE-021` fixed; manual dirty modal smoke remains under `ISSUE-043`.

### Валидация неодинакова

Часть `rules` содержит только `{ required: true }`, часть добавляет `whitespace` и message. Empty-string/null normalization также различается между формами.

## Рекомендации

- Keep repeat-submit and dirty-close smoke in release checklist.
- Унифицировать form rules: required message, whitespace для строк, numeric min/max там, где есть backend invariant.
- При появлении backend `details` для validation мапить field errors в AntD `form.setFields`.
