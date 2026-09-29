import React from 'react';
import { Button, Popconfirm, Space, Tag } from 'antd';
import { DeleteOutlined, EditOutlined, SyncOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

type BuildAssignmentColumnsParams = {
    canManageAssignments: boolean;
    currentUserId?: string;
    onEdit: (assignment: any) => void;
    onDelete: (id: string) => void;
    onManageSeries: (assignment: any) => void;
};

export const buildAssignmentColumns = ({
    canManageAssignments,
    currentUserId,
    onEdit,
    onDelete,
    onManageSeries,
}: BuildAssignmentColumnsParams) => [
    {
        title: 'Дата',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 100,
        render: (value: string) => dayjs(value).format('DD.MM.YYYY'),
    },
    { title: 'Тип', key: 'type', width: 130, render: (_: any, r: any) => r.type === 'acknowledgment' ? 'Ознакомление' : 'Исполнение' },
    { title: 'Содержание', dataIndex: 'content', key: 'content', render: (content: string, record: any) => record.type === 'acknowledgment' ? 'Ознакомиться с документом' : content },
    {
        title: 'Исполнитель / адресаты',
        key: 'executorName',
        width: 200,
        render: (_: any, record: any) => (
            <div>
                <div>{record.type === 'acknowledgment' ? (record.users || []).map((u: any) => `${u.userName}${u.confirmedAt ? ' ✓' : ''}`).join(', ') : record.executorName}</div>
                {record.coExecutors && record.coExecutors.length > 0 && (
                    <div style={{ fontSize: '11px', color: 'var(--app-text-muted)' }}>
                        + {record.coExecutors.map((user: any) => user.fullName).join(', ')}
                    </div>
                )}
            </div>
        ),
    },
    {
        title: 'Срок',
        dataIndex: 'deadline',
        key: 'deadline',
        width: 100,
        render: (value: string) => value ? dayjs(value).format('DD.MM.YYYY') : '',
    },
    {
        title: 'Статус',
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (status: string, record: any) => {
            if (record.type === 'acknowledgment') {
                const users = record.users || [];
                return <Tag color={record.status === 'finished' ? 'green' : 'orange'}>{record.status === 'finished' ? 'Ознакомлены' : 'Ожидает'} ({users.filter((u: any) => u.confirmedAt).length}/{users.length})</Tag>;
            }
            let color = 'default';
            let text = status;
            const isOverdue = status === 'completed' && record.completedAt && record.deadline
                && dayjs(record.completedAt).isAfter(dayjs(record.deadline), 'day');

            switch (status) {
                case 'new': color = 'blue'; text = 'Новое'; break;
                case 'in_progress': color = 'orange'; text = 'В работе'; break;
                case 'completed':
                    if (isOverdue) {
                        color = 'red';
                        text = 'Исполнено (просрочено)';
                    } else {
                        color = 'green';
                        text = 'Исполнено';
                    }
                    break;
                case 'finished': color = 'geekblue'; text = 'Завершён'; break;
                case 'cancelled': color = 'red'; text = 'Отменено'; break;
                case 'returned': color = 'volcano'; text = 'Возврат'; break;
            }
            return <Tag color={color}>{text}</Tag>;
        },
    },
    {
        title: '',
        key: 'actions',
        width: 150,
        render: (_: any, record: any) => {
            const acknowledgment = record.type === 'acknowledgment';
            const canEdit = canManageAssignments && record.status !== 'finished';
            const canDelete = acknowledgment ? canManageAssignments && record.creatorId === currentUserId : canEdit && !record.seriesId;

            return (
                <Space size={2}>
                    {canManageAssignments && (
                        <>
                            {canEdit && <Button size="small" title="Редактировать поручение" icon={<EditOutlined />} onClick={() => onEdit(record)} />}
                            {record.seriesId && <Button size="small" title="Управление серией" icon={<SyncOutlined />} onClick={() => onManageSeries(record)} />}
                            {canDelete && <Popconfirm
                                title="Удалить поручение?"
                                description="Это действие нельзя отменить. Поручение исчезнет из документа и списка исполнителя."
                                okText="Удалить"
                                cancelText="Отмена"
                                okButtonProps={{ danger: true }}
                                onConfirm={() => onDelete(record.id)}
                            >
                                <Button size="small" title="Удалить поручение" icon={<DeleteOutlined />} danger />
                            </Popconfirm>}
                        </>
                    )}
                </Space>
            );
        },
    },
];
