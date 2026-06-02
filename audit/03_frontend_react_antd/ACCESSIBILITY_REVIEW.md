# Accessibility Review

Дата аудита: 2026-05-28
Этап: D.06

## Вывод

Ant Design дает базовую доступность для форм, модалок, таблиц и кнопок. В `MainLayout` есть явный `aria-label` для кнопки сворачивания боковой панели. Явных кастомных control elements без кнопок в основных формах не найдено.

## Риски

- После backend/list errors UI в основном показывает transient toast через `message.error`; нет явного focus management или inline error summary для форм.
- Document/settings modals now ask before discarding touched fields; manual keyboard/focus smoke remains required.
- Link graph и визуальные графики требуют отдельного keyboard/fallback smoke; interaction выглядит pointer-oriented.

## Рекомендации

- Проверить keyboard-only сценарии: login, navigation, registration modal, edit modal, file upload/download, assignments, migration controls.
- Добавить focus recovery после закрытия модалки и после критичных ошибок.
- Для graph/report visualizations предусмотреть табличный/текстовый fallback там, где граф нужен для принятия решения.

Связанные issues: fixed `ISSUE-019`, fixed `ISSUE-021`; remaining UX smoke coverage is tracked by `ISSUE-043`.
