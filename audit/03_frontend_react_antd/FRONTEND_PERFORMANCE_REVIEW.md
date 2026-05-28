# Frontend Performance Review

Дата аудита: 2026-05-28
Этап: D.07

## Вывод

Предыдущий `npm run build` прошел успешно, но Vite предупредил о большом основном чанке около 3001 kB, gzip около 872 kB. Для Wails desktop это не автоматически blocker, но уже связано с production SLO: старт до экрана входа до 5 секунд и память до 512 MB.

## Наблюдения

- Route/page components импортируются статически в `App.tsx`.
- Есть dynamic imports для generated Wails service modules внутри handlers/effects, но они не решают общий route-level chunk.
- Большие страницы settings/statistics/document view увеличивают риск роста main bundle.

## Рекомендации

- На этапе F измерить фактический startup и memory в Wails build, а не только web bundle size.
- Если budget не выполняется, ввести route-level `React.lazy` для тяжелых страниц: settings, statistics, document view/link graph.
- Проверить, что code splitting не ломает Wails embedded assets и first-run organization setup.

Связанные issues: `ISSUE-004`, `RISK-007`.
