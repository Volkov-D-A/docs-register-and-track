import React, { useEffect, useState } from 'react';
import { Table, Spin, App, Tag, Typography } from 'antd';
import dayjs from 'dayjs';

const { Text } = Typography;

interface JournalEntry {
    id: string;
    documentId: string;
    documentType: string;
    userName?: string;
    action: string;
    details: string;
    createdAt: string;
}

interface JournalListProps {
    documentId: string;
    documentType: 'incoming' | 'outgoing';
}

const JournalList: React.FC<JournalListProps> = ({ documentId, documentType }) => {
    const { message } = App.useApp();
    const [entries, setEntries] = useState<JournalEntry[]>([]);
    const [loading, setLoading] = useState<boolean>(true);

    useEffect(() => {
        if (documentId) {
            loadJournal();
        }
    }, [documentId, documentType]);

    const loadJournal = async () => {
        setLoading(true);
        try {
            // @ts-ignore
            const { GetByDocumentID } = await import('../../wailsjs/go/services/JournalService');
            const res = await GetByDocumentID(documentId, documentType);
            setEntries(res || []);
        } catch (err: any) {
            message.error('Ошибка загрузки журнала: ' + (err.message || String(err)));
        } finally {
            setLoading(false);
        }
    };

    const actionColors: Record<string, string> = {
        'CREATE': 'green',
        'UPDATE': 'blue',
        'DELETE': 'red',
        'STATUS_CHANGE': 'orange',
        'ASSIGNMENT_CREATE': 'cyan',
        'ASSIGNMENT_UPDATE': 'blue',
        'ASSIGNMENT_STATUS': 'orange',
        'ASSIGNMENT_DELETE': 'red',
        'FILE_UPLOAD': 'purple',
        'FILE_DELETE': 'red',
        'LINK_CREATE': 'geekblue',
        'LINK_DELETE': 'red',
        'ACK_CREATE': 'cyan',
        'ACK_VIEW': 'blue',
        'ACK_CONFIRM': 'green',
        'ACK_DELETE': 'red',
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
            title: 'Действие',
            dataIndex: 'action',
            key: 'action',
            render: (text: string) => {
                const color = actionColors[text] || 'default';
                return <Tag color={color}>{text}</Tag>;
            },
            width: 180,
        },
        {
            title: 'Детали',
            dataIndex: 'details',
            key: 'details',
            render: (text: string) => <Text>{text}</Text>,
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
            pagination={{ pageSize: 15 }}
            size="small"
        />
    );
};

export default JournalList;
