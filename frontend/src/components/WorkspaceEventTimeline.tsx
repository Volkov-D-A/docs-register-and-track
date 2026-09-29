import React from 'react';
import { Empty } from 'antd';
import { CheckCircleFilled, FileTextOutlined, MessageOutlined, UserOutlined, WarningOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { dto } from '../../wailsjs/go/models';
import { getDocumentKindShortLabel } from '../constants/documentKinds';

type WorkspaceEventTimelineProps = {
    events: dto.UserEvent[];
    currentTime: Date;
    onOpenDocument: (id: string, kind: string) => void;
};

const eventAppearance = (event: dto.UserEvent): { tone: string; icon: React.ReactNode } => {
    switch (event.eventType) {
        case 'assignment_created': return { tone: 'red', icon: <UserOutlined /> };
        case 'assignment_finished':
        case 'assignment_acknowledged': return { tone: 'green', icon: <CheckCircleFilled /> };
        case 'assignment_returned': return { tone: 'red', icon: <WarningOutlined /> };
        default: return event.entityType === 'assignment'
            ? { tone: 'slate', icon: <MessageOutlined /> }
            : { tone: 'blue', icon: <FileTextOutlined /> };
    }
};

const WorkspaceEventTimeline: React.FC<WorkspaceEventTimelineProps> = ({ events, currentTime, onOpenDocument }) => {
    if (events.length === 0) {
        return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Новых событий нет" />;
    }
    const now = dayjs(currentTime);
    return (
        <ul className="workspace-event-timeline" aria-label="Персональные события">
            {events.map((event) => {
                const { tone, icon } = eventAppearance(event);
                const created = dayjs(event.createdAt);
                const time = !created.isValid() ? '—' : created.isSame(now, 'day') ? created.format('HH:mm')
                    : created.isSame(now.subtract(1, 'day'), 'day') ? 'Вчера' : created.format('DD.MM.YYYY');
                const documentLabel = `${getDocumentKindShortLabel(event.documentKind)}${event.documentNumber ? ` № ${event.documentNumber}` : ' без номера'}${event.documentDate ? ` от ${dayjs(event.documentDate).format('DD.MM.YYYY')}` : ''}`;
                return (
                    <li key={event.id} className={`workspace-event workspace-event--${tone}${event.readAt ? ' workspace-event--read' : ''}`}>
                        <button type="button" className="workspace-event-action" onClick={() => onOpenDocument(event.documentId, event.documentKind)}>
                            <span className="workspace-event-dot" aria-hidden="true" />
                            <span className="workspace-event-icon" aria-hidden="true">{icon}</span>
                            <time className="workspace-event-time" dateTime={created.isValid() ? created.toISOString() : undefined}
                                title={created.isValid() ? created.format('DD.MM.YYYY HH:mm') : undefined}>{time}</time>
                            <span className="workspace-event-content">
                                <span className="workspace-event-title" title={event.title}>{event.message || event.title}</span>
                                <span className="workspace-event-document">{documentLabel}</span>
                            </span>
                        </button>
                    </li>
                );
            })}
        </ul>
    );
};

export default WorkspaceEventTimeline;
