import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Card, Empty, Spin } from 'antd';
import { FileTextOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { dto } from '../../wailsjs/go/models';
import { GetRecentDocuments } from '../../wailsjs/go/services/WorkspaceService';
import { onServerEvent } from '../events/serverEvents';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { formatAppError, normalizeAppError } from '../utils/appError';

type RecentDocumentsPanelProps = {
    refreshVersion: number;
    onOpenDocument: (id: string, kind: string) => void;
};

const kinds: Record<string, { label: string; tone: string }> = {
    incoming_letter: { label: 'Вх.', tone: 'blue' },
    outgoing_letter: { label: 'Исх.', tone: 'blue' },
    administrative_order: { label: 'Приказ', tone: 'green' },
    citizen_appeal: { label: 'Обращение', tone: 'purple' },
};

const RecentDocumentsPanel: React.FC<RecentDocumentsPanelProps> = ({ refreshVersion, onOpenDocument }) => {
    const [result, setResult] = useState<dto.WorkspaceDocuments | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const request = useRef(new CoalescedRequest<dto.WorkspaceDocuments>());
    const load = useCallback(() => {
        setLoading(true);
        return request.current.refresh(() => GetRecentDocuments(), {
            onSuccess: (value) => { setResult(value); setError(''); },
            onError: (err) => {
                setResult(null);
                if (normalizeAppError(err).code === 'FORBIDDEN') {
                    setResult(dto.WorkspaceDocuments.createFrom({ available: false, items: [] }));
                    setError('');
                } else {
                    setError(formatAppError(err, 'Не удалось загрузить последние документы'));
                }
            },
            onSettled: () => setLoading(false),
        });
    }, []);

    useEffect(() => { void load(); }, [load, refreshVersion]);
    useEffect(() => onServerEvent((event) => {
        if (event.topic === 'documents' || event.topic === 'resync' || event.topic === 'user-events') void load();
    }), [load]);
    useEffect(() => () => request.current.invalidate(), []);

    if (result?.available === false) return null;
    return (
        <Card title="Последние документы" className="workspace-recent-documents">
            {error ? <Alert type="error" title={error} action={<Button size="small" onClick={() => { void load(); }}>Повторить</Button>} /> : (
                <Spin spinning={loading}>
                    {!result ? <div className="workspace-recent-placeholder" /> : result.items.length === 0 ? (
                        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Доступных документов пока нет" />
                    ) : (
                        <ul className="workspace-document-list" aria-label="Последние зарегистрированные документы">
                            {result.items.map((document) => {
                                const kind = kinds[document.documentKind] || { label: 'Документ', tone: 'blue' };
                                const title = `${kind.label} ${document.documentNumber ? `№ ${document.documentNumber}` : 'без номера'}${document.documentDate ? ` от ${dayjs(document.documentDate).format('DD.MM.YYYY')}` : ''}`;
                                const correspondents = document.correspondents || [];
                                const description = correspondents.length > 0
                                    ? `${correspondents[0]}${correspondents.length > 1 ? ` · ещё ${correspondents.length - 1}` : ''}`
                                    : document.description || '—';
                                return (
                                    <li key={document.id}>
                                        <button type="button" className={`workspace-document-row workspace-document-row--${kind.tone}`}
                                            onClick={() => onOpenDocument(document.id, document.documentKind)}>
                                            <span className="workspace-document-icon" aria-hidden="true"><FileTextOutlined /></span>
                                            <span className="workspace-document-content">
                                                <span className="workspace-document-title" title={title}>{title}</span>
                                                <span className="workspace-document-description" title={correspondents.length ? correspondents.join('; ') : description}>{description}</span>
                                            </span>
                                            <time className="workspace-document-registered" dateTime={document.registeredAt}
                                                title={`Зарегистрирован ${dayjs(document.registeredAt).format('DD.MM.YYYY HH:mm')}`}>
                                                {dayjs(document.registeredAt).format('DD.MM.YYYY')}
                                            </time>
                                        </button>
                                    </li>
                                );
                            })}
                        </ul>
                    )}
                </Spin>
            )}
        </Card>
    );
};

export default RecentDocumentsPanel;
