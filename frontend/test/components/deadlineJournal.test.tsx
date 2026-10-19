import { expect, test } from 'vitest';
import { formatDeadlineJournal } from '../../src/utils/deadlineJournal';
const after = { id: 'control-id', subject: 'Ответ на письмо', executorId: 'executor-id', coExecutorIds: [], dueDate: '2026-10-20', status: 'active', version: 1 };
test('formats existing creation snapshots without technical fields', () => {
  expect(formatDeadlineJournal('DEADLINE_CONTROL_CREATE', JSON.stringify({ before: null, after }))).toBe('Контроль установлен: Ответ на письмо; Срок: 20.10.2026');
});
test('shows changed fields and rescheduling reason', () => {
  expect(formatDeadlineJournal('DEADLINE_CONTROL_UPDATE', JSON.stringify({ before: after, after: { ...after, dueDate: '2026-10-23', version: 2 }, reason: 'Ожидаем сведения' }))).toBe('Контроль изменён: Ответ на письмо; Новый срок: 23.10.2026; Причина: Ожидаем сведения');
});
test('completion omits comment and repeated status', () => {
  expect(formatDeadlineJournal('DEADLINE_CONTROL_COMPLETE', JSON.stringify({ before: after, after: { ...after, status: 'completed', executedOn: '2026-10-19', comment: 'Ответ направлен' } }))).toBe('Контроль исполнен: Ответ на письмо; Дата исполнения: 19.10.2026');
});
test('series omits recurrence details and keeps a single stop reason', () => {
  expect(formatDeadlineJournal('DEADLINE_SERIES_CREATE', JSON.stringify({ after: { recurrence: { intervalUnit: 'month', intervalValue: 1, dayRule: 'last_day' } } }))).toBe('Регулярный контроль создан');
  expect(formatDeadlineJournal('DEADLINE_SERIES_STOP', JSON.stringify({ before: { active: true }, after: { active: false, stopReason: 'Больше не требуется' }, reason: 'Больше не требуется' }))).toBe('Регулярный контроль остановлен; Причина: Больше не требуется');
});
test('preserves legacy text and unrelated events', () => {
  expect(formatDeadlineJournal('DEADLINE_CONTROL_CREATE', 'Старое описание')).toBe('Старое описание');
  expect(formatDeadlineJournal('DEADLINE_CONTROL_CREATE', 'null')).toBe('null');
  expect(formatDeadlineJournal('UPDATE', '{"field":"value"}')).toBe('{"field":"value"}');
});
