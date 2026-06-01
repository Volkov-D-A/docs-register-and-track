# AntD Forms Review

Дата аудита: 2026-05-28
Этап: D.04

## Вывод

Основные формы построены на Ant Design `Form`, `DatePicker`, `Select`, `InputNumber`, `Switch`. Для документных форм есть required validation и numeric min constraints для количества листов. После исправления этапа C формы регистрации отправляют `idempotencyKey`; after remediation, critical submit/actions have local loading guards.

## Найденные Риски

### Повторный submit не везде заблокирован UI-состоянием

Fixed after audit: document registration/edit modals now use local register/edit submitting guards and `confirmLoading`; assignment/acknowledgment completion, file upload/delete, settings CRUD, migration and storage actions have repeat-click guards.

Связано: `ISSUE-020` fixed; manual double-click smoke remains under `ISSUE-043`.

### Unsaved changes guard отсутствует

`DocumentKindPage` закрывает registration/edit modals через переданный `onCancel` без проверки dirty fields. Пользователь может потерять введенные данные при закрытии модалки или смене flow.

Связано: `ISSUE-021`.

### Валидация неодинакова

Часть `rules` содержит только `{ required: true }`, часть добавляет `whitespace` и message. Empty-string/null normalization также различается между формами.

## Рекомендации

- Ввести локальный `submitting` для каждого create/update/critical action.
- Добавить dirty confirmation для registration/edit/settings forms.
- Унифицировать form rules: required message, whitespace для строк, numeric min/max там, где есть backend invariant.
- При появлении backend `details` для validation мапить field errors в AntD `form.setFields`.
