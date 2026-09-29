import React, { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, Card, Space, Spin, Statistic, Table, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, FileAddOutlined, FileDoneOutlined, FileTextOutlined, InboxOutlined, MessageOutlined, PlayCircleOutlined, RightOutlined, RollbackOutlined, SendOutlined, WarningOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { dto } from '../../wailsjs/go/models';
import { GetOverview } from '../../wailsjs/go/services/WorkspaceService';
import DocumentViewModal from '../components/DocumentViewModal';
import WorkspaceUserEventsPanel from '../components/WorkspaceUserEventsPanel';
import RecentDocumentsPanel from '../components/RecentDocumentsPanel';
import DocumentSearchPanel from '../components/DocumentSearchPanel';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import type { AssignmentMetric, AssignmentType } from '../components/assignmentNavigation';
import type { DocumentKindMeta } from '../constants/documentKinds';
import { useAuthStore } from '../store/useAuthStore';
import { useAssignmentModeStore } from '../store/useAssignmentModeStore';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { onAssignmentsChanged } from '../events/assignmentEvents';
import { onServerEvent } from '../events/serverEvents';
import { formatAppError, normalizeAppError } from '../utils/appError';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { workspaceDate, workspaceGreeting } from '../utils/workspaceHeading';

const { Title, Text } = Typography;
type WorkMode = 'execution' | 'control';

type DashboardPageProps = {
    onOpenAssignments: (mode: WorkMode, metric?: AssignmentMetric, type?: AssignmentType) => void;
    onOpenRegister: (kindCode: string, page: string) => void;
};

const metricIcons: Record<AssignmentMetric, React.ReactNode> = {
    new: <FileAddOutlined />,
    in_progress: <PlayCircleOutlined />,
    overdue: <WarningOutlined />,
    due_soon: <ClockCircleOutlined />,
    acceptance: <CheckCircleOutlined />,
    returned: <RollbackOutlined />,
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

const DashboardPage: React.FC<DashboardPageProps> = ({ onOpenAssignments, onOpenRegister }) => {
    const { message } = App.useApp();
    const user = useAuthStore((state) => state.user);
    const userId = user?.id;
    const { ready, registrationKinds } = useCurrentAccessSummary();
    const workMode = useAssignmentModeStore((state) => state.mode);
    const setWorkMode = useAssignmentModeStore((state) => state.setMode);
    const [overview, setOverview] = useState<dto.WorkspaceOverview | null>(null);
    const [currentTime, setCurrentTime] = useState(() => new Date());
    const [overviewError, setOverviewError] = useState('');
    const [documentsRefreshVersion, setDocumentsRefreshVersion] = useState(0);
    const [document, setDocument] = useState<{ id: string; kind: string } | null>(null);
    const overviewRequest = useRef(new CoalescedRequest<dto.WorkspaceOverview>());

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
        setDocument(null);
        setOverview(null);
        setOverviewError('');
        overviewRequest.current.invalidate();
    }, [userId]);

    const assignmentMode = workMode;

    const loadOverview = useCallback(() => {
        if (!ready || !userId) return Promise.resolve();
        return overviewRequest.current.refresh(() => GetOverview(assignmentMode), {
            onSuccess: (result) => { setOverview(result); setOverviewError(''); },
            onError: (error) => {
                const text = formatAppError(error, 'Не удалось загрузить рабочий стол');
                setOverview(null);
                setOverviewError(text);
                if (normalizeAppError(error).code === 'FORBIDDEN') {
                    setWorkMode('');
                }
                message.error(text);
            },
        });
    }, [ready, userId, assignmentMode, message, setWorkMode]);

    useEffect(() => {
        void loadOverview();
    }, [loadOverview]);

    useEffect(() => onAssignmentsChanged(() => { void loadOverview(); }), [loadOverview]);
    useEffect(() => onServerEvent((event) => {
        const resync = event.topic === 'documents' || event.topic === 'resync' || event.topic === 'access-changed';
        if (resync || (event.topic === 'document-changed' && (event.resource === 'assignments' || event.resource === 'document'))) void loadOverview();
    }), [loadOverview]);

    useEffect(() => {
        let timer: number | undefined;
        const onResume = () => {
            window.clearTimeout(timer);
            if (window.document.visibilityState === 'hidden') return;
            // Focus and visibility events can arrive together after restoring the window.
            timer = window.setTimeout(() => {
                setCurrentTime(new Date());
                void loadOverview();

                setDocumentsRefreshVersion((version) => version + 1);
            }, 150);
        };
        window.addEventListener('focus', onResume);
        window.document.addEventListener('visibilitychange', onResume);
        return () => {
            window.clearTimeout(timer);
            window.removeEventListener('focus', onResume);
            window.document.removeEventListener('visibilitychange', onResume);
        };
    }, [loadOverview]);

    useEffect(() => () => {
        overviewRequest.current.invalidate();
    }, []);

    const availableModes = (['execution', 'control'] as WorkMode[]).filter((mode) =>
        overview?.assignmentModes.includes(mode));
    const activeWorkMode = availableModes.includes(workMode as WorkMode) ? workMode : availableModes[0];
    const activeAssignmentMode = overview?.assignmentMode === activeWorkMode ? activeWorkMode : undefined;
    const changeWorkMode = (mode: WorkMode) => {
        setWorkMode(mode);
    };
    const openDocument = useCallback((id: string, kind: string) => setDocument({ id, kind }), []);
    const closeDocument = useCallback(() => { setDocument(null); setDocumentsRefreshVersion((version) => version + 1); }, []);
    const counts = overview?.assignmentCounts;
    const assignmentColumns: TableProps<dto.WorkspaceAssignment>['columns'] = [
        {
            title: 'Срок', dataIndex: 'deadline', key: 'deadline', width: 120,
            render: (value: string | undefined) => value ? dayjs(value).format('DD.MM.YYYY') : '—',
        },
        {
            title: 'Документ', key: 'document', width: 130,
            render: (_: unknown, item) => (
                <div>
                    <div>{item.documentNumber || '—'}</div>
                    <Text type="secondary">{item.documentDate ? dayjs(item.documentDate).format('DD.MM.YYYY') : '—'}</Text>
                </div>
            ),
        },
        {
            title: 'Содержание', dataIndex: 'documentContent', key: 'documentContent',
        },
        {
            title: 'Поручение', key: 'content', width: 240,
            render: (_: unknown, item) => (
                <div>
                    {item.type === 'acknowledgment' && <Tag>Ознакомление</Tag>}
                    {item.content}
                </div>
            ),
        },
        {
            title: 'Статус', dataIndex: 'status', key: 'status', width: 150,
            render: (value: string, item) => {
                if (item.type === 'acknowledgment') return <Tag color="orange">Ожидает</Tag>;
                const status = assignmentStatuses[value];
                return <Tag color={status?.color || 'default'}>{status?.label || value}</Tag>;
            },
        },
    ];

    if (!ready || (!overview && !overviewError)) {
        return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
    }

    return (
        <div className="workspace-dashboard">
            <Space align="start" wrap className="workspace-dashboard-header" style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}>
                <div>
                    <div className="workspace-heading">
                        <Title level={4} style={{ margin: 0 }}>{workspaceGreeting(currentTime, user)}</Title>
                        {availableModes.length > 1 && (
                            <span className="workspace-modes" role="group" aria-label="Режим рабочего стола">
                                {'( '}
                                <Button type="link" size="small" aria-pressed={activeWorkMode === 'execution'}
                                    className={`workspace-mode-link${activeWorkMode === 'execution' ? ' workspace-mode-link--active' : ''}`}
                                    onClick={() => changeWorkMode('execution')}>исполнение</Button>
                                {' \\ '}
                                <Button type="link" size="small" aria-pressed={activeWorkMode === 'control'}
                                    className={`workspace-mode-link${activeWorkMode === 'control' ? ' workspace-mode-link--active' : ''}`}
                                    onClick={() => changeWorkMode('control')}>контроль</Button>
                                {' )'}
                            </span>
                        )}
                    </div>
                    <Text type="secondary">{workspaceDate(currentTime)}</Text>
                </div>
                {userId && <DocumentSearchPanel key={`search:${userId}`} onOpenDocument={openDocument} />}
            </Space>
            {overviewError && <Card style={{ marginBottom: 12 }} extra={<Button onClick={() => { void loadOverview(); }}>Повторить</Button>}><Text type="danger">{overviewError}</Text></Card>}
            <div className={`workspace-overview-row${!activeAssignmentMode ? ' workspace-overview-row--right-only' : ''}`}>
                {activeAssignmentMode && (
                    <div className="workspace-work-column">
                        <Card className="workspace-metrics-panel">
                            <div className="workspace-metrics-grid">
                                {([
                                    ['Новые', counts?.new || 0, 'new'],
                                    ['В работе', counts?.inProgress || 0, 'in_progress'],
                                    ['На доработке', counts?.returned || 0, 'returned'],
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
                        <Card title="Поручения" extra={
                            <Button type="link" onClick={() => onOpenAssignments(activeAssignmentMode)}>Все поручения</Button>
                        }>
                            <Table<dto.WorkspaceAssignment> columns={assignmentColumns} dataSource={overview?.assignments || []}
                                rowKey="id" size="small" tableLayout="fixed" pagination={false} scroll={{ x: 800 }}
                                onRow={(item) => ({
                                    tabIndex: 0,
                                    style: { cursor: 'pointer' },
                                    onClick: () => openDocument(item.documentId, item.documentKind),
                                    onKeyDown: (event) => {
                                        if (event.key === 'Enter' || event.key === ' ') {
                                            event.preventDefault();
                                            openDocument(item.documentId, item.documentKind);
                                        }
                                    },
                                })}
                                locale={{ emptyText: 'Актуальных поручений нет' }} />
                        </Card>
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
                    {userId && <WorkspaceUserEventsPanel key={`events:${userId}`} currentTime={currentTime}
                        refreshVersion={documentsRefreshVersion} onOpenDocument={openDocument} />}
                    {userId && <RecentDocumentsPanel key={userId} refreshVersion={documentsRefreshVersion} onOpenDocument={openDocument} />}
                </div>
            </div>
            <DocumentViewModal open={!!document} onCancel={closeDocument} documentId={document?.id || ''}
                documentKind={document?.kind || ''} />
        </div>
    );
};

export default DashboardPage;
