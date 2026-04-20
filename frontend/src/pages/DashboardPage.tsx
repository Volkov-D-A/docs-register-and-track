import React, { useState, useEffect } from 'react';
import {
    Typography, Card, Row, Col, Tag, Spin, App,
    Button, Empty
} from 'antd';
import {
    ClockCircleOutlined, UserOutlined, ReloadOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_INCOMING_LETTER, getDocumentKindShortLabel } from '../constants/documentKinds';
import { useAuthStore } from '../store/useAuthStore';

import DocumentViewModal from '../components/DocumentViewModal';

const { Title, Text } = Typography;

/**
 * Главная страница дашборда (панель управления).
 * Отображает статистику системы в зависимости от роли текущего пользователя.
 */
const DashboardPage: React.FC = () => {
    const { message } = App.useApp();
    const { user } = useAuthStore();
    const [stats, setStats] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const [pendingAcks, setPendingAcks] = useState<any[]>([]);

    // Состояние модального окна просмотра
    const [viewDocId, setViewDocId] = useState('');
    const [viewDocKind, setViewDocKind] = useState(DOCUMENT_KIND_INCOMING_LETTER);
    const [viewModalOpen, setViewModalOpen] = useState(false);

    const loadStats = async () => {
        setLoading(true);
        try {
            const { GetStats } = await import('../../wailsjs/go/services/DashboardService');
            const data = await GetStats('', '', '');
            setStats(data);

            // Загрузка ожидающих ознакомлений
            // @ts-ignore
            const { GetPendingForCurrentUser, GetAllActive } = await import('../../wailsjs/go/services/AcknowledgmentService');

            let acks: any[] = [];
            if (data?.role === 'clerk') {
                acks = await GetAllActive();
            } else if (data?.role === 'executor' || data?.role === 'mixed') {
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
    }, []);

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
    const activeRole = stats?.role || 'executor';

    if (loading && !stats) {
        return <div style={{ textAlign: 'center', marginTop: 50 }}><Spin size="large" /></div>;
    }

    // --- Подкомпоненты ---

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
                                        {item.documentNumber && activeRole !== 'admin' && (
                                            <Tag
                                                style={{ cursor: 'pointer', margin: 0, flexShrink: 0 }}
                                                onClick={(e) => {
                                                    e.stopPropagation();
                                                    setViewDocId(item.documentId);
                                                    setViewDocKind(item.documentKind);
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
        const title = activeRole === 'clerk' ? "Все текущие ознакомления" : "Мои ознакомления";
        return (
            <Card title={title} variant="borderless" size="small" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
                {!hasItems ? (
                    <Empty description="Нет документов" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                        {pendingAcks.map((item: any) => (
                            (() => {
                                const currentAckUser = item.users?.find((u: any) => u.userId === user?.id);
                                const canSelfAcknowledge = !!currentAckUser && !currentAckUser.confirmedAt;

                                return (
                                    <div key={item.id} style={{ paddingBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                                            <Text style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', marginRight: 8 }}>{item.content || 'Без описания'}</Text>
                                            <Tag
                                                style={{ cursor: activeRole === 'admin' ? 'default' : 'pointer', margin: 0, flexShrink: 0 }}
                                                onClick={(e) => {
                                                    if (activeRole === 'admin') {
                                                        return;
                                                    }
                                                    e.stopPropagation();
                                                    setViewDocId(item.documentId);
                                                    setViewDocKind(item.documentKind);
                                                    setViewModalOpen(true);
                                                }}
                                            >
                                                {item.documentNumber || getDocumentKindShortLabel(item.documentKind)}
                                            </Tag>
                                        </div>

                                        {activeRole === 'clerk' ? (
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
                                                {canSelfAcknowledge && (
                                                    <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-start' }}>
                                                        <Button size="small" type="primary" onClick={() => onAcknowledge(item.id)}>ознакомлен</Button>
                                                    </div>
                                                )}
                                            </div>
                                        ) : (
                                            <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-start' }}>
                                                <Button size="small" type="primary" onClick={() => onAcknowledge(item.id)}>ознакомлен</Button>
                                            </div>
                                        )}
                                    </div>
                                );
                            })()
                        ))}
                    </div>
                )}
            </Card>
        );
    };

    // --- Представления ---

    const renderExecutorView = () => (
        <>
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

    const renderMixedView = () => (
        <>
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

    const renderAdminView = () => (
        <Card variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
            <Empty description="Для администратора оперативная активность не отображается" />
        </Card>
    );

    return (
        <div style={{ padding: 24 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
                <Title level={4} style={{ margin: 0 }}>Текущая активность</Title>
                <Button icon={<ReloadOutlined />} onClick={loadStats} disabled={loading}>Обновить</Button>
            </div>

            {activeRole === 'admin' && renderAdminView()}
            {activeRole === 'clerk' && renderClerkView()}
            {activeRole === 'mixed' && renderMixedView()}
            {activeRole !== 'admin' && activeRole !== 'clerk' && activeRole !== 'mixed' && renderExecutorView()}

            {activeRole !== 'admin' && (
                <DocumentViewModal
                    open={viewModalOpen}
                    onCancel={() => setViewModalOpen(false)}
                    documentId={viewDocId}
                    documentKind={viewDocKind}
                />
            )}
        </div>
    );
};

export default DashboardPage;
