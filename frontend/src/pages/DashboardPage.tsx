import React, { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, Card, Col, Drawer, List, Pagination, Radio, Row, Space, Spin, Statistic, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { dto } from '../../wailsjs/go/models';
import { GetOverview, ListAcknowledgments } from '../../wailsjs/go/services/WorkspaceService';
import { GetCurrentUserEvents } from '../../wailsjs/go/services/UserEventService';
import { models } from '../../wailsjs/go/models';
import DocumentViewModal from '../components/DocumentViewModal';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';
import { useAuthStore } from '../store/useAuthStore';
import { useRegisterDocumentStore } from '../store/useRegisterDocumentStore';
import { onAssignmentsChanged } from '../events/assignmentEvents';
import { onServerEvent } from '../events/serverEvents';
import { formatAppError, normalizeAppError } from '../utils/appError';
import { CoalescedRequest } from '../utils/coalescedRequest';

const { Title, Text } = Typography;
type WorkMode = 'execution' | 'control';
type AssignmentMetric = 'new' | 'overdue' | 'due_soon' | 'acceptance';

type DashboardPageProps = {
    onOpenAssignments: (mode: WorkMode, metric?: AssignmentMetric) => void;
    onOpenRegister: (kindCode: string, page: string) => void;
};

const modeOptions = [
    { label: 'Исполнение', value: 'execution' },
    { label: 'Контроль', value: 'control' },
];

const DashboardPage: React.FC<DashboardPageProps> = ({ onOpenAssignments, onOpenRegister }) => {
    const { message } = App.useApp();
    const userId = useAuthStore((state) => state.user?.id);
    const { ready, registrationKinds } = useCurrentAccessSummary();
    const [assignmentMode, setAssignmentMode] = useState<WorkMode | ''>('');
    const [acknowledgmentMode, setAcknowledgmentMode] = useState<WorkMode | ''>('');
    const [overview, setOverview] = useState<dto.WorkspaceOverview | null>(null);
    const [loading, setLoading] = useState(false);
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
    const assignmentTitle = activeAssignmentMode === 'control' ? 'Поручения под контролем' : 'Мои поручения';
    const acknowledgmentTitle = activeAcknowledgmentMode === 'control' ? 'Контроль ознакомлений' : 'Мои ознакомления';
    const counts = overview?.assignmentCounts;

    if (!ready || (loading && !overview)) {
        return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
    }

    return (
        <div style={{ padding: 8 }}>
            <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 20 }}>
                <Title level={4} style={{ margin: 0 }}>Рабочий стол</Title>
                <Button icon={<ReloadOutlined />} loading={loading} onClick={() => { void loadOverview(); void loadEvents(); }}>Обновить</Button>
            </Space>
            {overviewError && <Card style={{ marginBottom: 16 }}><Text type="danger">{overviewError}</Text></Card>}
            {activeAssignmentMode && (
                <Card title={assignmentTitle} extra={overview && overview.assignmentModes.length > 1 && (
                    <Radio.Group size="small" options={modeOptions} value={activeAssignmentMode}
                        onChange={(event) => setAssignmentMode(event.target.value as WorkMode)} />
                )} style={{ marginBottom: 16 }}>
                    <Row gutter={[12, 12]} style={{ marginBottom: 20 }}>
                        {([
                            ['Новые', counts?.new || 0, 'new'],
                            ['Просрочены', counts?.overdue || 0, 'overdue'],
                            ['Срок в ближайшие 3 дня', counts?.dueSoon || 0, 'due_soon'],
                            ...(activeAssignmentMode === 'control' ? [['Ожидают приёмки', counts?.awaitingAcceptance || 0, 'acceptance']] : []),
                        ] as [string, number, AssignmentMetric][]).map(([title, value, metric]) => (
                            <Col key={metric} xs={12} md={6}>
                                <Card size="small" hoverable role="button" tabIndex={0}
                                    onClick={() => onOpenAssignments(activeAssignmentMode, metric)}
                                    onKeyDown={(event) => { if (event.key === 'Enter') onOpenAssignments(activeAssignmentMode, metric); }}>
                                    <Statistic title={title} value={value} />
                                </Card>
                            </Col>
                        ))}
                    </Row>
                    <List dataSource={overview?.assignments || []} locale={{ emptyText: 'Актуальных поручений нет' }}
                        renderItem={(item) => <List.Item actions={[<Button key="open" type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>]}>
                            <List.Item.Meta title={<Space><Text>{item.content}</Text>{item.deadline && <Tag>{dayjs(item.deadline).format('DD.MM.YYYY')}</Tag>}</Space>}
                                description={`${item.documentNumber || 'Документ без номера'} · ${item.executorName || 'Исполнитель не указан'}`} />
                        </List.Item>} />
                    <Button type="link" onClick={() => onOpenAssignments(activeAssignmentMode)}>Все поручения</Button>
                </Card>
            )}
            {activeAcknowledgmentMode && (
                <Card title={acknowledgmentTitle} extra={overview && overview.acknowledgmentModes.length > 1 && (
                    <Radio.Group size="small" options={modeOptions} value={activeAcknowledgmentMode}
                        onChange={(event) => { setAcknowledgmentMode(event.target.value as WorkMode); setAckPage(1); }} />
                )} style={{ marginBottom: 16 }}>
                    <Button type="link" onClick={openAcknowledgments}>Ожидают внимания: {overview?.acknowledgmentCount || 0}</Button>
                    <List dataSource={overview?.acknowledgments || []} locale={{ emptyText: 'Актуальных ознакомлений нет' }}
                        renderItem={(item) => <List.Item actions={[<Button key="open" type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>]}>
                            <List.Item.Meta title={item.content || 'Ознакомление'} description={item.documentNumber || 'Документ без номера'} />
                        </List.Item>} />
                    <Button type="link" onClick={openAcknowledgments}>Все ознакомления</Button>
                </Card>
            )}
            {registrationKinds.length > 0 && (
                <Card title="Быстрая регистрация" style={{ marginBottom: 16 }}>
                    <Space wrap>{registrationKinds.map((kind) => (
                        <Button key={kind.code} onClick={() => {
                            useRegisterDocumentStore.getState().requestOpen(kind.code);
                            onOpenRegister(kind.code, kind.pageKey);
                        }}>{kind.label}</Button>
                    ))}</Space>
                </Card>
            )}
            <Card title="Новое для меня">
                <List dataSource={events} locale={{ emptyText: 'Новых событий нет' }}
                    renderItem={(item) => <List.Item actions={[<Button key="open" type="link" onClick={() => openDocument(item.documentId, item.documentKind)}>Открыть</Button>]}>
                        <List.Item.Meta title={item.title} description={item.message} />
                    </List.Item>} />
            </Card>
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
