import React, { useState, useEffect } from 'react';
import {
    Typography, Card, Row, Col, Statistic, List, Tag, Spin, message,
    Button, Empty, Select
} from 'antd';
import {
    ClockCircleOutlined, FileTextOutlined, CheckCircleOutlined,
    UserOutlined, DatabaseOutlined, ReloadOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';

const { Title, Text } = Typography;
const { Option } = Select;

const DashboardPage: React.FC = () => {
    const { user, hasRole, currentRole } = useAuthStore();
    const [stats, setStats] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const [period, setPeriod] = useState<string>('month');

    const loadStats = async () => {
        setLoading(true);
        try {
            const { GetStats } = await import('../../wailsjs/go/services/DashboardService');
            // Pass currentRole to GetStats. If null/undefined, backend handles default.
            // @ts-ignore
            const data = await GetStats(currentRole || '', period);
            setStats(data);
        } catch (err: any) {
            console.error(err);
            message.error('Ошибка загрузки дашборда: ' + (err.message || String(err)));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadStats();
    }, [currentRole, period]); // Reload when role or period changes

    // Use currentRole for view determination, fallback to stats.role if needed
    const activeRole = currentRole || stats?.role || user?.roles?.[0] || 'executor';

    if (loading && !stats) {
        return <div style={{ textAlign: 'center', marginTop: 50 }}><Spin size="large" /></div>;
    }

    // --- Sub-components --

    const StatCard = ({ title, value, icon, color = '#1890ff', suffix = '' }: any) => (
        <Card bordered={false} style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Statistic
                title={<span style={{ fontWeight: 500 }}>{title}</span>}
                value={value}
                prefix={icon && <span style={{ color, marginRight: 8, fontSize: 24 }}>{icon}</span>}
                suffix={suffix}
                valueStyle={{ fontWeight: 600 }}
            />
        </Card>
    );

    const ExpiringList = ({ list, title = 'Истекающий срок исполнения' }: any) => (
        <Card title={title} bordered={false} style={{ marginTop: 24, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <List
                dataSource={list || []}
                renderItem={(item: any) => {
                    const diff = dayjs(item.deadline).diff(dayjs(), 'day');
                    let color = 'green';
                    if (diff < 0) color = 'red';
                    else if (diff <= 3) color = 'orange';

                    return (
                        <List.Item>
                            <List.Item.Meta
                                avatar={
                                    <Tag color={color}>
                                        {dayjs(item.deadline).format('DD.MM')}
                                    </Tag>
                                }
                                title={
                                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                        <span>{item.content}</span>
                                        {item.documentNumber && <Tag>{item.documentNumber}</Tag>}
                                    </div>
                                }
                                description={
                                    <div>
                                        {item.executorName && <span style={{ marginRight: 10 }}><UserOutlined /> {item.executorName}</span>}
                                        <span style={{ fontSize: 12 }}>Статус: {
                                            item.status === 'new' ? 'Новое' :
                                                item.status === 'in_progress' ? 'В работе' : item.status
                                        }</span>
                                    </div>
                                }
                            />
                        </List.Item>
                    );
                }}
                locale={{ emptyText: <Empty description="Нет срочных поручений" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
            />
        </Card>
    );

    // --- Views ---

    const renderExecutorView = () => (
        <>
            <Title level={4}>Моя статистика</Title>
            <Row gutter={[16, 16]}>
                <Col xs={24} sm={6}>
                    <StatCard title="Новые" value={stats?.myAssignmentsNew || 0} icon={<FileTextOutlined />} color="#69c0ff" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="В работе" value={stats?.myAssignmentsInProgress || 0} icon={<ClockCircleOutlined />} color="#ffc069" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="Просрочено (активные)" value={stats?.myAssignmentsOverdue || 0} icon={<ClockCircleOutlined />} color="#ff7875" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard
                        title="Завершено"
                        value={stats?.myAssignmentsFinished || 0}
                        icon={<CheckCircleOutlined />}
                        color="#95de64"
                        suffix={
                            (stats?.myAssignmentsFinishedLate > 0) ? (
                                <span style={{ marginLeft: 12, fontSize: 'inherit' }}>
                                    <ClockCircleOutlined style={{ color: '#ff7875', marginRight: 4, fontSize: 24 }} />
                                    <span style={{ color: 'black' }}>{stats.myAssignmentsFinishedLate}</span>
                                </span>
                            ) : null
                        }
                    />
                </Col>
            </Row>

            <ExpiringList list={stats?.expiringAssignments} title="Мои срочные поручения" />
        </>
    );

    const renderClerkView = () => (
        <>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Title level={4} style={{ margin: 0 }}>
                    Статистика: {
                        period === 'month' ? 'Текущий месяц' :
                            period === 'quarter' ? 'Текущий квартал' :
                                'Текущий год'
                    }
                </Title>
                <Select value={period} style={{ width: 160 }} onChange={setPeriod}>
                    <Option value="month">Текущий месяц</Option>
                    <Option value="quarter">Текущий квартал</Option>
                    <Option value="year">Текущий год</Option>
                </Select>
            </div>
            <Row gutter={[16, 16]}>
                <Col xs={24} sm={6}>
                    <StatCard title="Входящие" value={stats?.incomingCountMonth || 0} icon={<FileTextOutlined />} color="#95de64" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="Исходящие" value={stats?.outgoingCountMonth || 0} icon={<FileTextOutlined />} color="#b37feb" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard title="Просрочено (всего)" value={stats?.allAssignmentsOverdue || 0} icon={<ClockCircleOutlined />} color="#ff7875" />
                </Col>
                <Col xs={24} sm={6}>
                    <StatCard
                        title="Завершено (всего)"
                        value={stats?.allAssignmentsFinished || 0}
                        icon={<CheckCircleOutlined />}
                        color="#95de64"
                        suffix={
                            (stats?.allAssignmentsFinishedLate > 0) ? (
                                <span style={{ marginLeft: 12, fontSize: 'inherit' }}>
                                    <ClockCircleOutlined style={{ color: '#ff7875', marginRight: 4, fontSize: 24 }} />
                                    <span style={{ color: 'black' }}>{stats.allAssignmentsFinishedLate}</span>
                                </span>
                            ) : null
                        }
                    />
                </Col>
            </Row>

            <ExpiringList list={stats?.expiringAssignments} title="Все поручения с истекающим сроком" />
        </>
    );

    const renderAdminView = () => (
        <>
            <Title level={4}>Системная статистика</Title>
            <Row gutter={[16, 16]}>
                <Col xs={24} sm={8}>
                    <StatCard title="Пользователи" value={stats?.userCount || 0} icon={<UserOutlined />} color="#1890ff" />
                </Col>
                <Col xs={24} sm={8}>
                    <StatCard title="Всего документов" value={stats?.totalDocuments || 0} icon={<FileTextOutlined />} />
                </Col>
                <Col xs={24} sm={8}>
                    <StatCard title="Размер БД" value={stats?.dbSize || "N/A"} icon={<DatabaseOutlined />} color="#52c41a" />
                </Col>
            </Row>

            {/* Admin also sees clerk view stats for overview */}
            <div style={{ marginTop: 24 }}>
                <Title level={5}>Документация и контроль</Title>
                <Row gutter={[16, 16]}>
                    <Col xs={24} sm={12}>
                        <StatCard title="Входящие (мес)" value={stats?.incomingCountMonth || 0} icon={<FileTextOutlined />} color="#52c41a" />
                    </Col>
                    <Col xs={24} sm={12}>
                        <StatCard title="Исходящие (мес)" value={stats?.outgoingCountMonth || 0} icon={<FileTextOutlined />} color="#722ed1" />
                    </Col>
                </Row>
            </div>
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
        </div>
    );
};

export default DashboardPage;
