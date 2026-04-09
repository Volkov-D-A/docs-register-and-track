import React, { useEffect, useState } from 'react';
import { Table, Spin, App, Tooltip, Typography } from 'antd';
import {
    PlusCircleOutlined, EditOutlined, DeleteOutlined,
    SyncOutlined, CheckCircleOutlined, UploadOutlined,
    LinkOutlined, EyeOutlined, ProfileOutlined,
    QuestionCircleOutlined, FileAddOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';

const { Text } = Typography;

interface JournalEntry {
    id: string;
    documentId: string;
    userName?: string;
    action: string;
    details: string;
    createdAt: string;
}

interface JournalListProps {
    documentId: string;
}

const JournalList: React.FC<JournalListProps> = ({ documentId }) => {
    const { message } = App.useApp();
    const [entries, setEntries] = useState<JournalEntry[]>([]);
    const [loading, setLoading] = useState<boolean>(true);

    useEffect(() => {
        if (documentId) {
            loadJournal();
        }
    }, [documentId]);

    const loadJournal = async () => {
        setLoading(true);
        try {
            // @ts-ignore
            const { GetByDocumentID } = await import('../../wailsjs/go/services/JournalService');
            const res = await GetByDocumentID(documentId);
            setEntries(res || []);
        } catch (err: any) {
            message.error('Ошибка загрузки журнала: ' + (err.message || String(err)));
        } finally {
            setLoading(false);
        }
    };

    const actionConfig: Record<string, { color: string; icon: React.ReactNode; tooltip: string }> = {
        'CREATE': { color: 'green', icon: <PlusCircleOutlined />, tooltip: 'Документ создан' },
        'UPDATE': { color: 'blue', icon: <EditOutlined />, tooltip: 'Документ обновлен' },
        'DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Документ удален' },
        'STATUS_CHANGE': { color: 'orange', icon: <SyncOutlined />, tooltip: 'Статус изменен' },
        'ASSIGNMENT_CREATE': { color: 'cyan', icon: <ProfileOutlined />, tooltip: 'Поручение выдано' },
        'ASSIGNMENT_UPDATE': { color: 'blue', icon: <EditOutlined />, tooltip: 'Поручение обновлено' },
        'ASSIGNMENT_STATUS': { color: 'orange', icon: <SyncOutlined />, tooltip: 'Статус поручения' },
        'ASSIGNMENT_DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Поручение удалено' },
        'FILE_UPLOAD': { color: 'purple', icon: <UploadOutlined />, tooltip: 'Файл загружен' },
        'FILE_DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Файл удален' },
        'LINK_CREATE': { color: 'geekblue', icon: <LinkOutlined />, tooltip: 'Добавлена связь' },
        'LINK_DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Связь удалена' },
        'ACK_CREATE': { color: 'cyan', icon: <FileAddOutlined />, tooltip: 'Ознакомление создано' },
        'ACK_VIEW': { color: 'blue', icon: <EyeOutlined />, tooltip: 'Ознакомление просмотрено' },
        'ACK_CONFIRM': { color: 'green', icon: <CheckCircleOutlined />, tooltip: 'Ознакомление подтверждено' },
        'ACK_DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Ознакомление удалено' },
    };

    const columns = [
        {
            title: 'Дата и время',
            dataIndex: 'createdAt',
            key: 'createdAt',
            render: (text: string) => dayjs(text).format('DD.MM.YYYY HH:mm:ss'),
            width: 160,
        },
        {
            title: 'Пользователь',
            dataIndex: 'userName',
            key: 'userName',
            width: 200,
        },
        {
            title: 'Детали',
            key: 'details',
            render: (record: JournalEntry) => {
                const config = actionConfig[record.action] || { color: '#888', icon: <QuestionCircleOutlined />, tooltip: record.action };
                return (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Tooltip title={config.tooltip}>
                            <div style={{ color: config.color, fontSize: 16, display: 'flex', alignItems: 'center' }}>
                                {config.icon}
                            </div>
                        </Tooltip>
                        <Text>{record.details}</Text>
                    </div>
                );
            },
        },
    ];

    if (loading) {
        return <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>;
    }

    return (
        <Table
            dataSource={entries}
            columns={columns}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            size="small"
        />
    );
};

export default JournalList;
