import { EventsOn } from '../../wailsjs/runtime/runtime';
import { models } from '../../wailsjs/go/models';
import { useAuthStore } from '../store/useAuthStore';
import { invalidateCurrentAccessSummary } from '../store/accessSummaryCache';

export type DocumentResource = 'document' | 'assignments' | 'files' | 'links';

export type ServerEvent = {
    topic: string;
    revision: number;
    documentId?: string;
    documentKind?: string;
    resource?: DocumentResource;
    visibilityChanged?: boolean;
    operation?: models.BackupOperation;
};

const listeners = new Set<(event: ServerEvent) => void>();
let stopTransport: (() => void) | undefined;

export const onServerEvent = (listener: (event: ServerEvent) => void) => {
    listeners.add(listener);
    if (!stopTransport) {
        stopTransport = EventsOn('server:event', (event: ServerEvent) => {
            // Restore capabilities deliberately survive invalidation of the login session.
            if (event.topic !== 'operation') {
                const session = useAuthStore.getState();
                if (!session.isAuthenticated || event.revision !== session.sessionRevision) return;
            }
            if (event.topic === 'resync' || event.topic === 'access-changed') invalidateCurrentAccessSummary();
            for (const subscriber of [...listeners]) subscriber(event);
        });
    }
    return () => {
        listeners.delete(listener);
        if (listeners.size === 0) {
            stopTransport?.();
            stopTransport = undefined;
        }
    };
};
