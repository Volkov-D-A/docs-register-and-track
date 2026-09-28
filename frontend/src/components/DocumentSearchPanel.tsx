import React, { useEffect, useRef, useState } from 'react';
import { Alert, Button, Input, Modal, Pagination, Table, Tag, Typography } from 'antd';
import type { TableProps } from 'antd';
import dayjs from 'dayjs';
import { dto } from '../../wailsjs/go/models';
import { Search } from '../../wailsjs/go/services/DocumentQueryService';
import { getDocumentKindColor, getDocumentKindShortLabel } from '../constants/documentKinds';
import { onServerEvent } from '../events/serverEvents';
import { formatAppError } from '../utils/appError';

const pageSize = 20;
const compactText = (value: string) => <Typography.Paragraph ellipsis={{ rows: 3, expandable: true, symbol: 'ещё' }} style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{value || '—'}</Typography.Paragraph>;

type Props = { onOpenDocument: (id: string, kind: string) => void };

const DocumentSearchPanel: React.FC<Props> = ({ onOpenDocument }) => {
    const [input, setInput] = useState('');
    const [query, setQuery] = useState('');
    const [open, setOpen] = useState(false);
    const [page, setPage] = useState(1);
    const [result, setResult] = useState<dto.DocumentSearchResult | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const version = useRef(0);

    const load = async (text: string, nextPage: number) => {
        const current = ++version.current;
        setLoading(true);
        setError('');
        setResult(null);
        try {
            const value = await Search(dto.DocumentSearchRequest.createFrom({ query: text, page: nextPage, pageSize }));
            if (current === version.current) setResult(value);
        } catch (err) {
            if (current === version.current) setError(formatAppError(err, 'Не удалось выполнить поиск документов'));
        } finally {
            if (current === version.current) setLoading(false);
        }
    };

    const close = () => {
        version.current += 1;
        setOpen(false);
        setResult(null);
        setLoading(false);
        setError('');
    };

    useEffect(() => () => { version.current += 1; }, []);
    // A changed read scope must discard already displayed rows as well as late replies.
    useEffect(() => onServerEvent((event) => {
        if (event.topic === 'access-changed' || event.visibilityChanged) {
            version.current += 1;
            setOpen(false);
            setResult(null);
            setLoading(false);
            setError('');
        }
    }), []);

    const columns: TableProps<dto.DocumentSearchItem>['columns'] = [
        {
            title: 'Документ', key: 'document', width: 190,
            render: (_value, item) => <div>
                <Tag color={getDocumentKindColor(item.kindCode)}>{getDocumentKindShortLabel(item.kindCode)}</Tag>
                <div><Button type="link" style={{ padding: 0 }} onClick={() => onOpenDocument(item.id, item.kindCode)}>
                    {item.registrationNumber ? `№ ${item.registrationNumber}` : 'Без номера'}
                </Button></div>
                <Typography.Text type="secondary">{dayjs(item.registrationDate).format('DD.MM.YYYY')}</Typography.Text>
            </div>,
        },
        { title: 'Содержание / Заголовок', dataIndex: 'content', key: 'content', width: 300, render: compactText },
        { title: 'Резолюция', dataIndex: 'resolution', key: 'resolution', width: 230, render: compactText },
        { title: 'Корреспондент / Получатель', dataIndex: 'correspondent', key: 'correspondent', width: 200, render: compactText },
        { title: 'Подписант / Адресат', dataIndex: 'person', key: 'person', width: 180, render: compactText },
    ];

    return <>
        <Input.Search aria-label="Поиск по документам" placeholder="Поиск по документам: содержание, резолюция, корреспондент, подписант"
            value={input} onChange={(event) => setInput(event.target.value)} allowClear maxLength={500}
            enterButton="Найти" style={{ marginBottom: 16 }} onSearch={(value) => {
                const text = value.trim().replace(/\s+/g, ' ');
                if (!text) return;
                setQuery(text);
                setPage(1);
                setOpen(true);
                void load(text, 1);
            }} />
        <Modal title="Поиск документов" open={open} onCancel={close} footer={null} width={1150}>
            <Typography.Paragraph>Результаты для «{query}» · по релевантности</Typography.Paragraph>
            {error ? <Alert type="error" title={error} action={<Button onClick={() => { void load(query, page); }}>Повторить</Button>} /> : <>
                <Table<dto.DocumentSearchItem> columns={columns} dataSource={result?.items || []} rowKey="id"
                    loading={loading} size="small" pagination={false} scroll={{ x: 1100, y: 480 }}
                    locale={{ emptyText: loading ? 'Выполняется поиск…' : 'Документы не найдены' }} />
                <Pagination current={page} total={result?.totalCount || 0} pageSize={pageSize} showSizeChanger={false}
                    disabled={loading} showTotal={(total) => `Найдено: ${total}`} style={{ marginTop: 16 }}
                    onChange={(value) => { setPage(value); void load(query, value); }} />
            </>}
        </Modal>
    </>;
};

export default DocumentSearchPanel;
