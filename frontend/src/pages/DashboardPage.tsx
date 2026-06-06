import React, { useCallback, useEffect, useState } from 'react';
import {
    Typography, Card, Row, Col, Tag, Spin, App,
    Button, Empty
} from 'antd';
import {
    UserOutlined, ReloadOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_INCOMING_LETTER, getDocumentKindShortLabel } from '../constants/documentKinds';
import { resolveUserProfile, useAuthStore } from '../store/useAuthStore';
import { useCurrentAccessSummary } from '../hooks/useCurrentAccessSummary';

import DocumentViewModal from '../components/DocumentViewModal';
import { formatAppError } from '../utils/appError';

const { Title, Text } = Typography;

/**
 * Главная страница дашборда (панель управления).
 * Отображает статистику системы в зависимости от роли текущего пользователя.
 */
const DashboardPage: React.FC = () => {
    const { message } = App.useApp();
    const { user } = useAuthStore();
    const { summary: accessSummary, kinds: readableKinds, ready: accessReady } = useCurrentAccessSummary();
    const [stats, setStats] = useState<any>(null);
    const [loading, setLoading] = useState(false);
    const [pendingAcks, setPendingAcks] = useState<any[]>([]);

    // Состояние модального окна просмотра
    const [viewDocId, setViewDocId] = useState('');
    const [viewDocKind, setViewDocKind] = useState(DOCUMENT_KIND_INCOMING_LETTER);
    const [viewModalOpen, setViewModalOpen] = useState(false);

    const profile = resolveUserProfile(accessSummary?.systemPermissions || user?.systemPermissions, readableKinds, user?.isDocumentParticipant);

    const loadStats = useCallback(async () => {
        if (!accessReady) {
            return;
        }
        setLoading(true);
        try {
            const { GetActivity } = await import('../../wailsjs/go/services/DashboardService');
            const data = await GetActivity();
            setStats(data);

            // Загрузка ожидающих ознакомлений
            const { GetPendingForCurrentUser, GetAllActive } = await import('../../wailsjs/go/services/AcknowledgmentService');

            let acks: any[] = [];
            if (profile === 'clerk' || profile === 'mixed') {
                acks = await GetAllActive();
            } else if (profile === 'executor') {
                acks = await GetPendingForCurrentUser();
            }
            setPendingAcks(acks || []);
        } catch (err: unknown) {
            console.error(err);
            message.error(formatAppError(err, 'Ошибка загрузки дашборда'));
        } finally {
            setLoading(false);
        }
    }, [accessReady, message, profile]);

    useEffect(() => {
        if (accessReady) {
            loadStats();
        }
    }, [accessReady, loadStats]);

    // Определение отображения по текущей роли
    if (!accessReady || (loading && !stats)) {
        return <div style={{ textAlign: 'center', marginTop: 50 }}><Spin size="large" /></div>;
    }

    // --- Подкомпоненты ---

    const ExpiringList = ({ list, title = 'Истекающий срок исполнения' }: any) => (
        <Card title={title} variant="borderless" size="small" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
            {(!list || list.length === 0) ? (
                <Empty
                    description={
                        <span>
                            Срочных поручений нет. Проверьте этот блок позже или обновите дашборд.
                        </span>
                    }
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    {list.map((item: any) => {
                        const diff = dayjs(item.deadline).diff(dayjs(), 'day');
                        let color = 'green';
                        if (diff < 0) color = 'red';
                        else if (diff <= 3) color = 'orange';

                        return (
                            <div key={item.id || Math.random()} style={{ display: 'flex', gap: 12, paddingBottom: 12, borderBottom: '1px solid var(--app-border)' }}>
                                <div style={{ flexShrink: 0, marginTop: 2 }}>
                                    <Tag color={color} style={{ margin: 0 }}>
                                        {dayjs(item.deadline).format('DD.MM')}
                                    </Tag>
                                </div>
                                <div style={{ flex: 1, minWidth: 0 }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                                        <Text style={{ whiteSpace: 'pre-wrap', marginRight: 8, wordBreak: 'break-word' }}>{item.content}</Text>
                                        {item.documentNumber && profile !== 'admin' && (
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
                                    <div style={{ color: 'var(--app-text-secondary)' }}>
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
        const title = profile === 'clerk' || profile === 'mixed' ? "Все текущие ознакомления" : "Мои ознакомления";
        return (
            <Card title={title} variant="borderless" size="small" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
                {!hasItems ? (
                    <Empty
                        description={
                            <span>
                                Документов для ознакомления нет. Новые задачи появятся здесь после назначения.
                            </span>
                        }
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                    />
                ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                        {pendingAcks.map((item: any) => (
                            (() => {
                                return (
                                    <div key={item.id} style={{ paddingBottom: 12, borderBottom: '1px solid var(--app-border)' }}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 4 }}>
                                            <Text style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', marginRight: 8 }}>{item.content || 'Без описания'}</Text>
                                            <Tag
                                                style={{ cursor: profile === 'admin' ? 'default' : 'pointer', margin: 0, flexShrink: 0 }}
                                                onClick={(e) => {
                                                    if (profile === 'admin') {
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

                                        {profile === 'clerk' || profile === 'mixed' ? (
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
                                            <Text type="secondary" style={{ fontSize: 12 }}>
                                                Откройте карточку документа для подтверждения ознакомления.
                                            </Text>
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
        <Card variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
            <Empty description="Оперативная активность администратора не отображается. Используйте раздел статистики или журнал администрирования для контроля системы." />
        </Card>
    );

    return (
        <div style={{ padding: 24 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
                <Title level={4} style={{ margin: 0 }}>Текущая активность</Title>
                <Button icon={<ReloadOutlined />} onClick={loadStats} disabled={loading || !accessReady}>Обновить</Button>
            </div>

            {profile === 'admin' && renderAdminView()}
            {profile === 'clerk' && renderClerkView()}
            {profile === 'mixed' && renderMixedView()}
            {profile !== 'admin' && profile !== 'clerk' && profile !== 'mixed' && renderExecutorView()}

            {profile !== 'admin' && (
                <DocumentViewModal
                    open={viewModalOpen}
                    onCancel={() => setViewModalOpen(false)}
                    documentId={viewDocId}
                    documentKind={viewDocKind}
                    onAssignmentsChanged={loadStats}
                    onAcknowledgmentsChanged={loadStats}
                />
            )}
        </div>
    );
};

export default DashboardPage;
