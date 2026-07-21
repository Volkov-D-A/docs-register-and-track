import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Table, Button, Tag, Space, Popconfirm, App } from 'antd';
import { PlusOutlined, DeleteOutlined, EyeOutlined, CheckCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import AcknowledgmentModal from './AcknowledgmentModal';
import { useDocumentKindAccess } from '../hooks/useDocumentKindAccess';
import { formatAppError } from '../utils/appError';
import { isAcknowledgmentUserEvent, onUserEventsReceived } from '../events/userEvents';
import { CoalescedRequest } from '../utils/coalescedRequest';

/**
 * Свойства компонента AcknowledgmentList.
 */
interface AcknowledgmentListProps {
    documentId: string;
    documentKind: string;
}

/**
 * Компонент списка ознакомлений с документом.
 * @param documentId Идентификатор документа
 */
const AcknowledgmentList: React.FC<AcknowledgmentListProps> = ({ documentId, documentKind }) => {
    const { message } = App.useApp();
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const acknowledgmentsRequestRef = useRef(new CoalescedRequest<any[]>());
    const { user } = useAuthStore();
    const { hasAction, ready: accessReady } = useDocumentKindAccess();
    const canManageAcknowledgments = accessReady && hasAction(documentKind, 'acknowledge');

    const load = useCallback(async () => {
        if (!documentId || !canManageAcknowledgments) return;
        setLoading(true);
        return acknowledgmentsRequestRef.current.refresh(async () => {
            const { GetList } = await import('../../wailsjs/go/services/AcknowledgmentService');
            return (await GetList(documentId)) || [];
        }, {
            onSuccess: setData,
            onError: console.error,
            onSettled: () => setLoading(false),
        });
    }, [canManageAcknowledgments, documentId]);

    useEffect(() => () => acknowledgmentsRequestRef.current.invalidate(), []);

    useEffect(() => { load(); }, [load]);

    useEffect(() => onUserEventsReceived((events) => {
        if (events.some((event) => isAcknowledgmentUserEvent(event) && event.documentId === documentId)) {
            void load();
        }
    }), [documentId, load]);

    const onDelete = async (id: string) => {
        try {
            const { Delete } = await import('../../wailsjs/go/services/AcknowledgmentService');
            await Delete(id);
            message.success('Задача ознакомления удалена');
            load();
        } catch (err: unknown) {
            message.error(formatAppError(err));
        }
    };

    const columns = [
        {
            title: 'Дата создания', dataIndex: 'createdAt', key: 'createdAt', width: 140,
            render: (v: string) => dayjs(v).format('DD.MM.YYYY HH:mm'),
        },
        { title: 'Содержание', dataIndex: 'content', key: 'content' },
        {
            title: 'Создатель', dataIndex: 'creatorName', key: 'creatorName', width: 150
        },
        {
            title: 'Инфо об ознакомлении', key: 'users',
            render: (_: any, r: any) => {
                // Determine overall status
                const users = r.users || [];
                const total = users.length;
                const confirmed = users.filter((u: any) => u.confirmedAt).length;
                const isCompleted = r.completedAt;

                return (
                    <div>
                        <div style={{ marginBottom: 4 }}>
                            {isCompleted ? <Tag color="green">Завершено</Tag> : <Tag color="orange">В процессе</Tag>}
                            <span style={{ fontSize: 12 }}>({confirmed} из {total})</span>
                        </div>
                        <div style={{ fontSize: 11, color: 'var(--app-text-muted)', maxHeight: 60, overflowY: 'auto' }}>
                            {users.map((u: any) => {
                                const isOk = !!u.confirmedAt;
                                return (
                                    <div key={u.userId} style={{ marginBottom: 2 }}>
                                        {isOk ? <CheckCircleOutlined style={{ color: 'green', marginRight: 4 }} /> : <EyeOutlined style={{ color: 'var(--app-text-muted)', marginRight: 4 }} />}
                                        {u.userName}
                                        {isOk && <span style={{ marginLeft: 4, color: 'var(--app-text-muted)' }}>{dayjs(u.confirmedAt).format('DD.MM')}</span>}
                                    </div>
                                );
                            })}
                        </div>
                    </div>
                );
            }
        },
        {
            title: '', key: 'actions', width: 50,
            render: (_: any, r: any) => {
                const canDelete = canManageAcknowledgments && user?.id === r.creatorId;

                return (
                    <Space size={2}>
                        {canDelete && (
                            <Popconfirm
                                title="Удалить задачу ознакомления?"
                                description="Это действие нельзя отменить. Сотрудник больше не увидит эту задачу в списке ознакомления."
                                okText="Удалить"
                                cancelText="Отмена"
                                okButtonProps={{ danger: true }}
                                onConfirm={() => onDelete(r.id)}
                            >
                                <Button size="small" title="Удалить задачу ознакомления" icon={<DeleteOutlined />} danger />
                            </Popconfirm>
                        )}
                    </Space>
                );
            }
        },
    ];

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                {canManageAcknowledgments && (
                    <Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
                        На ознакомление
                    </Button>
                )}
            </div>

            <Table
                columns={columns}
                dataSource={data}
                rowKey="id"
                size="small"
                pagination={false}
                loading={loading || !accessReady}
            />

            <AcknowledgmentModal
                open={modalOpen}
                onCancel={() => setModalOpen(false)}
                onSuccess={load}
                documentId={documentId}
            />
        </div>
    );
};

export default AcknowledgmentList;
