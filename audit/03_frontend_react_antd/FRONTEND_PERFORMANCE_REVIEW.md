# Frontend Performance Review

Дата аудита: 2026-05-28
Этап: D.07

## Вывод

Original audit `npm run build` passed but Vite warned about a large main chunk around 3001 kB, gzip around 872 kB. After `ISSUE-004`, page components are lazy-loaded and the main app chunk is around 34 kB. For Wails desktop this is still connected to production SLO: startup to login <= 5 seconds and memory <= 512 MB.

## Наблюдения

- Route/page components импортируются статически в `App.tsx`.
- Page components are lazy-loaded after remediation of `ISSUE-004`; the main app chunk is no longer the 3 MB bundle.
- The largest lazy chunk is `StatisticsPage` because it includes charting libraries; this is accepted under an explicit Wails desktop route-chunk warning budget, but still needs startup/memory baseline on target builds.

## Рекомендации

- На этапе F измерить фактический startup и memory в Wails build, а не только web bundle size.
- Если budget не выполняется, ввести route-level `React.lazy` для тяжелых страниц: settings, statistics, document view/link graph.
- Проверить, что code splitting не ломает Wails embedded assets и first-run organization setup.

Связанные issues: fixed `ISSUE-004`, `RISK-007`.
