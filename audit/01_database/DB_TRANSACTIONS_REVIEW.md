# DB Transactions Review

Дата аудита: 2026-05-27

## Summary

Многие multi-row operations уже используют transactions: создание/обновление документов с detail rows, поручения с соисполнителями, ознакомления с пользователями, departments с номенклатурой, users. Главная проблема не в отсутствии transactions вообще, а в том, что критичная выдача регистрационного номера находится вне transaction boundary создания документа.

## Major Findings

### ISSUE-006: numbering outside document create transaction

Severity: major
Пункты: B.05.066, B.05.067, B.05.068
Место: `NomenclatureRepository.GetNextNumber`, document create services/repositories

`GetNextNumber` выполняет:

```sql
UPDATE nomenclature
SET next_number = next_number + 1
WHERE id = $1
RETURNING next_number - 1
```

Document repositories затем отдельно создают `documents` и detail rows в своих transactions. Если ошибка возникает между выдачей номера и commit создания документа, номер потерян. Повторный submit без idempotency key может создать дубль и повторно израсходовать номер.

Локальное воспроизведение на test contour:

- подготовлена номенклатура `GAP` с `next_number=1`;
- выполнен отдельный autocommit-инкремент, эквивалентный текущему `GetNextNumber`, вернувший номер `1`;
- следующий `INSERT INTO documents` намеренно упал по FK `documents_created_by_fkey`;
- после ошибки `nomenclature.next_number=2`, а `COUNT(documents WHERE registration_number='GAP/1')=0`.

Вывод: gap возникает без конкурентности, только из-за того, что выдача номера committed отдельно от создания документа.

Рекомендация:

- перенести выдачу номера внутрь transaction регистрации документа;
- блокировать строку номенклатуры через `UPDATE ... RETURNING` в той же tx или `SELECT ... FOR UPDATE`;
- добавить `idempotency_key` и unique constraint;
- journal write регистрации включить в ту же logical transaction либо явно описать compensating behavior.

Минимальный целевой контракт:

- request регистрации всех 4 видов документов содержит `idempotency_key`;
- схема хранит ключ в `documents.idempotency_key UUID NOT NULL`;
- unique index/constraint: `(created_by, kind, idempotency_key)`;
- при повторе с тем же ключом backend возвращает уже созданный документ;
- автоматический номер выделяется только внутри той же DB transaction, где создаются `documents`, detail rows и записывается `idempotency_key`;
- при rollback transaction `next_number` не меняется.

Целевой порядок внутри transaction:

1. `SELECT id FROM documents WHERE created_by=$1 AND kind=$2 AND idempotency_key=$3`.
2. Если найдено — вернуть существующий документ без вызова numbering logic.
3. Если не найдено — выполнить `UPDATE nomenclature SET next_number = next_number + 1 ... RETURNING next_number - 1, index, separator, numbering_mode` в той же tx.
4. Вставить root `documents` с `idempotency_key`.
5. Вставить detail rows и дочерние строки.
6. Commit; после commit загрузить созданный документ для DTO.

Concurrent duplicate case: если два запроса с одним ключом одновременно дошли до insert, unique violation должен откатить всю вторую transaction, включая `next_number`; затем retry/read existing возвращает уже созданный документ без пропуска номера.

Decision: см. `DECISION-004` — `documents.idempotency_key` и выдача номера должны быть частью единой DB transaction регистрации документа.

Затронутые места:

- `IncomingLetterCommandHandler.Register`;
- `OutgoingLetterCommandHandler.Register`;
- `CitizenAppealCommandHandler.Register`;
- `AdministrativeOrderCommandHandler.Register`;
- `NomenclatureRepository.GetNextNumber`;
- `Create*DocRequest` models;
- `Incoming/Outgoing/Citizen/Administrative*Repository.Create`;
- frontend submit payloads в `IncomingPage`, `OutgoingPage`, `CitizenAppealsPage`, `OrdersPage`.

### Attachment metadata and MinIO

Severity: medium
Пункты: B.05.066

PostgreSQL metadata and MinIO object cannot be in one DB transaction. В рамках A подтверждена recovery policy: PostgreSQL и MinIO восстанавливаются из согласованного backup-набора. На C нужно проверить compensation paths upload/delete/bulk delete.

## Other Notes

- Assignment create/update correctly wraps parent row plus co-executors.
- Acknowledgment create/confirm wraps acknowledgment users and completion update.
- Document update replaces correspondents/resolutions inside tx.
- Some operations use plain `db.Exec` without context timeout; this is a backend/resource lifecycle topic for C/E, но в B фиксируется как DB runtime risk.

## Findings

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| B.05.066 | issue | major | Критичная registration transaction не включает numbering; локальный failure test подтвердил gap: `next_number=2`, `docs=0`. |
| B.05.067 | issue | major | Default isolation недостаточна для no-gaps, пока номер выдается вне tx/autocommit. |
| B.05.068 | issue | major | Нет idempotency key для конкурентных/повторных submits; request structs и frontend payloads ключ не передают. |
| B.05.069 | transferred | none | Lock wait проверить на C/F отдельным concurrent registration test после изменения transaction model. |
| B.05.070 | transferred | none | Constraint error mapping передан на C. |
| B.05.071 | transferred | none | DB disconnect/reconnect передан на C/E. |
| B.05.072 | issue | minor | `sql.DB` pool settings не заданы явно. |
| B.05.073 | issue | minor | Большинство repository calls без context timeout. |
