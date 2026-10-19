import React, { useState } from 'react';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import AppRouter from '../../src/components/AppRouter';
import { useCurrentAccessSummary } from '../../src/hooks/useCurrentAccessSummary';
import { onServerEvent } from '../../src/events/serverEvents';
import { resetCurrentAccessSummaryCache } from '../../src/store/accessSummaryCache';
import { useAuthStore } from '../../src/store/useAuthStore';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/pages/IncomingPage', () => ({ default: function IncomingPage() {
    const [open, setOpen] = useState(false);
    return <><button onClick={() => setOpen(true)}>Открыть документ</button>{open && <div role="dialog">Карточка документа</div>}</>;
} }));
vi.mock('../../src/pages/DeadlineDashboardPage', () => ({ default: () => <div>Рабочий стол</div> }));
const previous = useAuthStore.getState();
afterEach(() => { useAuthStore.setState(previous); resetCurrentAccessSummaryCache(); });
const summary = { sections: { dashboard: true, incoming: true }, documentKinds: [], systemPermissions: [] };
const Router = () => {
    const access = useCurrentAccessSummary();
    return <AppRouter currentPage="incoming" fallbackPage="dashboard" accessReady={access.ready}
        canAccessPage={access.canAccessPage} deadlineNavigation={null} onOpenControls={vi.fn()} onOpenRegister={vi.fn()} />;
};

test.each([false, true])('resync preserves open document during access refresh and handles revocation: %s', async (revoke) => {
    resetCurrentAccessSummaryCache();
    useAuthStore.setState({ isAuthenticated: true, sessionRevision: 8, user: { ...previous.user!, id: 'user-1' } });
    const pending = deferred<typeof summary>();
    const get = vi.fn().mockResolvedValueOnce(summary).mockReturnValueOnce(pending.promise);
    installWailsMock({ DocumentKindService: { GetCurrentAccessSummary: get } });
    const stop = onServerEvent(vi.fn());
    const view = renderWithApp(<Router />);
    try {
        fireEvent.click(await screen.findByRole('button', { name: 'Открыть документ' }));
        expect(screen.getByRole('dialog')).toHaveTextContent('Карточка документа');
        act(() => fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'resync', revision: 8 } })));
        await waitFor(() => expect(get).toHaveBeenCalledTimes(2));
        expect(screen.getByRole('dialog')).toHaveTextContent('Карточка документа');
        await act(async () => pending.resolve({ ...summary, sections: { dashboard: true, incoming: !revoke } }));
        if (revoke) {
            expect(await screen.findByText('Рабочий стол')).toBeInTheDocument();
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
        } else {
            expect(screen.getByRole('dialog')).toHaveTextContent('Карточка документа');
        }
    } finally { stop(); view.unmount(); }
});
