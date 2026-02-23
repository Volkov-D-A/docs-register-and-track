import React, { useState, useEffect } from 'react';
import {
    Typography, Card, Row, Col, Statistic, Tag, Spin, App,
    Button, Empty, Select, DatePicker
} from 'antd';
import {
    ClockCircleOutlined, FileTextOutlined, CheckCircleOutlined,
    UserOutlined, DatabaseOutlined, ReloadOutlined, EyeOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';

import DocumentViewModal from '../components/DocumentViewModal';

const { Title, Text } = Typography;
const { Option } = Select;

const DashboardPage: React.FC = () => {
    const { message } = App.useApp();
    const { user, hasRole, currentRole } = useAuthStore();
    const [stats, setStats] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs().startOf('month'), dayjs().endOf('month')]);
    const [pendingAcks, setPendingAcks] = useState<any[]>([]);

    // Состояние модального окна просмотра
    const [viewDocId, setViewDocId] = useState('');
    const [viewDocType, setViewDocType] = useState<'incoming' | 'outgoing'>('incoming');
    const [viewModalOpen, setViewModalOpen] = useState(false);

    const loadStats = async () => {
        setLoading(true);
        try {
            const { GetStats } = await import('../../wailsjs/go/services/DashboardService');
            // Передаём текущую роль в GetStats
            // @ts-ignore
            const data = await GetStats(currentRole || '', dateRange[0].format('YYYY-MM-DD'), dateRange[1].format('YYYY-MM-DD'));
            setStats(data);

            // Загрузка ожидающих ознакомлений
            // @ts-ignore
            const { GetPendingForCurrentUser, GetAllActive } = await import('../../wailsjs/go/services/AcknowledgmentService');

            let acks = [];
            if (currentRole === 'admin' || currentRole === 'clerk') {
                acks = await GetAllActive();
            } else {
                acks = await GetPendingForCurrentUser();
            }
            setPendingAcks(acks || []);
        } catch (err: any) {
            console.error(err);
            message.error('Ошибка загрузки дашборда: ' + (err.message || String(err)));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadStats();
    }, [currentRole, dateRange]); // Перезагрузка при смене роли или периода

    const onAcknowledge = async (id: string) => {
        try {
            // @ts-ignore
            const { MarkConfirmed } = await import('../../wailsjs/go/services/AcknowledgmentService');
            await MarkConfirmed(id);
            message.success('Ознакомлен');
            loadStats();
        } catch (err: any) {
            message.error(err?.message || String(err));
        }
    };

    // Определение отображения по текущей роли
    const activeRole = currentRole || stats?.role || user?.roles?.[0] || 'executor';

    if (loading && !stats) {
        return <div style={{ textAlign: 'center', marginTop: 50 }}><Spin size="large" /></div>;
    }

    // --- Подкомпоненты ---

    const StatCard = ({ title, value, icon, color = '#1890ff', suffix = '' }: any) => (
        <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Statistic
                title={<span style={{ fontWeight: 500 }}>{title}</span>}
                value={value}
                prefix={icon && <span style={{ color, marginRight: 8, fontSize: 24 }}>{icon}</span>}
                suffix={suffix}
                styles={{ content: { fontWeight: 600 } }}
            />
        </Card>
    );

    const ExpiringList = ({ list, title = 'Истекающий срок исполнения' }: any) => (
        <Card title={title} variant="borderless" size="small" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            {(!list || list.length === 0) ? (
                <Empty description="Нет поручений" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    {list.map((item: any) => {
                        const diff = dayjs(item.deadline).diff(dayjs(), 'day');
                        let color = 'green';
                        if (diff < 0) color = 'red';
                        else if (diff <= 3) color = 'orange';

                        return (
                            <div key={item.id || Math.random()} style={{ display: 'flex', gap: 12, paddingBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
                                <div style={{ flexShrink: 0, marginTop: 2 }}>
                                    <Tag color={color} style={{ margin: 0 }}>
                                        {dayjs(item.deadline).format('DD.MM')}
                                    </Tag>
                                </div>
                                <div style={{ flex: 1, minWidth: 0 }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                                        <Text style={{ whiteSpace: 'pre-wrap', marginRight: 8, wordBreak: 'break-word' }}>{item.content}</Text>
                                        {item.documentNumber && (
                                            <Tag
                                                style={{ cursor: 'pointer', margin: 0, flexShrink: 0 }}
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    setViewDocId(item.documentId);
                                                    setViewDocType(item.documentType);
                                                    setViewModalOpen(true);
                                                }}
                                            >
                                                {item.documentNumber}
                                            </Tag>
                                        )}
                                    </div>
                                    <div style={{ color: 'rgba(0, 0, 0, 0.45)' }}>
                                        {item.executorName && <span style={{ marginRight: 10 }}><UserOutlined /> {item.executorName}</span>}
                                        <span style={{ fontSize: 12 }}>{
                                            item.status === 'new' ? 'Новое' :
                                                item.status === 'in_progress' ? 'В работе' : item.status
                                        }</span>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </Card>
    );

    const PendingAcksList = () => {
        const hasItems = pendingAcks && pendingAcks.length > 0;
        const title = (activeRole === 'admin' || activeRole === 'clerk') ? "Все текущие ознакомления" : "Мои ознакомления";
        return (
            <Card title={title} variant="borderless" size="small" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)', borderLeft: '4px solid #faad14' }}>
                {!hasItems ? (
                    <Empty description="Нет документов" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                        {pendingAcks.map((item: any) => (
                            <div key={item.id} style={{ paddingBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                                    <Text style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', marginRight: 8 }}>{item.content || 'Без описания'}</Text>
                                    <Tag
                                        style={{ cursor: 'pointer', margin: 0, flexShrink: 0 }}
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setViewDocId(item.documentId);
                                            setViewDocType(item.documentType as 'incoming' | 'outgoing');
                                            setViewModalOpen(true);
                                        }}
                                    >
                                        {item.documentNumber || (item.documentType === 'incoming' ? 'Bx' : 'Исх')}
                                    </Tag>
                                </div>

                                {(activeRole === 'admin' || activeRole === 'clerk') ? (
                                    <div style={{ marginTop: 8 }}>
                                        {item.users && item.users.length > 0 ? (
                                            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                                                {item.users.map((u: any) => (
                                                    <Tag key={u.userId} color={u.confirmedAt ? 'green' : 'default'} style={{ margin: 0 }}>
                                                        {u.userName}
                                                    </Tag>
                                                ))}
                                            </div>
                                        ) : <Text type="secondary" style={{ fontSize: 12 }}>Нет участников</Text>}
                                    </div>
                                ) : (
                                    <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-start' }}>
                                        <Button size="small" type="primary" onClick={() => onAcknowledge(item.id)}>ознакомлен</Button>
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                )}
            </Card>
        );
    };

    // --- Представления ---

    const renderExecutorView = () => (
        <>
            <Title level={4}>Моя статистика</Title>
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
                <Col xs={24} sm={6}>
                    <StatCard title="Новые" value={stats?.myAssignmentsNew || 0} icon={<FileTextOutlined />} color="#69c0ff" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="В работе" value={stats?.myAssignmentsInProgress || 0} icon={<ClockCircleOutlined />} color="#ffc069" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="Просрочено" value={stats?.myAssignmentsOverdue || 0} icon={<ClockCircleOutlined />} color="#ff7875" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard
                        title="Завершено"
                        value={stats?.myAssignmentsFinished || 0}
                        icon={<CheckCircleOutlined />}
                        color="#95de64"
                        suffix={
                            (stats?.myAssignmentsFinishedLate > 0) ? (
                                <span style={{ marginLeft: 8, fontSize: 'inherit' }}>
                                    <ClockCircleOutlined style={{ color: '#ff7875', marginRight: 4, fontSize: 16 }} />
                                    <span style={{ color: 'black' }}>{stats.myAssignmentsFinishedLate}</span>
                                </span>
                            ) : null
                        }
                    />
                </Col>
            </Row>

            <Row gutter={[16, 16]}>
                <Col xs={24} md={12}>
                    <PendingAcksList />
                </Col>
                <Col xs={24} md={12}>
                    <ExpiringList list={stats?.expiringAssignments} title="Срочные поручения" />
                </Col>
            </Row>
        </>
    );

    const renderClerkView = () => (
        <>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>
                    Статистика за период
                </Title>
                <DatePicker.RangePicker
                    value={dateRange}
                    onChange={(dates) => {
                        if (dates && dates[0] && dates[1]) {
                            setDateRange([dates[0], dates[1]]);
                        }
                    }}
                    allowClear={false}
                />
            </div>
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
                <Col xs={24} sm={6}>
                    <StatCard title="Входящие" value={stats?.incomingCount || 0} icon={<FileTextOutlined />} color="#95de64" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="Исходящие" value={stats?.outgoingCount || 0} icon={<FileTextOutlined />} color="#b37feb" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="Просрочено" value={stats?.allAssignmentsOverdue || 0} icon={<ClockCircleOutlined />} color="#ff7875" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard
                        title="Завершено"
                        value={stats?.allAssignmentsFinished || 0}
                        icon={<CheckCircleOutlined />}
                        color="#95de64"
                        suffix={
                            (stats?.allAssignmentsFinishedLate > 0) ? (
                                <span style={{ marginLeft: 8, fontSize: 'inherit' }}>
                                    <ClockCircleOutlined style={{ color: '#ff7875', marginRight: 4, fontSize: 16 }} />
                                    <span style={{ color: 'black' }}>{stats.allAssignmentsFinishedLate}</span>
                                </span>
                            ) : null
                        }
                    />
                </Col>
            </Row>

            <Row gutter={[16, 16]}>
                <Col xs={24} md={12}>
                    <PendingAcksList />
                </Col>
                <Col xs={24} md={12}>
                    <ExpiringList list={stats?.expiringAssignments} title="Все срочные" />
                </Col>
            </Row>
        </>
    );

    const renderAdminView = () => (
        <>
            <Title level={4}>Система</Title>
            <Row gutter={[16, 16]}>
                <Col xs={24} sm={8}>
                    <StatCard title="Пользователи" value={stats?.userCount || 0} icon={<UserOutlined />} color="#1890ff" />
                </Col>
                <Col xs={24} sm={8}>
                    <StatCard title="Документы" value={stats?.totalDocuments || 0} icon={<FileTextOutlined />} />
                </Col>
                <Col xs={24} sm={8}>
                    <StatCard title="БД" value={stats?.dbSize || "N/A"} icon={<DatabaseOutlined />} color="#52c41a" />
                </Col>
            </Row>
        </>
    );

    return (
        <div style={{ padding: 24 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
                <Title level={3} style={{ margin: 0 }}>Дашборд</Title>
                <Button icon={<ReloadOutlined />} onClick={loadStats} disabled={loading}>Обновить</Button>
            </div>

            {activeRole === 'admin' && renderAdminView()}
            {activeRole === 'clerk' && renderClerkView()}
            {activeRole !== 'admin' && activeRole !== 'clerk' && renderExecutorView()}

            <DocumentViewModal
                open={viewModalOpen}
                onCancel={() => setViewModalOpen(false)}
                documentId={viewDocId}
                documentType={viewDocType}
            />
        </div>
    );
};

export default DashboardPage;
