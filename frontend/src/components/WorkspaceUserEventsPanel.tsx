import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Card } from 'antd';
import { dto, models } from '../../wailsjs/go/models';
import { GetCurrentUserEvents } from '../../wailsjs/go/services/UserEventService';
import { onServerEvent } from '../events/serverEvents';
import { CoalescedRequest } from '../utils/coalescedRequest';
import WorkspaceEventTimeline from './WorkspaceEventTimeline';

type Props = {
    currentTime: Date;
    refreshVersion: number;
    onOpenDocument: (id: string, kind: string) => void;
};

const WorkspaceUserEventsPanel = ({ currentTime, refreshVersion, onOpenDocument }: Props) => {
    const [events, setEvents] = useState<dto.UserEvent[]>([]);
    const request = useRef(new CoalescedRequest<dto.PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_UserEvent_>());
    const load = useCallback(() => request.current.refresh(
        () => GetCurrentUserEvents(models.UserEventFilter.createFrom({ page: 1, pageSize: 5 })), {
            onSuccess: (result) => setEvents(result?.items || []),
            onError: (error) => console.error('Workspace events:', error),
        },
    ), []);
    useEffect(() => { void load(); }, [load, refreshVersion]);
    useEffect(() => onServerEvent((event) => {
        if (event.topic === 'user-events' || event.topic === 'resync' || event.topic === 'access-changed') void load();
    }), [load]);
    useEffect(() => () => request.current.invalidate(), []);
    return <Card title="Новое для меня" className="workspace-events-panel">
        <WorkspaceEventTimeline events={events} currentTime={currentTime} onOpenDocument={onOpenDocument} />
    </Card>;
};

export default React.memo(WorkspaceUserEventsPanel);
