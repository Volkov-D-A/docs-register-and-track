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

Fixed after audit: `useAuthStore.formatAuthError` now reads structured code `USER_LOCKED` through `getAppErrorCode`, and frontend catch handlers use `formatAppError`/`normalizeAppError` for Wails `{code,message,status}` instead of raw string parsing.

Связано: fixed `ISSUE-019`, `DECISION-007`.

### Duplicate profile classification

`resolveUserProfile` во frontend остается UI-only profile classification. `ISSUE-003` fixed after audit: backend dashboard no longer duplicates UX profile labels and computes assignment scope directly.

## Рекомендации

- Keep new catch handlers on `formatAppError(err)`; do not reintroduce raw `err?.message || String(err)`.
- Cover auth lockout, forbidden, validation, not_found and conflict in frontend smoke/e2e work.
- Оставить Zustand stores небольшими; не переносить page-local form/filter state в global store без необходимости.
