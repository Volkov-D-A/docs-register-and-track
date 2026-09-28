import React, { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, Card, Drawer, List, Pagination, Radio, Space, Spin, Statistic, Table, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, FileAddOutlined, FileDoneOutlined, FileTextOutlined, InboxOutlined, MessageOutlined, PlayCircleOutlined, ReloadOutlined, RightOutlined, SendOutlined, WarningOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { dto } from '../../wailsjs/go/models';
import { GetOverview, ListAcknowledgments } from '../../wailsjs/go/services/WorkspaceService';
import { GetCurrentUserEvents } from '../../wailsjs/go/services/UserEventService';
import { models } from '../../wailsjs/go/models';
import DocumentViewModal from '../components/DocumentViewModal';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import type { DocumentKindMeta } from '../constants/documentKinds';
import { useAuthStore } from '../store/useAuthStore';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { onAssignmentsChanged } from '../events/assignmentEvents';
import { onServerEvent } from '../events/serverEvents';
import { formatAppError, normalizeAppError } from '../utils/appError';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { workspaceDate, workspaceGreeting } from '../utils/workspaceHeading';

const { Title, Text } = Typography;
type WorkMode = 'execution' | 'control';
type AssignmentMetric = 'new' | 'in_progress' | 'overdue' | 'due_soon' | 'acceptance';

type DashboardPageProps = {
    onOpenAssignments: (mode: WorkMode, metric?: AssignmentMetric) => void;
    onOpenRegister: (kindCode: string, page: string) => void;
};

const metricIcons: Record<AssignmentMetric, React.ReactNode> = {
    new: <FileAddOutlined />,
    in_progress: <PlayCircleOutlined />,
    overdue: <WarningOutlined />,
    due_soon: <ClockCircleOutlined />,
    acceptance: <CheckCircleOutlined />,
};

const registrationIcons: Record<string, React.ReactNode> = {
    incoming: <InboxOutlined />,
    outgoing: <SendOutlined />,
    appeals: <MessageOutlined />,
    orders: <FileDoneOutlined />,
};

const assignmentStatuses: Record<string, { label: string; color: string }> = {
    new: { label: 'Новое', color: 'blue' },
    in_progress: { label: 'В работе', color: 'orange' },
    returned: { label: 'Возврат', color: 'volcano' },
    completed: { label: 'Ожидает приёмки', color: 'green' },
};

const modeOptions = [
    { label: 'Исполнение', value: 'execution' },
    { label: 'Контроль', value: 'control' },
];

const DashboardPage: React.FC<DashboardPageProps> = ({ onOpenAssignments, onOpenRegister }) => {
    const { message } = App.useApp();
    const user = useAuthStore((state) => state.user);
    const userId = user?.id;
    const { ready, registrationKinds } = useCurrentAccessSummary();
    const [assignmentMode, setAssignmentMode] = useState<WorkMode | ''>('');
    const [acknowledgmentMode, setAcknowledgmentMode] = useState<WorkMode | ''>('');
    const [overview, setOverview] = useState<dto.WorkspaceOverview | null>(null);
    const [loading, setLoading] = useState(false);
    const [currentTime, setCurrentTime] = useState(() => new Date());
    const [overviewError, setOverviewError] = useState('');
    const [document, setDocument] = useState<{ id: string; kind: string } | null>(null);
    const [ackListOpen, setAckListOpen] = useState(false);
    const [ackPage, setAckPage] = useState(1);
    const [ackItems, setAckItems] = useState<dto.WorkspaceAcknowledgment[]>([]);
    const [ackTotal, setAckTotal] = useState(0);
    const [ackLoading, setAckLoading] = useState(false);
    const [events, setEvents] = useState<dto.UserEvent[]>([]);
    const overviewRequest = useRef(new CoalescedRequest<dto.WorkspaceOverview>());
    const acknowledgmentsRequest = useRef(new CoalescedRequest<dto.PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_WorkspaceAcknowledgment_>());
    const eventsRequest = useRef(new CoalescedRequest<dto.PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_UserEvent_>());

    useEffect(() => {
        let timer: number;
        const scheduleUpdate = () => {
            const now = new Date();
            const untilNextMinute = 60_000 - now.getSeconds() * 1000 - now.getMilliseconds();
            timer = window.setTimeout(() => {
                setCurrentTime(new Date());
                scheduleUpdate();
            }, untilNextMinute);
        };
        scheduleUpdate();
        return () => window.clearTimeout(timer);
    }, []);

    useEffect(() => {
        setAssignmentMode('');
        setAcknowledgmentMode('');
        setOverview(null);
        setOverviewError('');
        setAckListOpen(false);
        setEvents([]);
        overviewRequest.current.invalidate();
        acknowledgmentsRequest.current.invalidate();
        eventsRequest.current.invalidate();
    }, [userId]);

    const loadOverview = useCallback(() => {
        if (!ready || !userId) return Promise.resolve();
        setLoading(true);
        return overviewRequest.current.refresh(() => GetOverview(assignmentMode, acknowledgmentMode), {
            onSuccess: (result) => { setOverview(result); setOverviewError(''); },
            onError: (error) => {
                const text = formatAppError(error, 'Не удалось загрузить рабочий стол');
                setOverview(null);
                setOverviewError(text);
                if (normalizeAppError(error).code === 'FORBIDDEN') {
                    setAssignmentMode('');
                    setAcknowledgmentMode('');
                }
                message.error(text);
            },
            onSettled: () => setLoading(false),
        });
    }, [ready, userId, assignmentMode, acknowledgmentMode, message]);

    const loadEvents = useCallback(() => {
        if (!ready || !userId) return Promise.resolve();
        return eventsRequest.current.refresh(() => GetCurrentUserEvents(models.UserEventFilter.createFrom({ page: 1, pageSize: 5 })), {
            onSuccess: (result) => setEvents(result?.items || []),
            onError: (error) => console.error('Workspace events:', error),
        });
    }, [ready, userId]);

    useEffect(() => {
        void loadOverview();
        void loadEvents();
    }, [loadOverview, loadEvents]);

    useEffect(() => onAssignmentsChanged(() => { void loadOverview(); }), [loadOverview]);
    useEffect(() => onServerEvent((event) => {
        if (event.topic === 'user-events' || event.topic === 'resync') {
            void loadOverview();
            void loadEvents();
        }
    }), [loadOverview, loadEvents]);

    useEffect(() => {
        if (!ackListOpen || !overview?.acknowledgmentMode) return;
        setAckLoading(true);
        void acknowledgmentsRequest.current.refresh(
            () => ListAcknowledgments(overview.acknowledgmentMode, ackPage, 10), {
                onSuccess: (result) => {
                    setAckItems(result?.items || []);
                    setAckTotal(result?.totalCount || 0);
                },
                onError: (error) => message.error(formatAppError(error, 'Не удалось загрузить ознакомления')),
                onSettled: () => setAckLoading(false),
            },
        );
    }, [ackListOpen, ackPage, overview?.acknowledgmentMode, overview?.acknowledgmentCount, message]);

    useEffect(() => () => {
        overviewRequest.current.invalidate();
        acknowledgmentsRequest.current.invalidate();
        eventsRequest.current.invalidate();
    }, []);

    const activeAssignmentMode = overview?.assignmentMode as WorkMode | undefined;
    const activeAcknowledgmentMode = overview?.acknowledgmentMode as WorkMode | undefined;
    const openAcknowledgments = () => {
        setAckPage(1);
        setAckListOpen(true);
    };
    const openDocument = (id: string, kind: string) => setDocument({ id, kind });
    const acknowledgmentTitle = activeAcknowledgmentMode === 'control' ? 'Контроль ознакомлений' : 'Мои ознакомления';
    const counts = overview?.assignmentCounts;
    const assignmentColumns: TableProps<dto.WorkspaceAssignment>['columns'] = [
        {
            title: 'Срок', dataIndex: 'deadline', key: 'deadline', width: 120,
            render: (value: string | undefined) => value ? dayjs(value).format('DD.MM.YYYY') : '—',
        },
        {
            title: 'Документ', key: 'document', width: 190,
            render: (_: unknown, item) => (
                <div>
                    <div>{item.documentNumber || '—'}</div>
                    <Text type="secondary">{item.documentDate ? dayjs(item.documentDate).format('DD.MM.YYYY') : '—'}</Text>
                </div>
            ),
        },
        { title: 'Содержание поручения', dataIndex: 'content', key: 'content' },
        {
            title: 'Статус', dataIndex: 'status', key: 'status', width: 150,
            render: (value: string) => {
                const status = assignmentStatuses[value];
                return <Tag color={status?.color || 'default'}>{status?.label || value}</Tag>;
            },
        },
        {
            title: 'Действие', key: 'action', width: 100,
            render: (_: unknown, item) => (
                <Button type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>
            ),
        },
    ];

    if (!ready || (loading && !overview)) {
        return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
    }

    return (
        <div style={{ padding: 8 }}>
            <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 20 }}>
                <div>
                    <Title level={4} style={{ margin: 0 }}>{workspaceGreeting(currentTime, user)}</Title>
                    <Text type="secondary">{workspaceDate(currentTime)}</Text>
                </div>
                <Button icon={<ReloadOutlined />} loading={loading} onClick={() => { void loadOverview(); void loadEvents(); }}>Обновить</Button>
            </Space>
            {overviewError && <Card style={{ marginBottom: 16 }}><Text type="danger">{overviewError}</Text></Card>}
            <div className={`workspace-overview-row${!(activeAssignmentMode || activeAcknowledgmentMode) ? ' workspace-overview-row--right-only' : ''}`}>
                {(activeAssignmentMode || activeAcknowledgmentMode) && (
                    <div className="workspace-work-column">
                        {activeAssignmentMode && (
                            <>
                                <Card className="workspace-metrics-panel">
                                    <div className="workspace-metrics-grid">
                                        {([
                                            ['Новые', counts?.new || 0, 'new'],
                                            ['В работе', counts?.inProgress || 0, 'in_progress'],
                                            ['Просрочены', counts?.overdue || 0, 'overdue'],
                                            ['Срок в ближайшие 3 дня', counts?.dueSoon || 0, 'due_soon'],
                                            ...(activeAssignmentMode === 'control' ? [['Ожидают приёмки', counts?.awaitingAcceptance || 0, 'acceptance']] : []),
                                        ] as [string, number, AssignmentMetric][]).map(([title, value, metric]) => (
                                            <Card key={metric} size="small" hoverable role="button" tabIndex={0}
                                                className={`workspace-metric-card workspace-metric-card--${metric}`}
                                                onClick={() => onOpenAssignments(activeAssignmentMode, metric)}
                                                onKeyDown={(event) => { if (event.key === 'Enter') onOpenAssignments(activeAssignmentMode, metric); }}>
                                                <div className="workspace-metric-content">
                                                    <Statistic title={title} value={value} />
                                                    <span className="workspace-metric-icon" aria-hidden="true">{metricIcons[metric]}</span>
                                                </div>
                                            </Card>
                                        ))}
                                    </div>
                                </Card>
                                <Card title={<Space>
                                    <span>Поручения</span>
                                    {overview && overview.assignmentModes.length > 1 ? (
                                        <span className="workspace-assignment-modes">
                                            {'( '}
                                            <Button type="link" size="small" aria-pressed={activeAssignmentMode === 'execution'}
                                                className={`workspace-assignment-mode-link${activeAssignmentMode === 'execution' ? ' workspace-assignment-mode-link--active' : ''}`}
                                                onClick={() => setAssignmentMode('execution')}>исполнение</Button>
                                            {' \\ '}
                                            <Button type="link" size="small" aria-pressed={activeAssignmentMode === 'control'}
                                                className={`workspace-assignment-mode-link${activeAssignmentMode === 'control' ? ' workspace-assignment-mode-link--active' : ''}`}
                                                onClick={() => setAssignmentMode('control')}>контроль</Button>
                                            {' )'}
                                        </span>
                                    ) : <Text type="secondary">( {activeAssignmentMode === 'control' ? 'контроль' : 'исполнение'} )</Text>}
                                </Space>} extra={
                                    <Button type="link" onClick={() => onOpenAssignments(activeAssignmentMode)}>Все поручения</Button>
                                }>
                                    <Table<dto.WorkspaceAssignment> columns={assignmentColumns} dataSource={overview?.assignments || []}
                                        rowKey="id" size="small" pagination={false} scroll={{ x: 800 }}
                                        locale={{ emptyText: 'Актуальных поручений нет' }} />
                                </Card>
                            </>
                        )}
                        {activeAcknowledgmentMode && (
                            <Card title={acknowledgmentTitle} extra={overview && overview.acknowledgmentModes.length > 1 && (
                                <Radio.Group size="small" options={modeOptions} value={activeAcknowledgmentMode}
                                    onChange={(event) => { setAcknowledgmentMode(event.target.value as WorkMode); setAckPage(1); }} />
                            )}>
                                <Button type="link" onClick={openAcknowledgments}>Ожидают внимания: {overview?.acknowledgmentCount || 0}</Button>
                                <List dataSource={overview?.acknowledgments || []} locale={{ emptyText: 'Актуальных ознакомлений нет' }}
                                    renderItem={(item) => <List.Item actions={[<Button key="open" type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>]}>
                                        <List.Item.Meta title={item.content || 'Ознакомление'} description={item.documentNumber || 'Документ без номера'} />
                                    </List.Item>} />
                                <Button type="link" onClick={openAcknowledgments}>Все ознакомления</Button>
                            </Card>
                        )}
                    </div>
                )}
                <div className="workspace-side-column">
                    {registrationKinds.length > 0 && (
                        <Card title="Быстрая регистрация" className="workspace-registration-panel">
                            <Table<DocumentKindMeta> className="workspace-registration-table" dataSource={registrationKinds}
                                rowKey="code" size="small" showHeader={false} pagination={false}
                                columns={[{
                                    key: 'kind',
                                    render: (_value, kind) => (
                                        <Button type="text" block className="workspace-registration-action" onClick={() => {
                                            useRegisterDocumentStore.getState().requestOpen(kind.code);
                                            onOpenRegister(kind.code, kind.pageKey);
                                        }}>
                                            <span className={`workspace-registration-icon workspace-registration-icon--${kind.pageKey}`} aria-hidden="true">
                                                {registrationIcons[kind.pageKey] || <FileTextOutlined />}
                                            </span>
                                            <span>{kind.label}</span>
                                            <RightOutlined className="workspace-registration-chevron" aria-hidden="true" />
                                        </Button>
                                    ),
                                }]} />
                        </Card>
                    )}
                    <Card title="Новое для меня">
                        <List dataSource={events} locale={{ emptyText: 'Новых событий нет' }}
                            renderItem={(item) => <List.Item actions={[<Button key="open" type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>]}>
                                <List.Item.Meta title={item.title} description={item.message} />
                            </List.Item>} />
                    </Card>
                </div>
            </div>
            <Drawer title={acknowledgmentTitle} open={ackListOpen} width={640} onClose={() => setAckListOpen(false)}>
                <List loading={ackLoading} dataSource={ackItems} locale={{ emptyText: 'Ознакомлений нет' }}
                    renderItem={(item) => <List.Item actions={[<Button key="open" type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>]}>
                        <List.Item.Meta title={item.content || 'Ознакомление'} description={item.documentNumber || 'Документ без номера'} />
                    </List.Item>} />
                <Pagination current={ackPage} pageSize={10} total={ackTotal} onChange={setAckPage} style={{ marginTop: 16 }} />
            </Drawer>
            <DocumentViewModal open={!!document} onCancel={() => setDocument(null)} documentId={document?.id || ''}
                documentKind={document?.kind || ''} onAssignmentsChanged={loadOverview} onAcknowledgmentsChanged={loadOverview} />
        </div>
    );
};

export default DashboardPage;
