const listeners = new Set<() => void>();
export const deadlinesChanged = () => { for (const listener of listeners) listener(); };
export const onDeadlinesChanged = (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener); }; };
export const organizationToday = (date = new Date()) => {
  const parts = new Intl.DateTimeFormat('en', { timeZone: 'Asia/Yekaterinburg', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(date);
  const value = (type: string) => parts.find((part) => part.type === type)?.value || '';
  return `${value('year')}-${value('month')}-${value('day')}`;
};
export const addCalendarDays = (date: string, days: number) => { const value = new Date(`${date}T00:00:00Z`); value.setUTCDate(value.getUTCDate() + days); return value.toISOString().slice(0, 10); };
export type DeadlineView = { mineOnly?: boolean; manageOnly?: boolean; status?: string; overdueOnly?: boolean; dueFrom?: string; dueTo?: string; seriesStoppedOnly?: boolean };
