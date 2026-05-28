# DTO Review

Дата аудита: 2026-05-27

## Findings

DTO слой в `internal/dto` в основном отделен от database models: UUID наружу отдаются как строки, внутренние DB-only fields часто скрыты через mapper. Это хорошая база для стабильного frontend contract.

Проблемные зоны:

- command DTO для регистрации документов находятся в `internal/services`, а не в `internal/dto`, и принимаются через `any`;
- create DTO не содержат `idempotencyKey`;
- даты в response идут как `time.Time`, а command layer принимает строки `YYYY-MM-DD`; для optional datetime приказа допускается RFC3339 или date-only;
- request nullable policy основана на пустых строках, response nullable policy основана на pointers/`omitempty`;
- `omitempty` на bool/int fields в списках может скрывать осмысленные false/0, если frontend различает "нет данных" и "false/0".

## Issues

### ISSUE-013: missing idempotency DTO

Severity: major
Пункты: C.03.086, C.02.084
Место: document register requests

`IncomingLetterRegisterRequest`, `OutgoingLetterRegisterRequest`, `CitizenAppealRegisterRequest`, `AdministrativeOrderRegisterRequest` не содержат `IdempotencyKey`.

Рекомендация: добавить обязательный `idempotencyKey` в create DTO и проверить frontend repeated-submit behavior.

### ISSUE-014: unstable command DTO boundary

Severity: major
Пункты: C.02.085, C.03.086
Место: `DocumentRegistrationService.Register/Update`

`any` + JSON roundtrip делает public contract менее явным и не отклоняет лишние поля.

Рекомендация: typed methods или strict normalization с проверкой unknown fields.

## Point Status

| Пункт | Статус | Severity | Вывод |
| --- | --- | --- | --- |
| C.03.086 | issue | major | DTO в целом стабильны, но document command API нестабилен и без idempotency. |
| C.03.087 | ok | none | DTO не раскрывают ключевую внутреннюю DB структуру. |
| C.03.088 | issue | minor | Date/datetime contract надо явно закрепить. |
| C.03.089 | issue | minor | Empty string vs null нужно унифицировать. |
| C.03.090 | issue | major | Frontend сейчас получает plain error strings. |
