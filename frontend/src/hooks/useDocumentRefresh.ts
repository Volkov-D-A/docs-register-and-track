import { useEffect, useRef } from 'react';
import { onServerEvent, type DocumentResource } from '../events/serverEvents';

// A mounted component owns its subscription; bursts cause one background refresh.
export const useDocumentRefresh = (
    documentId: string,
    resource: DocumentResource | 'journal',
    refresh: () => unknown,
    enabled = true,
) => {
    const refreshRef = useRef(refresh);
    useEffect(() => { refreshRef.current = refresh; }, [refresh]);
    useEffect(() => {
        if (!enabled || !documentId) return;
        let timer: ReturnType<typeof setTimeout> | undefined;
        const stop = onServerEvent((event) => {
            const resync = event.topic === 'resync' || event.topic === 'access-changed' || event.topic === 'documents';
            const changed = event.topic === 'document-changed' && event.documentId === documentId
                && (event.resource === resource || resource === 'journal' || event.resource === 'document'
                    || (resource === 'document' && event.visibilityChanged));
            if (!resync && !changed) return;
            clearTimeout(timer);
            timer = setTimeout(() => { void refreshRef.current(); }, 100);
        });
        return () => { clearTimeout(timer); stop(); };
    }, [documentId, resource, enabled]);
};

// Register pages listen to their document kind; task changes only affect the
// register when they can change implicit document visibility.
export const useDocumentListRefresh = (kindCode: string, refresh: () => unknown, enabled: boolean) => {
    const refreshRef = useRef(refresh);
    useEffect(() => { refreshRef.current = refresh; }, [refresh]);
    useEffect(() => {
        if (!enabled) return;
        let timer: ReturnType<typeof setTimeout> | undefined;
        const stop = onServerEvent((event) => {
            const resync = event.topic === 'resync' || event.topic === 'access-changed' || event.topic === 'documents';
            const changed = event.topic === 'document-changed'
                && (!event.documentKind || event.documentKind === kindCode)
                && (event.resource === 'document' || event.visibilityChanged);
            if (!resync && !changed) return;
            clearTimeout(timer);
            timer = setTimeout(() => { void refreshRef.current(); }, 100);
        });
        return () => { clearTimeout(timer); stop(); };
    }, [kindCode, enabled]);
};
