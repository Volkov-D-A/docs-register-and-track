import React from 'react';
import { Button, Space } from 'antd';
import { EyeOutlined, EditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_OUTGOING_LETTER } from '../../../constants/documentKinds';

type ColumnFactoryParams = {
    isExecutorOnly: boolean;
    openViewModal: (documentId: string) => void;
    onEdit: (record: any) => void;
};

export const outgoingLetterPageConfig = {
    kindCode: DOCUMENT_KIND_OUTGOING_LETTER,
    title: 'Исходящие документы',
    tableClassName: 'outgoing-documents-table',
    registerModalTitle: 'Регистрация исходящего документа',
    getEditModalTitle: () => 'Редактирование документа',
    registerInitialValues: { outgoingDate: dayjs(), pagesCount: 1 },
    buildColumns: ({ isExecutorOnly, openViewModal, onEdit }: ColumnFactoryParams) => [
        {
            title: 'Номер / Дата',
            key: 'number',
            width: 140,
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.outgoingNumber}</div>
                    <div style={{ fontSize: 12, color: '#888' }}>
                        от {dayjs(record.outgoingDate).format('DD.MM.YYYY')}
                    </div>
                </div>
            ),
        },
        {
            title: 'Получатель',
            key: 'recipient',
            width: '26%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.recipientOrgName}</div>
                    <div style={{ fontSize: 13 }}>Адресат: {record.addressee}</div>
                </div>
            ),
        },
        {
            title: 'Содержание',
            key: 'content',
            width: '26%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 500, whiteSpace: 'pre-wrap' }}>{record.content}</div>
                    <div style={{ fontSize: 13, color: '#666' }}>{record.documentTypeName}</div>
                </div>
            ),
        },
        {
            title: 'Исполнитель / Подписант',
            key: 'executor',
            width: '26%',
            render: (_: any, record: any) => (
                <div style={{ fontSize: 13 }}>
                    <div>Исп: {record.senderExecutor}</div>
                    <div>Подп: {record.senderSignatory}</div>
                </div>
            ),
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
