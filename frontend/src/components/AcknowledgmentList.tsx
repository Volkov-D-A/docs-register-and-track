import React, { useEffect, useState } from 'react';
import { Table, Button, Tag, Space, Popconfirm, message, Tooltip } from 'antd';
import { PlusOutlined, DeleteOutlined, EyeOutlined, CheckCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';
import AcknowledgmentModal from './AcknowledgmentModal';

interface AcknowledgmentListProps {
    documentId: string;
    documentType: 'incoming' | 'outgoing';
}

const AcknowledgmentList: React.FC<AcknowledgmentListProps> = ({ documentId, documentType }) => {
    const [data, setData] = useState<any[]>([]);
    const [loading, setLoading] = useState(false);
    const [modalOpen, setModalOpen] = useState(false);
    const { user, hasRole } = useAuthStore();

    const load = async () => {
        if (!documentId) return;
        setLoading(true);
        try {
            // @ts-ignore
            const { GetList } = await import('../../wailsjs/go/services/AcknowledgmentService');
            const result = await GetList(documentId);
            setData(result || []);
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, [documentId]);

    const onDelete = async (id: string) => {
        try {
            // @ts-ignore
            const { Delete } = await import('../../wailsjs/go/services/AcknowledgmentService');
            await Delete(id);
            message.success('Удалено');
            load();
        } catch (err: any) {
            message.error(err?.message || String(err));
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
                        <div style={{ fontSize: 11, color: '#666', maxHeight: 60, overflowY: 'auto' }}>
                            {users.map((u: any) => {
                                const isOk = !!u.confirmedAt;
                                return (
                                    <div key={u.userId} style={{ marginBottom: 2 }}>
                                        {isOk ? <CheckCircleOutlined style={{ color: 'green', marginRight: 4 }} /> : <EyeOutlined style={{ color: '#ccc', marginRight: 4 }} />}
                                        {u.userName}
                                        {isOk && <span style={{ marginLeft: 4, color: '#999' }}>{dayjs(u.confirmedAt).format('DD.MM')}</span>}
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
                const isAdmin = hasRole('admin');
                const isClerk = hasRole('clerk');
                const canDelete = isAdmin || (isClerk && user?.id === r.creatorId);

                return (
                    <Space size={2}>
                        {canDelete && (
                            <Popconfirm title="Удалить задачу ознакомления?" onConfirm={() => onDelete(r.id)}>
                                <Button size="small" icon={<DeleteOutlined />} danger />
                            </Popconfirm>
                        )}
                    </Space>
                );
            }
        },
    ];

    const canCreate = hasRole('admin') || hasRole('clerk');

    return (
        <div>
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                {canCreate && (
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
                loading={loading}
            />

            <AcknowledgmentModal
                open={modalOpen}
                onCancel={() => setModalOpen(false)}
                onSuccess={load}
                documentId={documentId}
                documentType={documentType}
            />
        </div>
    );
};

export default AcknowledgmentList;
