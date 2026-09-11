import { EventsOn } from '../../wailsjs/runtime/runtime';
import { models } from '../../wailsjs/go/models';
import { useAuthStore } from '../store/useAuthStore';

type ServerEvent = {
    topic: string;
    revision: number;
    operation?: models.BackupOperation;
};

export const onServerEvent = (listener: (event: ServerEvent) => void) => EventsOn('server:event', (event: ServerEvent) => {
    // Restore capabilities deliberately survive invalidation of the login session.
    if (event.topic !== 'operation') {
        const session = useAuthStore.getState();
        if (!session.isAuthenticated || event.revision !== session.sessionRevision) return;
    }
    listener(event);
});
