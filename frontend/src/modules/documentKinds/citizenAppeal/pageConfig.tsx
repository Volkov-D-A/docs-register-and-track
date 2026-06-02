import React from 'react';
import { Button, Space, Tag, Tooltip } from 'antd';
import { EyeOutlined, EditOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DOCUMENT_KIND_CITIZEN_APPEAL } from '../../../constants/documentKinds';

type ColumnFactoryParams = {
    isExecutorOnly: boolean;
    openViewModal: (documentId: string) => void;
    onEdit: (record: any) => void;
};

export const citizenAppealPageConfig = {
    kindCode: DOCUMENT_KIND_CITIZEN_APPEAL,
    title: 'Обращения граждан',
    tableClassName: 'citizen-appeals-table',
    registerModalTitle: 'Регистрация обращения граждан',
    getEditModalTitle: (record: any) => `Редактирование обращения: ${record?.registrationNumber || ''}`,
    registerInitialValues: {
        registrationDate: dayjs(),
        appealDate: dayjs(),
        appealType: 'заявление',
        appealPagesCount: 1,
        attachmentPagesCount: 0,
        hasEnvelope: false,
        receivedFromPos: false,
        correspondents: [],
        resolutions: [{}],
    },
    buildColumns: ({ isExecutorOnly, openViewModal, onEdit }: ColumnFactoryParams) => [
        {
            title: 'Номер / Регистрация',
            key: 'number',
            width: 160,
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.registrationNumber}</div>
                    <div style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>
                        от {dayjs(record.registrationDate).format('DD.MM.YYYY')}
                    </div>
                </div>
            ),
        },
        {
            title: 'Обращение',
            key: 'appeal',
            width: '18%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontSize: 13, color: 'var(--app-text-muted)' }}>
                        Дата: {dayjs(record.appealDate).format('DD.MM.YYYY')}
                    </div>
                    <div style={{ fontSize: 13, color: 'var(--app-text-muted)' }}>
                        Листов: {record.appealPagesCount || 0} + {record.attachmentPagesCount || 0}
                    </div>
                </div>
            ),
        },
        {
            title: 'Обратившийся',
            key: 'applicant',
            width: '24%',
            render: (_: any, record: any) => (
                <div>
                    <div style={{ fontWeight: 600 }}>{record.applicantFullName}</div>
                    <div style={{ fontSize: 12, color: 'var(--app-text-muted)' }}>{record.registrationAddress}</div>
                </div>
            ),
        },
        {
            title: 'Содержание',
            key: 'content',
            width: '28%',
            render: (_: any, record: any) => (
                <div style={{ fontWeight: 500, whiteSpace: 'pre-wrap' }}>{record.content}</div>
            ),
        },
        {
            title: 'Резолюция',
            key: 'resolution',
            width: '18%',
            render: (_: any, record: any) => {
                const count = record.resolutions?.length || 0;
                return (
                    <div style={{ fontSize: 13 }}>
                        {record.resolution ? (
                            <>
                                <div style={{ fontStyle: 'italic', color: 'var(--app-text-muted)' }}>{record.resolution}</div>
                                {count > 1 && <Tag style={{ marginTop: 4 }}>Еще: {count - 1}</Tag>}
                            </>
                        ) : (
                            <span style={{ color: 'var(--app-text-muted)' }}>—</span>
                        )}
                    </div>
                );
            },
        },
        {
            title: 'Действия',
            key: 'actions',
            width: 120,
            render: (_: any, record: any) => (
                <Space>
                    <Tooltip title="Просмотреть карточку документа">
                        <Button size="small" icon={<EyeOutlined />} onClick={() => openViewModal(record.id)} />
                    </Tooltip>
                    {!isExecutorOnly && (
                        <Tooltip title="Редактировать документ">
                            <Button size="small" icon={<EditOutlined />} onClick={() => onEdit(record)} />
                        </Tooltip>
                    )}
                </Space>
            ),
        },
    ],
};
