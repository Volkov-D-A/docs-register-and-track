import { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, Card, Space, Statistic, Table, Typography } from 'antd';
import { models } from '../../wailsjs/go/models';
import { List, ListExecutors } from '../../wailsjs/go/services/DeadlineControlService';
import { useOrganizationDay } from '../hooks/useOrganizationDay';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { formatAppError } from '../utils/appError';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { onServerEvent } from '../events/serverEvents';
import { addCalendarDays, onDeadlinesChanged, type DeadlineView } from '../features/deadlines/deadlineEvents';
import dayjs from 'dayjs';
import { ClockCircleOutlined, CheckCircleOutlined, WarningOutlined, PauseCircleOutlined, FileTextOutlined, RightOutlined } from '@ant-design/icons';
import { useAuthStore } from '../store/useAuthStore';
import { workspaceDate, workspaceGreeting } from '../utils/workspaceHeading';
import RecentDocumentsPanel from '../components/RecentDocumentsPanel';
import DocumentSearchPanel from '../components/DocumentSearchPanel';
import WorkspaceUserEventsPanel from '../components/WorkspaceUserEventsPanel';
import LazyDocumentModal from '../components/DocumentViewModal';

type Props = { onOpenControls: (view: DeadlineView) => void; onOpenRegister: (kind: string, page: string) => void };
export default function DeadlineDashboardPage({ onOpenControls, onOpenRegister }: Props) {
  const { message } = App.useApp();
  const { hasAnyAction, registrationKinds } = useCurrentAccessSummary();
  const user = useAuthStore((state) => state.user);
  const currentTime = new Date();
  const today = useOrganizationDay();
  const [refreshVersion, setRefreshVersion] = useState(0);
  const [executors, setExecutors] = useState<Record<string, string>>({});
  const [items, setItems] = useState<models.DocumentDeadlineControl[]>([]);
  const [mine, setMine] = useState(true);
  const [counts, setCounts] = useState<number[]>([]);
  const [loadError, setLoadError] = useState('');
  const [document, setDocument] = useState<{ id: string; kind: string }>();
  const request = useRef(new CoalescedRequest<{ counts: number[]; items: models.DocumentDeadlineControl[]; executors: Record<string, string> }>());
  const base: DeadlineView = { mineOnly: mine, manageOnly: !mine, status: 'active' };
  const metrics: { label: string; view: DeadlineView }[] = [
    { label: 'Активные', view: base },
    { label: 'Просроченные', view: { ...base, overdueOnly: true } },
    { label: 'Сегодня', view: { ...base, dueFrom: today, dueTo: today } },
    { label: 'Ближайшие три дня', view: { ...base, dueFrom: today, dueTo: addCalendarDays(today, 3) } },
    { label: 'Исполненные', view: { ...base, status: 'completed' } },
    ...(!mine ? [{ label: 'Остановленные серии', view: { ...base, status: '', seriesStoppedOnly: true } }] : []),
  ];
  const load = useCallback(async () => {
    const base = { mineOnly: mine, manageOnly: !mine, status: 'active' };
    const filters = [base, { ...base, overdueOnly: true }, { ...base, dueFrom: today, dueTo: today }, { ...base, dueFrom: today, dueTo: addCalendarDays(today, 3) }, { ...base, status: 'completed' }, ...(!mine ? [{ ...base, status: '', seriesStoppedOnly: true }] : [])];
    await request.current.refresh(async () => {
      const [pages, people] = await Promise.all([
        Promise.all(filters.map((filter, index) => List(models.DeadlineControlFilter.createFrom({ ...filter, page: 1, pageSize: index === 0 ? 5 : 1 })))),
        mine ? Promise.resolve([]) : ListExecutors(true),
      ]);
      return { counts: pages.map((page) => page.total), items: pages[0].items, executors: Object.fromEntries(people.map((person) => [person.id, person.name])) };
    }, {
      onSuccess: (result) => { setCounts(result.counts); setItems(result.items); setExecutors(result.executors); setLoadError(''); }, onError: (error) => { setCounts([]); setItems([]); setExecutors({}); setLoadError(formatAppError(error)); message.error(formatAppError(error)); },
    });
  }, [mine, message, today]);
  useEffect(() => { const current = request.current; void load(); return () => current.invalidate(); }, [load, user?.id]);
  useEffect(() => {
    const stop = onDeadlinesChanged(() => { void load(); });
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stopServer = onServerEvent((event) => { if (['resync', 'access-changed', 'documents', 'document-changed'].includes(event.topic)) { setCounts([]); setItems([]); setExecutors({}); clearTimeout(timer); timer = setTimeout(() => { void load(); }, 100); } });
    const resume = () => { if (window.document.visibilityState === 'hidden') return; setRefreshVersion((value) => value + 1); void load(); };
    window.addEventListener('focus', resume);
    window.document.addEventListener('visibilitychange', resume);
    return () => { stop(); stopServer(); clearTimeout(timer); window.removeEventListener('focus', resume); window.document.removeEventListener('visibilitychange', resume); };
  }, [load]);
  const icons = [<FileTextOutlined />, <WarningOutlined />, <ClockCircleOutlined />, <ClockCircleOutlined />, <CheckCircleOutlined />, <PauseCircleOutlined />];
  const tones = ['new', 'overdue', 'in_progress', 'due_soon', 'acceptance', 'returned'];
  const openDocument = (id: string, kind: string) => setDocument({ id, kind });
  return <div className="workspace-dashboard">
    <Space align="start" wrap className="workspace-dashboard-header" style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}>
      <div><div className="workspace-heading">
        <Typography.Title level={4} style={{ margin: 0 }}>{workspaceGreeting(currentTime, user)}</Typography.Title>
        {hasAnyAction('control_deadlines') && <span className="workspace-modes" role="group" aria-label="Режим рабочего стола">
          {'( '}<Button type="link" size="small" aria-pressed={mine} className={`workspace-mode-link${mine ? ' workspace-mode-link--active' : ''}`} onClick={() => setMine(true)}>мои сроки</Button>
          {' / '}<Button type="link" size="small" aria-pressed={!mine} className={`workspace-mode-link${!mine ? ' workspace-mode-link--active' : ''}`} onClick={() => setMine(false)}>контроль</Button>{' )'}
        </span>}
      </div><Typography.Text type="secondary">{workspaceDate(currentTime)}</Typography.Text></div>
      <DocumentSearchPanel onOpenDocument={openDocument} />
    </Space>
    <div className="workspace-overview-row">
      <div className="workspace-work-column">
        <Card className="workspace-metrics-panel"><div className="workspace-metrics-grid">
          {metrics.map((metric, index) => <Card key={metric.label} size="small" hoverable role="button" tabIndex={0} aria-label={metric.label}
            className={`workspace-metric-card workspace-metric-card--${tones[index]}`}
            onClick={() => onOpenControls(metric.view)} onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); onOpenControls(metric.view); } }}>
            <div className="workspace-metric-content"><Statistic title={metric.label} value={counts[index] ?? '—'} /><span className="workspace-metric-icon" aria-hidden>{icons[index]}</span></div>
          </Card>)}
        </div></Card>
        <Card title="Актуальные сроки" extra={<Button type="link" onClick={() => onOpenControls(base)}>Все сроки</Button>}>
          {loadError && <Space style={{ marginBottom: 8 }}><Typography.Text type="danger">{loadError}</Typography.Text><Button onClick={() => void load()}>Повторить</Button></Space>}
          <Table<models.DocumentDeadlineControl> size="small" tableLayout="fixed" rowKey="id" dataSource={items} pagination={false} scroll={{ x: mine ? 600 : 860 }}
            columns={[
              { title: 'Срок', dataIndex: 'dueDate', width: 110, render: (value: string) => <Typography.Text type={value < today ? 'danger' : undefined}>{dayjs(value).format('DD.MM.YYYY')}</Typography.Text> },
              { title: 'Документ', dataIndex: 'documentNumber', width: 150 },
              { title: 'Предмет контроля', dataIndex: 'subject' },
              ...(!mine ? [{ title: 'Исполнители', key: 'executors', width: 260, render: (_value: unknown, item: models.DocumentDeadlineControl) => <>
                <div>{executors[item.executorId] || 'Нет в справочнике'}</div>
                {!!item.coExecutorIds?.length && <Typography.Text type="secondary">Соисполнители: {item.coExecutorIds.map((id) => executors[id] || 'Нет в справочнике').join(', ')}</Typography.Text>}
              </> }] : []),
            ]}
            onRow={(item) => ({ tabIndex: 0, style: { cursor: 'pointer' }, onClick: () => openDocument(item.documentId, item.documentKind), onKeyDown: (event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); openDocument(item.documentId, item.documentKind); } } })}
            locale={{ emptyText: 'Активных сроков нет' }} />
        </Card>
      </div>
      <div className="workspace-side-column">
        {registrationKinds.length > 0 && <Card title="Быстрая регистрация" className="workspace-registration-panel">
          <Table dataSource={registrationKinds} rowKey="code" size="small" showHeader={false} pagination={false} className="workspace-registration-table" columns={[{ key: 'kind', render: (_value, kind) => <Button type="text" block className="workspace-registration-action" onClick={() => { useRegisterDocumentStore.getState().requestOpen(kind.code); onOpenRegister(kind.code, kind.pageKey); }}><span className={`workspace-registration-icon workspace-registration-icon--${kind.pageKey}`}><FileTextOutlined /></span><span>{kind.label}</span><RightOutlined className="workspace-registration-chevron" /></Button> }]} />
        </Card>}
        <WorkspaceUserEventsPanel currentTime={currentTime} refreshVersion={refreshVersion} onOpenDocument={openDocument} />
        <RecentDocumentsPanel refreshVersion={refreshVersion} onOpenDocument={openDocument} />
      </div>
    </div>
    {document && <LazyDocumentModal documentId={document.id} documentKind={document.kind} open onCancel={() => { setDocument(undefined); setRefreshVersion((value) => value + 1); }} />}
  </div>;
}
