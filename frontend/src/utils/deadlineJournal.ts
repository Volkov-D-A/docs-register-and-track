type Snapshot = Record<string, unknown>;
const object = (value: unknown): Snapshot | undefined => value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Snapshot : undefined;
const date = (value: unknown) => typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value) ? value.split('-').reverse().join('.') : String(value ?? '');
const actions: Record<string, string> = {
  DEADLINE_CONTROL_CREATE: 'Контроль установлен', DEADLINE_CONTROL_UPDATE: 'Контроль изменён',
  DEADLINE_CONTROL_COMPLETE: 'Контроль исполнен', DEADLINE_CONTROL_CANCEL: 'Контроль отменён',
  DEADLINE_SERIES_CREATE: 'Регулярный контроль создан', DEADLINE_SERIES_UPDATE: 'Шаблон регулярного контроля изменён',
  DEADLINE_SERIES_STOP: 'Регулярный контроль остановлен', DEADLINE_SERIES_AUTO_STOP: 'Регулярный контроль остановлен автоматически',
  DEADLINE_SERIES_RESUME: 'Регулярный контроль возобновлён',
};

/** Render stored snapshots without changing the journal's original audit data. */
export function formatDeadlineJournal(action: string, details: string): string {
  if (!actions[action]) return details;
  let payload: Snapshot | undefined;
  try { payload = object(JSON.parse(details)); } catch { return details; }
  const after = object(payload?.after);
  if (!after) return details;
  const before = object(payload?.before);
  const parts = [actions[action]];
  if (typeof after.subject === 'string' && after.subject.trim()) parts[0] += `: ${after.subject.trim()}`;
  if (action.endsWith('_CREATE')) {
    const due = after.dueDate || after.firstDueDate;
    if (due) parts.push(`Срок: ${date(due)}`);
  } else if (action === 'DEADLINE_CONTROL_UPDATE' && after.dueDate && before?.dueDate !== after.dueDate) {
    parts.push(`Новый срок: ${date(after.dueDate)}`);
  } else if (action === 'DEADLINE_CONTROL_COMPLETE' && after.executedOn) {
    parts.push(`Дата исполнения: ${date(after.executedOn)}`);
  }
  const reason = action === 'DEADLINE_CONTROL_CANCEL' ? after.cancellationReason || payload?.reason
    : action === 'DEADLINE_SERIES_STOP' || action === 'DEADLINE_SERIES_AUTO_STOP' ? after.stopReason || payload?.reason
    : action === 'DEADLINE_CONTROL_UPDATE' ? payload?.reason : undefined;
  if (typeof reason === 'string' && reason.trim()) parts.push(`Причина: ${reason.trim()}`);
  return parts.join('; ');
}
