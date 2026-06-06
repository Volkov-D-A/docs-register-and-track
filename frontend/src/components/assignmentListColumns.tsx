import React from 'react';
import { Button, Popconfirm, Space, Tag } from 'antd';
import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

type BuildAssignmentColumnsParams = {
    canManageAssignments: boolean;
    onEdit: (assignment: any) => void;
    onDelete: (id: string) => void;
};

export const buildAssignmentColumns = ({
    canManageAssignments,
    onEdit,
    onDelete,
}: BuildAssignmentColumnsParams) => [
    {
        title: 'Дата',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 100,
        render: (value: string) => dayjs(value).format('DD.MM.YYYY'),
    },
    { title: 'Содержание', dataIndex: 'content', key: 'content' },
    {
        title: 'Ответственный исполнитель',
        key: 'executorName',
        width: 200,
        render: (_: any, record: any) => (
            <div>
                <div>{record.executorName}</div>
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
            const canEdit = canManageAssignments && record.status !== 'finished';

            return (
                <Space size={2}>
                    {canEdit && (
                        <>
                            <Button size="small" title="Редактировать поручение" icon={<EditOutlined />} onClick={() => onEdit(record)} />
                            <Popconfirm
                                title="Удалить поручение?"
                                description="Это действие нельзя отменить. Поручение исчезнет из документа и списка исполнителя."
                                okText="Удалить"
                                cancelText="Отмена"
                                okButtonProps={{ danger: true }}
                                onConfirm={() => onDelete(record.id)}
                            >
                                <Button size="small" title="Удалить поручение" icon={<DeleteOutlined />} danger />
                            </Popconfirm>
                        </>
                    )}
                </Space>
            );
        },
    },
];
