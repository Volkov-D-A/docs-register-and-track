import { useEffect } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { GetSessionState } from '../../wailsjs/go/services/AuthService';
import { useAuthStore } from '../store/useAuthStore';

export const useSessionEvents = () => {
    useEffect(() => {
        let active = true;
        const unsubscribe = EventsOn('auth:session-ended', (state) => {
            if (active) useAuthStore.getState().sessionEnded(state);
        });
        // Subscribe before reading: an event missed during mounting is recovered
        // by the snapshot; out-of-order responses are filtered by revision.
        void GetSessionState().then((state) => {
            if (active) useAuthStore.getState().sessionEnded(state);
        }).catch(() => { /* A failed local snapshot does not revoke a session. */ });
        return () => { active = false; unsubscribe(); };
    }, []);
};
