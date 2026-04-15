import React from 'react';
import { Button, Space } from 'antd';
import { EyeOutlined, EditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_INCOMING_LETTER } from '../../../constants/documentKinds';

type ColumnFactoryParams = {
    isExecutorOnly: boolean;
    openViewModal: (documentId: string) => void;
    onEdit: (record: any) => void;
};

export const incomingLetterPageConfig = {
    kindCode: DOCUMENT_KIND_INCOMING_LETTER,
    title: 'Входящие документы',
    tableClassName: 'incoming-documents-table',
    registerModalTitle: 'Регистрация входящего документа',
    getEditModalTitle: (record: any) => `Редактирование: ${record?.incomingNumber || ''}`,
    registerInitialValues: { incomingDate: dayjs(), pagesCount: 1 },
    buildColumns: ({ isExecutorOnly, openViewModal, onEdit }: ColumnFactoryParams) => [
        {
            title: 'Номер / Дата',
            key: 'number',
            width: 140,
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.incomingNumber}</div>
                    <div style={{ fontSize: 12, color: '#888' }}>
                        от {dayjs(record.incomingDate).format('DD.MM.YYYY')}
                    </div>
                </div>
            ),
        },
        {
            title: 'Отправитель',
            key: 'sender',
            width: '26%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.senderOrgName}</div>
                    {(record.outgoingNumberSender || record.outgoingDateSender) && (
                        <div style={{ fontSize: 13, color: '#666' }}>
                            Исх: {record.outgoingNumberSender} {record.outgoingDateSender ? `от ${dayjs(record.outgoingDateSender).format('DD.MM.YYYY')}` : ''}
                        </div>
                    )}
                </div>
            ),
        },
        {
            title: 'Содержание',
            key: 'content',
            width: '26%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 500 }}>{record.content}</div>
                </div>
            ),
        },
        {
            title: 'Резолюция',
            key: 'resolution',
            width: '26%',
            render: (_: any, record: any) => (
                <div style={{ fontSize: 13 }}>
                    {record.resolution && <div style={{ fontStyle: 'italic', color: '#555' }}>{record.resolution}</div>}
                    {!record.resolution && <span style={{ color: '#bbb' }}>—</span>}
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
