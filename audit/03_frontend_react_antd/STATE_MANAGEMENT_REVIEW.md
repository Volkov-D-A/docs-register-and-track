# State Management Review

Дата аудита: 2026-05-28
Этап: D.03

## Вывод

Глобальное состояние используется умеренно:

- `useAuthStore` хранит пользователя, auth loading/error и методы auth/profile;
- `useDraftLinkStore` хранит short-lived intent создания связанного документа;
- `useRegisterDocumentStore` хранит запрос открыть регистрацию конкретного вида;
- `useDocumentListPage` локально управляет списком, пагинацией, поиском, loading и view modal.

Хороший признак: в `useDocumentListPage` есть `requestIdRef`, который защищает список от устаревших async responses.

## Issues

### Frontend error state depends on raw strings

`useAuthStore.formatAuthError` ищет lockout по русскому тексту backend error. Остальные catch handlers в основном вызывают `message.error(err?.message || String(err))`. После backend structured errors frontend должен читать `code/message`, а не разбирать текст.

Связано: `ISSUE-019`, `DECISION-007`.

### Duplicate profile classification

`resolveUserProfile` во frontend остается отдельной реализацией profile classification. Это уже зафиксировано как `ISSUE-003`: access/profile поведение должно приходить из backend access summary либо иметь явный contract test.

## Рекомендации

- Добавить единый `formatAppError(err)` или hook `useAppError()`.
- Перевести auth lockout, forbidden, validation, not_found и conflict на `err.code`.
- Оставить Zustand stores небольшими; не переносить page-local form/filter state в global store без необходимости.
