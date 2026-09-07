import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { useAuthStore } from '../../src/store/useAuthStore';
import { useDraftLinkStore } from '../../src/store/useDraftLinkStore';
import { useRegisterDocumentStore } from '../../src/store/useRegisterDocumentStore';
import { cachedSummary, loadAccessSummary, resetCurrentAccessSummaryCache } from '../../src/store/accessSummaryCache';
import { useSessionEvents } from '../../src/hooks/useSessionEvents';
import { serverclient, dto } from '../../wailsjs/go/models';

const api = vi.hoisted(() => ({
    Login: vi.fn(), Logout: vi.fn(), GetSessionState: vi.fn(), ChangePassword: vi.fn(),
    ChangeRequiredPassword: vi.fn(), UpdateProfile: vi.fn(), GetCurrentAccessSummary: vi.fn(),
    EventsOn: vi.fn(), unsubscribe: vi.fn(),
}));
vi.mock('../../wailsjs/go/services/AuthService', () => api);
vi.mock('../../wailsjs/go/services/DocumentKindService', () => api);
vi.mock('../../wailsjs/runtime/runtime', () => ({ EventsOn: api.EventsOn }));

const user = { id: 'user-1', login: 'user', fullName: 'User', isActive: true, isDocumentParticipant: false, failedLoginAttempts: 0, systemPermissions: ['admin'] };
const state = (revision: number, authenticated = false, reason = 'session_invalid') => (
    serverclient.SessionState.createFrom({ revision, authenticated, userId: authenticated ? user.id : '', reason })
);
function deferred<T>() {
    let resolve!: (value: T) => void;
    let reject!: (reason: unknown) => void;
    const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
    return { promise, resolve, reject };
}

beforeEach(() => {
    vi.resetAllMocks();
    api.EventsOn.mockReturnValue(api.unsubscribe);
    api.GetSessionState.mockResolvedValue(state(1, true));
    useAuthStore.setState({ user, isAuthenticated: true, sessionRevision: 1, authAttempt: 0, error: null, isLoading: false });
    useDraftLinkStore.getState().clearDraftLink();
    useRegisterDocumentStore.getState().clearRequest();
    resetCurrentAccessSummaryCache();
});

describe('session invalidation', () => {
    test('event clears authentication, sensitive drafts and access cache; subscription is removed', async () => {
        api.GetCurrentAccessSummary.mockResolvedValue(dto.CurrentAccessSummary.createFrom({}));
        await loadAccessSummary(user.id);
        expect(cachedSummary).not.toBeNull();
        useDraftLinkStore.getState().setDraftLink('document-secret', 'incoming_letter', 'secret-number', 'outgoing_letter');
        useRegisterDocumentStore.getState().requestOpen('outgoing_letter', { subject: 'secret' });
        const { unmount } = renderHook(useSessionEvents);
        expect(api.EventsOn).toHaveBeenCalledWith('auth:session-ended', expect.any(Function));
        act(() => api.EventsOn.mock.calls[0][1](state(2)));
        expect(useAuthStore.getState()).toMatchObject({ user: null, isAuthenticated: false, isLoading: false, error: 'Сессия завершена. Войдите снова.' });
        expect(useDraftLinkStore.getState().sourceId).toBe('');
        expect(useRegisterDocumentStore.getState().initialValues).toBeNull();
        expect(cachedSummary).toBeNull();
        expect(api.Logout).not.toHaveBeenCalled();
        unmount();
        expect(api.unsubscribe).toHaveBeenCalledOnce();
    });

    test('snapshot recovers an invalidation missed before subscribing', async () => {
        api.GetSessionState.mockResolvedValue(state(2));
        renderHook(useSessionEvents);
        await waitFor(() => expect(useAuthStore.getState().isAuthenticated).toBe(false));
    });

    test('late and duplicate events do not erase a new login or repeat cleanup', () => {
        useAuthStore.getState().sessionEnded(state(2));
        useAuthStore.setState({ user, isAuthenticated: true, sessionRevision: 3 });
        useDraftLinkStore.getState().setDraftLink('new-document', 'incoming_letter', 'new', 'outgoing_letter');
        useAuthStore.getState().sessionEnded(state(2));
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
        expect(useDraftLinkStore.getState().sourceId).toBe('new-document');
    });

    test('a pending access response cannot repopulate the cache after logout', async () => {
        const response = deferred<dto.CurrentAccessSummary>();
        api.GetCurrentAccessSummary.mockReturnValue(response.promise);
        const load = loadAccessSummary(user.id);
        useAuthStore.getState().sessionEnded(state(2));
        response.resolve(dto.CurrentAccessSummary.createFrom({}));
        await load;
        expect(cachedSummary).toBeNull();
    });

    test('session invalidated while login resolves cannot restore authentication', async () => {
        const snapshot = deferred<serverclient.SessionState>();
        api.Login.mockResolvedValue(user);
        api.GetSessionState.mockReturnValue(snapshot.promise);
        const login = useAuthStore.getState().login('user', 'password');
        await waitFor(() => expect(api.GetSessionState).toHaveBeenCalled());
        useAuthStore.getState().sessionEnded(state(3));
        snapshot.resolve(state(2, true));
        await login;
        expect(useAuthStore.getState().isAuthenticated).toBe(false);
        expect(useAuthStore.getState().sessionRevision).toBe(3);
    });

    test('a new login succeeds despite a late event for the previous session', async () => {
        const loginResult = deferred<typeof user>();
        api.Login.mockReturnValue(loginResult.promise);
        api.GetSessionState.mockResolvedValue(state(3, true));
        const login = useAuthStore.getState().login('user', 'password');
        useAuthStore.getState().sessionEnded(state(2));
        loginResult.resolve(user);
        await login;
        expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: true, sessionRevision: 3, error: null });
        useAuthStore.getState().sessionEnded(state(2));
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    test('logout is immediate and its delayed completion cannot erase a new login', async () => {
        const response = deferred<void>();
        api.Logout.mockReturnValue(response.promise);
        const logout = useAuthStore.getState().logout();
        expect(useAuthStore.getState().isAuthenticated).toBe(false);
        api.Login.mockResolvedValue(user);
        api.GetSessionState.mockResolvedValue(state(3, true));
        await useAuthStore.getState().login('user', 'password');
        response.resolve();
        await logout;
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    test('logout prevents a pending login from restoring the UI', async () => {
        const response = deferred<typeof user>();
        api.Login.mockReturnValue(response.promise);
        const login = useAuthStore.getState().login('user', 'password');
        await useAuthStore.getState().logout();
        response.resolve(user);
        await login;
        expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    test('password change transport failure synchronizes cleared backend state', async () => {
        api.ChangePassword.mockRejectedValue(new Error('network failure'));
        api.GetSessionState.mockResolvedValue(state(2, false, 'password_changed'));
        await expect(useAuthStore.getState().changePassword('old', 'new')).rejects.toThrow('network failure');
        expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: false, user: null, isLoading: false });
    });

    test('wrong password keeps the current session', async () => {
        api.ChangePassword.mockRejectedValue(new Error('wrong password'));
        await expect(useAuthStore.getState().changePassword('old', 'new')).rejects.toThrow();
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
        expect(useAuthStore.getState().isLoading).toBe(false);
    });

    test('old profile result cannot change the new session user', async () => {
        const response = deferred<void>();
        api.UpdateProfile.mockReturnValue(response.promise);
        const update = useAuthStore.getState().updateProfile('old-login', 'old-name');
        useAuthStore.getState().sessionEnded(state(2));
        useAuthStore.setState({ user: { ...user, fullName: 'New session' }, isAuthenticated: true, sessionRevision: 3 });
        response.resolve();
        await update;
        expect(useAuthStore.getState().user?.fullName).toBe('New session');
    });
});
