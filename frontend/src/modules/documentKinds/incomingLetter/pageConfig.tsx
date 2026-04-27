import React from 'react';
import { Button, Space } from 'antd';
import { EyeOutlined, EditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_INCOMING_LETTER } from '../../../constants/documentKinds';
import { DEFAULT_DOCUMENT_TYPE } from '../../../constants/documentTypes';

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
    registerInitialValues: { incomingDate: dayjs(), pagesCount: 1, documentTypeId: DEFAULT_DOCUMENT_TYPE, correspondents: [{}] },
    buildColumns: ({ isExecutorOnly, openViewModal, onEdit }: ColumnFactoryParams) => [
        {
            title: 'Номер / Дата',
            key: 'number',
            width: 140,
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.incomingNumber}</div>
                    <div style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
                        от {dayjs(record.incomingDate).format('DD.MM.YYYY')}
                    </div>
                </div>
            ),
        },
        {
            title: 'Корреспондент',
            key: 'correspondent',
            width: '26%',
            render: (_: any, record: any) => {
                const correspondents = record.correspondents || [];
                if (correspondents.length === 0) {
                    return <span style={{ color: 'var(--app-text-muted)' }}>—</span>;
                }
                return (
                    <div>
                        {correspondents.slice(0, 2).map((item: any) => (
                            <div key={item.id || `${item.registrationNumber}-${item.correspondentName}`} style={{ marginBottom: 4 }}>
                                <div style={{ fontWeight: 600 }}>{item.correspondentName}</div>
                                <div style={{ fontSize: 13, color: 'var(--app-text-muted)' }}>
                                    № {item.registrationNumber} от {dayjs(item.registrationDate).format('DD.MM.YYYY')}
                                </div>
                            </div>
                        ))}
                        {correspondents.length > 2 && (
                            <div style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>Еще: {correspondents.length - 2}</div>
                        )}
                    </div>
                );
            },
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
                    {record.resolution && <div style={{ fontStyle: 'italic', color: 'var(--app-text-muted)' }}>{record.resolution}</div>}
                    {!record.resolution && <span style={{ color: 'var(--app-text-muted)' }}>—</span>}
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
