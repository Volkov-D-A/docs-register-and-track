import { useEffect } from 'react';
import { models } from '../../wailsjs/go/models';
import { GetCurrentUserEvents, GetUnreadCount } from '../../wailsjs/go/services/UserEventService';
import { onServerEvent } from '../events/serverEvents';
import { emitUserEventsReceived, onUserEventsDocumentRead } from '../events/userEvents';
import { useAuthStore } from '../store/useAuthStore';
import { useUserEventsStore } from '../store/useUserEventsStore';
import { CoalescedRequest } from '../utils/coalescedRequest';
import { formatAppError } from '../utils/appError';

// Keep notifications current even while the dashboard is not mounted.
export function useUserEventsSync() {
    const userId = useAuthStore((state) => state.user?.id);
    useEffect(() => {
        const request = new CoalescedRequest<{ events: Awaited<ReturnType<typeof GetCurrentUserEvents>>['items']; unreadCount: number }>();
        const known = new Set<string>();
        let initialized = false;
        const refresh = () => request.refresh(async () => {
            const [result, unreadCount] = await Promise.all([
                GetCurrentUserEvents(models.UserEventFilter.createFrom({ page: 1, pageSize: 20 })), GetUnreadCount(),
            ]);
            return { events: result?.items || [], unreadCount };
        }, {
            onSuccess: ({ events, unreadCount }) => {
                const added = events.filter((event) => !event.readAt && !known.has(event.id));
                events.forEach((event) => known.add(event.id));
                useUserEventsStore.setState({ events, unreadCount, error: '' });
                if (initialized) emitUserEventsReceived(added);
                initialized = true;
            },
            onError: (error) => useUserEventsStore.setState({ events: [], unreadCount: 0, error: formatAppError(error, 'Не удалось загрузить события') }),
        });
        useUserEventsStore.setState({ events: [], unreadCount: 0, error: '', refresh });
        void refresh();
        const stopServer = onServerEvent((event) => {
            if (['user-events', 'resync', 'access-changed'].includes(event.topic)) void refresh();
        });
        const stopRead = onUserEventsDocumentRead(() => { void refresh(); });
        return () => {
            request.invalidate(); stopServer(); stopRead();
            useUserEventsStore.setState({ events: [], unreadCount: 0, error: '', refresh: () => Promise.resolve() });
        };
    }, [userId]);
}
