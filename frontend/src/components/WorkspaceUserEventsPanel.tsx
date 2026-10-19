import React, { useEffect, useState } from 'react';
import { Alert, Badge, Button, Card } from 'antd';
import { useUserEventsStore } from '../store/useUserEventsStore';
import WorkspaceEventTimeline from './WorkspaceEventTimeline';
import UserEventsModal from './UserEventsModal';

type Props = {
    currentTime: Date;
    refreshVersion: number;
    onOpenDocument: (id: string, kind: string) => void;
};

const WorkspaceUserEventsPanel = ({ currentTime, refreshVersion, onOpenDocument }: Props) => {
    const { events, unreadCount, error, refresh } = useUserEventsStore();
    const [open, setOpen] = useState(false);
    const preview = events.slice(0, 5);
    const hiddenUnread = Math.max(0, unreadCount - preview.filter((event) => !event.readAt).length);
    useEffect(() => { if (refreshVersion > 0) void refresh(); }, [refresh, refreshVersion]);
    return <Card title="Новое для меня" className="workspace-events-panel" extra={
        <Badge count={hiddenUnread} size="small" title={`Непрочитанных вне списка: ${hiddenUnread}`}>
            <Button type="link" size="small" onClick={() => setOpen(true)}>Все события</Button>
        </Badge>
    }>
        {error ? <Alert type="error" title={error} action={<Button onClick={() => void refresh()}>Повторить</Button>} /> : <WorkspaceEventTimeline events={preview} currentTime={currentTime} onOpenDocument={onOpenDocument} />}
        {open && <UserEventsModal onClose={() => setOpen(false)} onOpenDocument={onOpenDocument} />}
    </Card>;
};

export default React.memo(WorkspaceUserEventsPanel);
