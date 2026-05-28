# UX Text Review

Дата аудита: 2026-05-28
Этап: H. UX-тексты, терминология, ошибки, empty states

## Общий Вывод

Интерфейс в целом говорит на понятном русском языке и использует привычные термины предметной области: документы, поручения, ознакомления, номенклатура, регистрационный номер, вложения, журнал. Основные пользовательские действия узнаваемы.

Главные production-риски этапа H не в стиле, а в точности и безопасности микротекстов:

- ошибки часто показывают raw backend text и не подсказывают следующий шаг;
- destructive confirmations слишком короткие и не всегда называют сущность;
- терминология местами расходится: "тип документа" vs "вид документа", "дело" vs "номенклатура", "содержание" vs "краткое содержание";
- сокращения `ПОС`, `Рег. №`, `Проср.` не раскрываются;
- empty states часто только говорят "нет данных", но не объясняют, что делать дальше;
- есть смешение `е/ё` and forms like `обновлен/обновлён`, `завершен/завершён`, `удален/удалён`.

## Контрольные Пункты H

| Код | Статус | Severity | Экран/компонент | Текущий текст | Проблема | Предложение |
| --- | --- | --- | --- | --- | --- | --- |
| H.01.221 | ok | none | pages/layout | Page titles mostly Title level 4 / modal titles. | Стиль заголовков в целом единый. | Сохранить pattern. |
| H.01.222 | issue | minor | buttons | `Откатить последнюю`, `Исполнено`, `Отметить`, `Проср.` | Не все кнопки начинаются с ясного действия. | `Откатить последнюю миграцию`, `Отметить исполненным`, `Отметить ознакомление`, `Просроченные`. |
| H.01.223 | ok | none | document pages | `Зарегистрировать`, primary buttons. | Primary action generally visible. | Нет. |
| H.01.224 | issue | minor | migration/storage actions | destructive buttons visually danger, but copy can be stronger. | Secondary/destructive actions need clearer consequence text. | See `MICROCOPY_REVIEW.md`. |
| H.02.225 | issue | major | forms/cards | `Вид документа`, backend `тип документа`, UI `Вид`, glossary `Тип документа`. | Domain terms conflict. | Separate: "Вид документа" = incoming/outgoing/appeal/order; "Тип документа" = Письмо/Договор/Акт. |
| H.02.226 | issue | major | document forms/view | `Дело`, `Номенклатура`, `Все дела`. | Same entity named differently. | Use `Дело (номенклатура)` in forms first, then `Дело`; use `Номенклатура` only in admin settings if needed. |
| H.02.227 | issue | minor | forms | `Исполнитель` can mean поручение executor, sender executor, resolution executor. | Different entities share same label. | Use `Ответственный исполнитель`, `Исполнитель письма`, `Исполнители резолюции`. |
| H.02.228 | issue | minor | icon buttons | Some icon buttons have tooltips, some not. | Meaning not always discoverable. | Add tooltips for edit/delete/view actions consistently. |
| H.02.229 | ok | none | forms | Most inputs have labels. | Placeholder not replacing labels except login fields where icons/title context exists. | Acceptable. |
| H.02.230 | issue | minor | filters/forms | `Поиск...`, `ФИО`, `Проср.` | Placeholders/examples often too vague. | `Поиск по номеру, содержанию или исполнителю`; `ФИО контролирующего`; `Просроченные`. |
| H.02.231 | issue | minor | forms | Help texts sparse. | Неочевидные fields like `ПОС`, manual numbering, file type list need short help. | Add concise help/tooltip. |
| H.03.232-H.03.234 | issue | major | errors | `err?.message || String(err)`, `Ошибка загрузки...` | Errors may be technical and not actionable. | Structured safe messages by code; include next step. |
| H.03.235 | issue | minor | success messages | `Удалено`, `Статус обновлен`. | Too generic. | `Поручение удалено`, `Статус поручения обновлён`. |
| H.03.236-H.03.237 | issue | major | destructive actions | `Удалить?`, `Удалить файл?`, rollback confirm. | Confirmation often lacks entity/consequence. | Name entity and consequence. |
| H.03.238 | issue | minor | Empty states | `Нет прикрепленных файлов`, charts with empty AntD Empty. | No next step. | `Файлы не прикреплены. Нажмите "Загрузить файл", чтобы добавить вложение.` |
| H.03.239-H.03.240 | issue | major | startup/errors/audit | raw backend messages, IDs in admin audit details. | Technical jargon/IDs can leak to user-facing surfaces. | Keep IDs in technical audit only; UI messages safe. |
| H.04.241-H.04.244 | issue | minor | many | `обновлен`, `завершен`, `dirty`, `N/A`. | Inconsistent Russian style and English leakage. | Use consistent Russian copy, avoid `dirty`/`N/A`. |
| H.04.245-H.04.247 | issue | minor | forms/status | `ПОС`, `Рег. №`, dates. | Abbreviations not expanded; dates mostly ok. | Expand on first use; keep `DD.MM.YYYY`. |
| H.05.248-H.05.256 | issue | minor | controls/forms | Switches mostly ok; some defaults and validation copy inconsistent. | Labels/help should match validation. | See form/microcopy reviews. |

## Приоритет

Major before production or as part of D/E remediation:

- structured safe error copy;
- destructive confirmations;
- terminology split `вид документа` / `тип документа`;
- startup/migration/system messages without technical jargon.

Minor can be batched:

- `е/ё` consistency;
- placeholders;
- empty state next steps;
- tooltip completeness.

Связанные issues: `ISSUE-044`-`ISSUE-049`.
