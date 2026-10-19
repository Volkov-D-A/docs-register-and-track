import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Modal, Pagination, Spin } from 'antd';
import { dto, models } from '../../wailsjs/go/models';
import { GetCurrentUserEvents, MarkAllRead } from '../../wailsjs/go/services/UserEventService';
import { onServerEvent } from '../events/serverEvents';
import { useUserEventsStore } from '../store/useUserEventsStore';
import { formatAppError } from '../utils/appError';
import { CoalescedRequest } from '../utils/coalescedRequest';
import WorkspaceEventTimeline from './WorkspaceEventTimeline';

type Props = { onClose: () => void; onOpenDocument: (id: string, kind: string) => void };
export default function UserEventsModal({ onClose, onOpenDocument }: Props) {
    const [page, setPage] = useState(1);
    const [events, setEvents] = useState<dto.UserEvent[]>([]);
    const [total, setTotal] = useState(0);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(true);
    const marked = useRef(false);
    const request = useRef(new CoalescedRequest<Awaited<ReturnType<typeof GetCurrentUserEvents>>>());
    const load = useCallback(async () => {
        setLoading(true);
        await request.current.refresh(async () => {
            if (!marked.current) {
                await MarkAllRead();
                marked.current = true;
                await useUserEventsStore.getState().refresh();
            }
            return GetCurrentUserEvents(models.UserEventFilter.createFrom({ page, pageSize: 20 }));
        }, {
            onSuccess: (result) => { setEvents(result.items || []); setTotal(result.totalCount); setError(''); },
            onError: (error) => { setEvents([]); setError(formatAppError(error, 'Не удалось загрузить события')); },
            onSettled: () => setLoading(false),
        });
    }, [page]);
    useEffect(() => { const current = request.current; void load(); return () => current.invalidate(); }, [load]);
    useEffect(() => onServerEvent((event) => {
        if (['user-events', 'resync', 'access-changed'].includes(event.topic)) void load();
    }), [load]);
    return <Modal open title="Все события" onCancel={onClose} footer={null} width={760}>
        {error ? <Alert type="error" title={error} action={<Button onClick={() => void load()}>Повторить</Button>} /> : <Spin spinning={loading}>
            <WorkspaceEventTimeline events={events} currentTime={new Date()} onOpenDocument={(id, kind) => { onClose(); onOpenDocument(id, kind); }} />
            <Pagination style={{ marginTop: 16 }} current={page} pageSize={20} total={total} showSizeChanger={false} onChange={setPage} />
        </Spin>}
    </Modal>;
}
