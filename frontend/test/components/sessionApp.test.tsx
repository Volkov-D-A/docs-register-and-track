import React from 'react';
import { act, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import App from '../../src/App';
import { useAuthStore } from '../../src/store/useAuthStore';
import { renderWithApp } from '../componentTestUtils';

const api = vi.hoisted(() => ({ EventsOn: vi.fn(), GetSessionState: vi.fn(), NeedsInitialSetup: vi.fn() }));
vi.mock('../../wailsjs/runtime/runtime', () => ({ EventsOn: api.EventsOn }));
vi.mock('../../wailsjs/go/services/AuthService', async (original) => ({
    ...await original<object>(), GetSessionState: api.GetSessionState, NeedsInitialSetup: api.NeedsInitialSetup,
}));
vi.mock('../../src/components/SystemBootstrapGate', () => ({ default: ({ children }: { children: React.ReactNode }) => <>{children}</> }));
vi.mock('../../src/components/MainLayout', () => ({ default: ({ children }: { children: React.ReactNode }) => <>{children}</> }));
vi.mock('../../src/components/AppRouter', () => ({ default: () => <div>Private document content</div> }));
vi.mock('../../src/components/modals/OrganizationSetupModal', () => ({ default: () => null }));
vi.mock('../../src/hooks/useOrganizationSetup', () => ({ useOrganizationSetup: () => ({ open: false }) }));
vi.mock('../../src/hooks/useCurrentAccessSummary', () => ({ useCurrentAccessSummary: () => ({
    summary: {}, kinds: [], loading: false, ready: true, sections: {},
    canAccessPage: () => true, getDefaultPage: () => 'dashboard',
}) }));
vi.mock('../../wailsjs/go/services/ReleaseNoteService', () => ({ GetCurrent: vi.fn().mockResolvedValue(null), MarkCurrentViewed: vi.fn() }));

test('session event removes protected screens and shows the login explanation', async () => {
    api.EventsOn.mockReturnValue(vi.fn());
    api.GetSessionState.mockResolvedValue({ revision: 1, authenticated: true, userId: 'user-1', reason: '' });
    api.NeedsInitialSetup.mockResolvedValue(false);
    useAuthStore.setState({ user: { id: 'user-1', login: 'user', fullName: 'User', isActive: true, isDocumentParticipant: false, failedLoginAttempts: 0, systemPermissions: [] }, isAuthenticated: true, sessionRevision: 1 });
    renderWithApp(<App />);
    expect(screen.getByText('Private document content')).toBeInTheDocument();
    act(() => api.EventsOn.mock.calls[0][1]({ revision: 2, authenticated: false, userId: '', reason: 'session_invalid' }));
    expect(screen.queryByText('Private document content')).not.toBeInTheDocument();
    expect(await screen.findByText('Сессия завершена. Войдите снова.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Войти/ })).toBeInTheDocument();
});
