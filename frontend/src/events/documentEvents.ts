import type { ServerEvent } from './serverEvents';

type TopicEvent = Pick<ServerEvent, 'topic'>;

export const isDocumentResyncEvent = (event: TopicEvent): boolean => (
    event.topic === 'resync' || event.topic === 'access-changed'
);

export const isDocumentUpdateEvent = (event: TopicEvent): boolean => (
    isDocumentResyncEvent(event) || event.topic === 'document-changed'
);
