import { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, Card, Select, Space, Statistic, Typography } from 'antd';
import { models } from '../../wailsjs/go/models';
import { List } from '../../wailsjs/go/services/DeadlineControlService';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { formatAppError } from '../utils/appError';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { onServerEvent } from '../events/serverEvents';
import { addCalendarDays, onDeadlinesChanged, organizationToday, type DeadlineView } from '../features/deadlines/deadlineEvents';
import RecentDocumentsPanel from '../components/RecentDocumentsPanel';
import DocumentSearchPanel from '../components/DocumentSearchPanel';
import WorkspaceUserEventsPanel from '../components/WorkspaceUserEventsPanel';
import LazyDocumentModal from '../components/DocumentViewModal';

type Props = { onOpenControls: (view: DeadlineView) => void; onOpenRegister: (kind: string, page: string) => void };
export default function DeadlineDashboardPage({ onOpenControls, onOpenRegister }: Props) {
  const { message } = App.useApp();
  const { hasAnyAction, registrationKinds } = useCurrentAccessSummary();
  const [mine, setMine] = useState(true);
  const [counts, setCounts] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);
  const [document, setDocument] = useState<{ id: string; kind: string }>();
  const request = useRef(new CoalescedRequest<number[]>());
  const today = organizationToday();
  const base: DeadlineView = { mineOnly: mine, manageOnly: !mine, status: 'active' };
  const metrics: { label: string; view: DeadlineView }[] = [
    { label: 'Активные', view: base },
    { label: 'Просроченные', view: { ...base, overdueOnly: true } },
    { label: 'Сегодня', view: { ...base, dueFrom: today, dueTo: today } },
    { label: 'Ближайшие три дня', view: { ...base, dueFrom: today, dueTo: addCalendarDays(today, 3) } },
    { label: 'Исполненные', view: { ...base, status: 'completed' } },
    { label: 'Остановленные серии', view: { ...base, status: '', seriesStoppedOnly: true } },
  ];
  const load = useCallback(async () => {
    setLoading(true);
    const today = organizationToday();
    const base = { mineOnly: mine, manageOnly: !mine, status: 'active' };
    const filters = [base, { ...base, overdueOnly: true }, { ...base, dueFrom: today, dueTo: today }, { ...base, dueFrom: today, dueTo: addCalendarDays(today, 3) }, { ...base, status: 'completed' }, { ...base, status: '', seriesStoppedOnly: true }];
    await request.current.refresh(() => Promise.all(filters.map(async (filter) => (await List(models.DeadlineControlFilter.createFrom({ ...filter, page: 1, pageSize: 1 }))).total)), {
      onSuccess: setCounts, onError: (error) => { setCounts([]); message.error(formatAppError(error)); }, onSettled: () => setLoading(false),
    });
  }, [mine, message]);
  useEffect(() => { const current = request.current; void load(); return () => current.invalidate(); }, [load]);
  useEffect(() => {
    const stop = onDeadlinesChanged(() => { void load(); });
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stopServer = onServerEvent((event) => { if (['resync', 'access-changed', 'documents'].includes(event.topic) || event.resource === 'controls') { setCounts([]); clearTimeout(timer); timer = setTimeout(() => { void load(); }, 100); } });
    const clock = setInterval(() => { void load(); }, 60_000);
    return () => { stop(); stopServer(); clearTimeout(timer); clearInterval(clock); };
  }, [load]);
  return <>
    <Typography.Title level={2}>Рабочий стол</Typography.Title>
    <Space style={{ marginBottom: 16 }}><Select value={mine ? 'mine' : 'queue'} onChange={(value) => setMine(value === 'mine')} options={[{ value: 'mine', label: 'Мои сроки' }, ...(hasAnyAction('control_deadlines') ? [{ value: 'queue', label: 'Контроль сроков' }] : [])]} style={{ width: 200 }} /><Button loading={loading} onClick={() => void load()}>Обновить</Button></Space>
    <Space wrap style={{ marginBottom: 24 }}>
      {metrics.map((metric, index) => <Card key={metric.label}><Statistic title={metric.label} value={counts[index] ?? '—'} /><Button type="link" onClick={() => onOpenControls(metric.view)}>Открыть список</Button></Card>)}
    </Space>
    <Space wrap style={{ marginBottom: 24 }}>{registrationKinds.map((kind) => <Button key={kind.code} onClick={() => { useRegisterDocumentStore.getState().requestOpen(kind.code); onOpenRegister(kind.code, kind.pageKey); }}>Зарегистрировать: {kind.label}</Button>)}</Space>
    <DocumentSearchPanel onOpenDocument={(id, kind) => setDocument({ id, kind })} />
    <RecentDocumentsPanel refreshVersion={0} onOpenDocument={(id, kind) => setDocument({ id, kind })} />
    <WorkspaceUserEventsPanel currentTime={new Date()} refreshVersion={0} onOpenDocument={(id, kind) => setDocument({ id, kind })} />
    {document && <LazyDocumentModal documentId={document.id} documentKind={document.kind} open onCancel={() => setDocument(undefined)} />}
  </>;
}
