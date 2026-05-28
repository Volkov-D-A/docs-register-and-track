# Tables Filters Pagination Review

Дата аудита: 2026-05-28
Этап: D.05

## Вывод

Документные списки используют общий `DocumentListTable` с AntD `Table`, `rowKey="id"`, `tableLayout="fixed"`, server-side pagination и `pageSizeOptions ['10','20','50']`. Фильтры сбрасывают page на 1, а `useDocumentListPage` защищает от устаревших responses через `requestIdRef`.

## Замечания

- Ошибки загрузки списков показываются через raw `message.error(err?.message || String(err))`, без code-specific UX.
- Search запускается через `Input.Search.onSearch`, то есть не делает запрос на каждый символ; это нормально для текущего объема.
- Общий `DocumentListTable` типизирован через `any[]` и `any[] columns`; это acceptable для текущего AntD слоя, но снижает compile-time protection при изменении DTO.

## Рекомендации

- Перевести list error handling на единый frontend error adapter.
- На этапе F проверить latency списков на production-like данных с фильтрами и доступами.
- Если DTO начнут часто меняться, типизировать columns/data по document-kind models.

Связанные issues: `ISSUE-019`, `ISSUE-004`, `RISK-011`.
