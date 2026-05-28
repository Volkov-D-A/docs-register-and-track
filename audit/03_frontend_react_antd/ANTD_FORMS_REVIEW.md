# AntD Forms Review

Дата аудита: 2026-05-28
Этап: D.04

## Вывод

Основные формы построены на Ant Design `Form`, `DatePicker`, `Select`, `InputNumber`, `Switch`. Для документных форм есть required validation и numeric min constraints для количества листов. После исправления этапа C формы регистрации отправляют `idempotencyKey`.

## Найденные Риски

### Повторный submit не везде заблокирован UI-состоянием

Document registration теперь защищена backend idempotency, но UX guard неполный. Например, `IncomingPage` передает `onOk: () => registerForm.submit()` без `confirmLoading`; другие document pages местами используют `confirmLoading: loading`, который относится к загрузке списка, а не к конкретному submit. В settings/actions картина неоднородная.

Связано: `ISSUE-020`.

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
