import { dto } from '../../wailsjs/go/models';

export const USER_EVENTS_RECEIVED_EVENT = 'user-events:received';
export const USER_EVENTS_DOCUMENT_READ_EVENT = 'user-events:document-read';

export type UserEventsReceivedDetail = {
    events: dto.UserEvent[];
};

export type UserEventsDocumentReadDetail = {
    documentId: string;
};

export const emitUserEventsReceived = (events: dto.UserEvent[]) => {
    if (events.length === 0) {
        return;
    }

    window.dispatchEvent(new CustomEvent<UserEventsReceivedDetail>(USER_EVENTS_RECEIVED_EVENT, {
        detail: { events },
    }));
};

export const onUserEventsReceived = (
    listener: (events: dto.UserEvent[]) => void,
) => {
    const handler = (event: Event) => {
        listener((event as CustomEvent<UserEventsReceivedDetail>).detail.events);
    };

    window.addEventListener(USER_EVENTS_RECEIVED_EVENT, handler);
    return () => window.removeEventListener(USER_EVENTS_RECEIVED_EVENT, handler);
};

export const emitUserEventsDocumentRead = (documentId: string) => {
    if (!documentId) {
        return;
    }

    window.dispatchEvent(new CustomEvent<UserEventsDocumentReadDetail>(USER_EVENTS_DOCUMENT_READ_EVENT, {
        detail: { documentId },
    }));
};

export const onUserEventsDocumentRead = (
    listener: (documentId: string) => void,
) => {
    const handler = (event: Event) => {
        listener((event as CustomEvent<UserEventsDocumentReadDetail>).detail.documentId);
    };

    window.addEventListener(USER_EVENTS_DOCUMENT_READ_EVENT, handler);
    return () => window.removeEventListener(USER_EVENTS_DOCUMENT_READ_EVENT, handler);
};

export const isAssignmentUserEvent = (event: dto.UserEvent) => (
    event.entityType === 'assignment' || event.eventType.startsWith('assignment_')
);

export const isAcknowledgmentUserEvent = (event: dto.UserEvent) => (
    event.entityType === 'acknowledgment' || event.eventType.startsWith('acknowledgment_')
);
