import React from 'react';
import { Button, Space, Tag } from 'antd';
import { EyeOutlined, EditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_ADMINISTRATIVE_ORDER } from '../../../constants/documentKinds';

type ColumnFactoryParams = {
    isExecutorOnly: boolean;
    openViewModal: (documentId: string) => void;
    onEdit: (record: any) => void;
};

export const administrativeOrderPageConfig = {
    kindCode: DOCUMENT_KIND_ADMINISTRATIVE_ORDER,
    title: 'Приказы',
    tableClassName: 'administrative-orders-table',
    registerModalTitle: 'Регистрация приказа',
    getEditModalTitle: () => 'Редактирование приказа',
    registerInitialValues: { orderDate: dayjs(), isActive: true, acknowledgmentFullNames: [] },
    buildColumns: ({ isExecutorOnly, openViewModal, onEdit }: ColumnFactoryParams) => [
        {
            title: 'Номер / Дата',
            key: 'number',
            width: 150,
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.orderNumber}</div>
                    <div style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
                        от {dayjs(record.orderDate).format('DD.MM.YYYY')}
                    </div>
                </div>
            ),
        },
        {
            title: 'Заголовок',
            key: 'title',
            width: '34%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 500, whiteSpace: 'pre-wrap' }}>{record.title || record.content}</div>
                    <div style={{ marginTop: 4 }}>
                        <Tag color={record.isActive === false ? 'default' : 'green'}>
                            {record.isActive === false ? 'Не действующий' : 'Действующий'}
                        </Tag>
                        {record.pendingAcknowledgmentsCount > 0 && (
                            <Tag color="gold">Внешнее ознакомление: {record.pendingAcknowledgmentsCount}</Tag>
                        )}
                    </div>
                </div>
            ),
        },
        {
            title: 'Контроль',
            key: 'control',
            width: '24%',
            render: (_: any, record: any) => (
                <div style={{ fontSize: 13 }}>
                    <div>{record.executionController || '—'}</div>
                    {record.executionDeadline && (
                        <div style={{ color: 'var(--app-text-muted)' }}>
                            срок: {dayjs(record.executionDeadline).format('DD.MM.YYYY')}
                        </div>
                    )}
                </div>
            ),
        },
        {
            title: 'Дело',
            dataIndex: 'nomenclatureName',
            key: 'nomenclature',
            width: '22%',
            render: (value: string) => <span style={{ fontSize: 13 }}>{value}</span>,
        },
        {
            title: 'Действия',
            key: 'actions',
            width: 120,
            render: (_: any, record: any) => (
                <Space>
                    <Button size="small" icon={<EyeOutlined />} onClick={() => openViewModal(record.id)} />
                    {!isExecutorOnly && (
                        <Button size="small" icon={<EditOutlined />} onClick={() => onEdit(record)} />
                    )}
                </Space>
            ),
        },
    ],
};
