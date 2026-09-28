import { act, fireEvent, renderHook, waitFor } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import { useCurrentAccessSummary } from '../../src/hooks/useCurrentAccessSummary';
import { resetCurrentAccessSummaryCache } from '../../src/store/accessSummaryCache';
import { onServerEvent } from '../../src/events/serverEvents';
import { useAuthStore } from '../../src/store/useAuthStore';
import { deferred, installWailsMock } from '../componentTestUtils';

const previous = useAuthStore.getState();
afterEach(() => { useAuthStore.setState(previous); resetCurrentAccessSummaryCache(); });

const summary = { sections: { dashboard: true }, documentKinds: [], systemPermissions: [] };

test.each([false, true])('access invalidation shares one background request and preserves ready state (failure: %s)', async (failure) => {
    resetCurrentAccessSummaryCache();
    useAuthStore.setState({ isAuthenticated: true, sessionRevision: 8, user: { ...previous.user!, id: 'user-1' } });
    const pending = deferred<typeof summary>();
    const get = vi.fn().mockResolvedValueOnce(summary).mockReturnValueOnce(pending.promise);
    installWailsMock({ DocumentKindService: { GetCurrentAccessSummary: get } });
    const first = renderHook(() => useCurrentAccessSummary());
    const second = renderHook(() => useCurrentAccessSummary());
    await waitFor(() => expect(first.result.current.ready && second.result.current.ready).toBe(true));
    expect(get).toHaveBeenCalledTimes(1);
    const stopA = onServerEvent(vi.fn());
    const stopB = onServerEvent(vi.fn());
    try {
        act(() => fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'access-changed', revision: 8 } })));
        await waitFor(() => expect(get).toHaveBeenCalledTimes(2));
        expect(first.result.current.ready && second.result.current.ready).toBe(true);
        expect(first.result.current.sections.dashboard).toBe(true);
        await act(async () => {
            if (failure) pending.reject(new Error('Network unavailable'));
            else pending.resolve({ ...summary, sections: { dashboard: false } });
        });
        expect(first.result.current.sections.dashboard).toBe(failure);
        expect(second.result.current.sections.dashboard).toBe(failure);
        expect(first.result.current.ready && second.result.current.ready).toBe(true);
    } finally { stopA(); stopB(); first.unmount(); second.unmount(); }
});
