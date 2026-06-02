# UX Text Review

Дата аудита: 2026-05-28
Этап: H. UX-тексты, терминология, ошибки, empty states

## Общий Вывод

Интерфейс в целом говорит на понятном русском языке и использует привычные термины предметной области: документы, поручения, ознакомления, номенклатура, регистрационный номер, вложения, журнал. Основные пользовательские действия узнаваемы.

Главные production-риски этапа H не в стиле, а в точности и безопасности микротекстов:

- safe error copy is fixed after `ISSUE-044`; smoke coverage is maintained after `ISSUE-043`;
- destructive confirmations are strengthened after `ISSUE-046`; smoke coverage is maintained after `ISSUE-043`;
- terminology consistency is fixed after `ISSUE-045`;
- common abbreviations/placeholders are clarified after `ISSUE-048`;
- empty states are action-aware after `ISSUE-047`; smoke coverage is maintained after `ISSUE-043`;
- microcopy style consistency is fixed for audited examples after `ISSUE-049`.

## Контрольные Пункты H

| Код | Статус | Severity | Экран/компонент | Текущий текст | Проблема | Предложение |
| --- | --- | --- | --- | --- | --- | --- |
| H.01.221 | ok | none | pages/layout | Page titles mostly Title level 4 / modal titles. | Стиль заголовков в целом единый. | Сохранить pattern. |
| H.01.222 | fixed | minor | buttons | `Откатить последнюю`, `Исполнено`, `Отметить`, `Проср.` | Не все кнопки начинались с ясного действия. | Fixed across `ISSUE-046`, `ISSUE-048`, `ISSUE-049`. |
| H.01.223 | ok | none | document pages | `Зарегистрировать`, primary buttons. | Primary action generally visible. | Нет. |
| H.01.224 | issue | minor | migration/storage actions | destructive buttons visually danger, but copy can be stronger. | Secondary/destructive actions need clearer consequence text. | See `MICROCOPY_REVIEW.md`. |
| H.02.225 | fixed | major | forms/cards | `Вид документа`, backend `тип документа`, UI `Вид`, glossary `Тип документа`. | Domain terms conflict. | Fixed in `ISSUE-045`: document type fields/cards use `Тип документа`; document family filters keep `Вид документа`. |
| H.02.226 | fixed | major | document forms/view | `Дело`, `Номенклатура`, `Все дела`. | Same entity named differently. | Fixed in `ISSUE-045`: user-facing forms/cards/statistics use `Дело`; admin settings keep `Номенклатура`. |
| H.02.227 | fixed | minor | forms | `Исполнитель` can mean поручение executor, sender executor, resolution executor. | Different entities share same label. | Fixed in `ISSUE-045`: `Ответственный исполнитель`, `Исполнитель письма`, `Исполнитель резолюции`. |
| H.02.228 | fixed | minor | icon buttons | Some icon buttons have tooltips, some not. | Meaning was not always discoverable. | Fixed in `ISSUE-048` for key lists/actions; keep accessibility smoke in `docs/ux_safety_smoke.md`. |
| H.02.229 | ok | none | forms | Most inputs have labels. | Placeholder not replacing labels except login fields where icons/title context exists. | Acceptable. |
| H.02.230 | fixed | minor | filters/forms | `Поиск...`, `ФИО`, `Проср.` | Placeholders/examples were often too vague. | Fixed in `ISSUE-048` for known generic search/date labels. |
| H.02.231 | fixed | minor | forms | Help texts sparse. | Неочевидные fields like `ПОС`, manual numbering, file type list needed short help. | Fixed in `ISSUE-048` for `Платформа обратной связи` and registration-number labels. |
| H.03.232-H.03.234 | issue | major | errors | `err?.message || String(err)`, `Ошибка загрузки...` | Errors may be technical and not actionable. | Structured safe messages by code; include next step. |
| H.03.235 | fixed | minor | success messages | `Удалено`, `Статус обновлен`. | Too generic. | Fixed in `ISSUE-049` with action-specific messages. |
| H.03.236-H.03.237 | fixed | major | destructive actions | `Удалить?`, `Удалить файл?`, rollback confirm. | Confirmation often lacked entity/consequence. | Fixed in `ISSUE-046`; keep smoke coverage in `docs/ux_safety_smoke.md`. |
| H.03.238 | fixed | minor | Empty states | `Нет прикрепленных файлов`, charts with empty AntD Empty. | Empty states had no next step. | Fixed in `ISSUE-047`; keep smoke coverage in `docs/ux_safety_smoke.md`. |
| H.03.239-H.03.240 | issue | major | startup/errors/audit | raw backend messages, IDs in admin audit details. | Technical jargon/IDs can leak to user-facing surfaces. | Keep IDs in technical audit only; UI messages safe. |
| H.04.241-H.04.244 | fixed | minor | many | `обновлен`, `завершен`, `dirty`, `N/A`. | Inconsistent Russian style and English leakage. | Fixed in `ISSUE-049` for audited UI strings. |
| H.04.245-H.04.247 | fixed | minor | forms/status | `ПОС`, `Рег. №`, dates. | Abbreviations were not expanded; dates mostly ok. | Fixed in `ISSUE-048`; keep `DD.MM.YYYY`. |
| H.05.248-H.05.256 | issue | minor | controls/forms | Switches mostly ok; some defaults and validation copy inconsistent. | Labels/help should match validation. | See form/microcopy reviews. |

## Приоритет

Major before production or as part of D/E remediation:

- structured safe error copy;
- destructive confirmation smoke coverage;
- terminology split `вид документа` / `тип документа` is fixed in audited frontend flows;
- startup/migration/system messages without technical jargon.

Minor can be batched:

- `е/ё` consistency;
- terminology polish;
- empty state next steps;
- tooltip completeness.

Связанные issues: `ISSUE-044`-`ISSUE-049`.
