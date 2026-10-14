import { useDocumentRefresh } from '../hooks/useDocumentRefresh';
import React, { useCallback, useEffect, useState } from 'react';
import { Table, Spin, App, Tooltip, Typography } from 'antd';
import {
    PlusCircleOutlined, EditOutlined, DeleteOutlined,
    SyncOutlined, CheckCircleOutlined, ProfileOutlined,
    LinkOutlined,
    QuestionCircleOutlined
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { formatAppError } from '../utils/appError';

const { Text } = Typography;

interface JournalEntry {
    id: string;
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

    const loadJournal = useCallback(async () => {
        setLoading(true);
        try {
            const { GetByDocumentID } = await import('../../wailsjs/go/services/JournalService');
            const res = await GetByDocumentID(documentId);
            setEntries(res || []);
        } catch (err: unknown) {
            message.error(formatAppError(err, 'Ошибка загрузки журнала'));
        } finally {
            setLoading(false);
        }
    }, [documentId, message]);

    useEffect(() => {
        if (documentId) {
            loadJournal();
        }
    }, [documentId, loadJournal]);

    useDocumentRefresh(documentId, 'journal', loadJournal);

    const actionConfig: Record<string, { color: string; icon: React.ReactNode; tooltip: string }> = {
        'CREATE': { color: 'green', icon: <PlusCircleOutlined />, tooltip: 'Документ создан' },
        'UPDATE': { color: 'blue', icon: <EditOutlined />, tooltip: 'Документ обновлён' },
        'DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Документ удалён' },
        'STATUS_CHANGE': { color: 'orange', icon: <SyncOutlined />, tooltip: 'Статус изменён' },
        'DEADLINE_CONTROL_CREATE': { color: 'cyan', icon: <ProfileOutlined />, tooltip: 'Контроль установлен' },
        'DEADLINE_CONTROL_UPDATE': { color: 'blue', icon: <EditOutlined />, tooltip: 'Контроль изменён' },
        'DEADLINE_CONTROL_COMPLETE': { color: 'green', icon: <CheckCircleOutlined />, tooltip: 'Контроль окончательно исполнен' },
        'DEADLINE_CONTROL_CANCEL': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Контроль отменён' },
        'DEADLINE_SERIES_CREATE': { color: 'cyan', icon: <ProfileOutlined />, tooltip: 'Серия создана' },
        'DEADLINE_SERIES_UPDATE': { color: 'blue', icon: <EditOutlined />, tooltip: 'Шаблон серии изменён' },
        'DEADLINE_SERIES_STOP': { color: 'orange', icon: <SyncOutlined />, tooltip: 'Серия остановлена' },
        'DEADLINE_SERIES_AUTO_STOP': { color: 'orange', icon: <SyncOutlined />, tooltip: 'Серия остановлена автоматически' },
        'DEADLINE_SERIES_RESUME': { color: 'green', icon: <SyncOutlined />, tooltip: 'Серия возобновлена' },
        'LINK_CREATE': { color: 'geekblue', icon: <LinkOutlined />, tooltip: 'Добавлена связь' },
        'LINK_DELETE': { color: 'red', icon: <DeleteOutlined />, tooltip: 'Связь удалена' },
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
                const config = actionConfig[record.action] || { color: 'var(--app-text-muted)', icon: <QuestionCircleOutlined />, tooltip: record.action };
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
