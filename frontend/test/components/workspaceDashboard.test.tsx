import React from 'react';
import { afterEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import DashboardPage from '../../src/pages/DashboardPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { renderWithApp } from '../componentTestUtils';

const api = vi.hoisted(() => ({ GetOverview: vi.fn(), ListAcknowledgments: vi.fn(), GetCurrentUserEvents: vi.fn(), serverListeners: [] as Array<(event: { topic: string }) => void> }));
vi.mock('../../wailsjs/go/services/WorkspaceService', () => ({ GetOverview: api.GetOverview, ListAcknowledgments: api.ListAcknowledgments }));
vi.mock('../../wailsjs/go/services/UserEventService', () => ({ GetCurrentUserEvents: api.GetCurrentUserEvents }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string }) => void) => {
    api.serverListeners.push(listener);
    return () => { api.serverListeners = api.serverListeners.filter((item) => item !== listener); };
} }));
vi.mock('../../src/components/DocumentViewModal', () => ({ default: () => null }));
vi.mock('../../src/hooks/useCurrentAccessSummary', () => ({ useCurrentAccessSummary: () => ({ ready: true, registrationKinds: [] }) }));

const overview = (assignmentMode: string, acknowledgmentMode: string, mixed = true) => ({
    assignmentModes: mixed ? ['execution', 'control'] : ['control'],
    assignmentMode: assignmentMode || (mixed ? 'execution' : 'control'),
    assignmentCounts: { new: 2, overdue: 1, dueSoon: 1, awaitingAcceptance: 3 },
    assignments: [],
    acknowledgmentModes: mixed ? ['execution', 'control'] : ['control'],
    acknowledgmentMode: acknowledgmentMode || (mixed ? 'execution' : 'control'),
    acknowledgmentCount: 4,
    acknowledgments: [],
});
const setUser = () => useAuthStore.setState({
    isAuthenticated: true,
    user: { id: 'workspace-user', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [] },
});

afterEach(() => {
    api.GetOverview.mockReset();
    api.ListAcknowledgments.mockReset();
    api.GetCurrentUserEvents.mockReset();
    api.serverListeners = [];
    useAuthStore.setState({ user: null, isAuthenticated: false });
});

test('mixed user switches assignment and acknowledgment scopes independently', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const openAssignments = vi.fn();
    renderWithApp(<DashboardPage onOpenAssignments={openAssignments} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Мои поручения')).toBeInTheDocument();
    expect(screen.getByText('Мои ознакомления')).toBeInTheDocument();
    const assignmentCard = screen.getByText('Мои поручения').closest('.ant-card')!;
    fireEvent.click(within(assignmentCard as HTMLElement).getByText('Контроль'));
    expect(await screen.findByText('Поручения под контролем')).toBeInTheDocument();
    expect(screen.getByText('Мои ознакомления')).toBeInTheDocument();
    expect(api.GetOverview).toHaveBeenLastCalledWith('control', '');
    fireEvent.click(screen.getByText('Ожидают приёмки').closest('[role="button"]')!);
    expect(openAssignments).toHaveBeenCalledWith('control', 'acceptance');
});

test('controller only sees the control mode and opens its scoped acknowledgment list', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode, false));
    api.ListAcknowledgments.mockResolvedValue({ items: [], totalCount: 0, page: 1, pageSize: 10 });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения под контролем')).toBeInTheDocument();
    expect(screen.queryByText('Мои поручения')).not.toBeInTheDocument();
    expect(screen.queryByRole('radio')).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Ожидают внимания: 4'));
    expect(api.ListAcknowledgments).toHaveBeenCalledWith('control', 1, 10);
});

test('failed refresh hides stale task counts and shows the access error', async () => {
    setUser();
    api.GetOverview.mockResolvedValueOnce(overview('', '')).mockRejectedValueOnce(new Error('Доступ прекращён'));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Мои поручения')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Обновить'));
    await waitFor(() => expect(screen.queryByText('Мои поручения')).not.toBeInTheDocument());
    expect(screen.getByText('Не удалось выполнить действие. Повторите попытку или обратитесь к администратору, если ошибка повторяется.')).toBeInTheDocument();
});

test('resync rereads tasks and personal events from the server', async () => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Мои поручения')).toBeInTheDocument();
    await waitFor(() => expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(1));
    await act(async () => { api.serverListeners.forEach((listener) => listener({ topic: 'resync' })); });
    await waitFor(() => expect(api.GetOverview).toHaveBeenCalledTimes(2));
    expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(2);
});

test('revoked control permission resets the selected mode', async () => {
    setUser();
    let requests = 0;
    api.GetOverview.mockImplementation(async (assignmentMode: string) => {
        requests += 1;
        if (assignmentMode === 'control') {
            throw new Error(JSON.stringify({ code: 'FORBIDDEN', status: 403, message: 'Доступ отозван' }));
        }
        return requests === 1 ? overview('', '') : { ...overview('', ''), assignmentModes: ['execution'], acknowledgmentModes: ['execution'] };
    });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Мои поручения')).toBeInTheDocument();
    fireEvent.click(within(screen.getByText('Мои поручения').closest('.ant-card') as HTMLElement).getByText('Контроль'));
    await waitFor(() => expect(api.GetOverview).toHaveBeenCalledTimes(3));
    expect(api.GetOverview).toHaveBeenLastCalledWith('', '');
    expect(await screen.findByText('Мои поручения')).toBeInTheDocument();
});
