import React from 'react';
import { afterEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import DashboardPage from '../../src/pages/DashboardPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { renderWithApp } from '../componentTestUtils';

const api = vi.hoisted(() => ({ GetOverview: vi.fn(), ListAcknowledgments: vi.fn(), GetCurrentUserEvents: vi.fn(), serverListeners: [] as Array<(event: { topic: string }) => void>, registrationKinds: [] as Array<{ code: string; label: string; pageKey: string }> }));
vi.mock('../../wailsjs/go/services/WorkspaceService', () => ({ GetOverview: api.GetOverview, ListAcknowledgments: api.ListAcknowledgments }));
vi.mock('../../wailsjs/go/services/UserEventService', () => ({ GetCurrentUserEvents: api.GetCurrentUserEvents }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string }) => void) => {
    api.serverListeners.push(listener);
    return () => { api.serverListeners = api.serverListeners.filter((item) => item !== listener); };
} }));
vi.mock('../../src/components/DocumentViewModal', () => ({ default: ({ open, documentId }: { open: boolean; documentId: string }) => (
    open ? <div data-testid="opened-document">{documentId}</div> : null
) }));
vi.mock('../../src/hooks/useCurrentAccessSummary', () => ({ useCurrentAccessSummary: () => ({ ready: true, registrationKinds: api.registrationKinds }) }));

const overview = (assignmentMode: string, acknowledgmentMode: string, mixed = true) => ({
    assignmentModes: mixed ? ['execution', 'control'] : ['control'],
    assignmentMode: assignmentMode || (mixed ? 'execution' : 'control'),
    assignmentCounts: { new: 2, inProgress: 5, overdue: 1, dueSoon: 1, awaitingAcceptance: 3 },
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
    api.registrationKinds = [];
    useAuthStore.setState({ user: null, isAuthenticated: false });
});

test('mixed user switches assignment and acknowledgment scopes independently', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const openAssignments = vi.fn();
    renderWithApp(<DashboardPage onOpenAssignments={openAssignments} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    expect(screen.getByText('Мои ознакомления')).toBeInTheDocument();
    expect(screen.getByText('В работе')).toBeInTheDocument();
    const assignmentCard = screen.getByText('Поручения').closest('.ant-card')!;
    expect(assignmentCard.querySelector('.workspace-assignment-modes')).toHaveTextContent('( исполнение \\ контроль )');
    expect(within(assignmentCard as HTMLElement).getByText('Все поручения').closest('.ant-card-head'))
        .toBe(screen.getByText('Поручения').closest('.ant-card-head'));
    const summaryCard = screen.getByText('Новые').closest('.ant-card')?.parentElement?.closest('.ant-card');
    expect(summaryCard).toBeTruthy();
    expect(assignmentCard.contains(summaryCard)).toBe(false);
    fireEvent.click(screen.getByText('В работе').closest('[role="button"]')!);
    expect(openAssignments).toHaveBeenCalledWith('execution', 'in_progress');
    fireEvent.click(within(assignmentCard as HTMLElement).getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith('control', ''));
    expect(screen.getByText('Поручения')).toBeInTheDocument();
    expect(screen.getByText('Мои ознакомления')).toBeInTheDocument();
    await waitFor(() => expect(within(assignmentCard as HTMLElement).getByRole('button', { name: 'контроль' })).toHaveAttribute('aria-pressed', 'true'));
    fireEvent.click(screen.getByText('Ожидают приёмки').closest('[role="button"]')!);
    expect(openAssignments).toHaveBeenCalledWith('control', 'acceptance');
});

test('quick registration appears to the right of metrics and opens the selected form', async () => {
    setUser();
    api.registrationKinds = [
        { code: 'incoming_letter', label: 'Входящий документ', pageKey: 'incoming' },
        { code: 'outgoing_letter', label: 'Исходящий документ', pageKey: 'outgoing' },
    ];
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const onOpenRegister = vi.fn();
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={onOpenRegister} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    const row = screen.getByText('Новые').closest('.workspace-overview-row');
    expect(row).toBeInTheDocument();
    expect(row?.querySelector('.workspace-metrics-panel')).toBeInTheDocument();
    expect(row?.querySelector('.workspace-registration-panel')).toContainElement(screen.getByRole('button', { name: 'Входящий документ' }));
    expect(row?.children[0]).toHaveClass('workspace-work-column');
    expect(row?.children[1]).toHaveClass('workspace-side-column');
    const workColumn = row?.querySelector('.workspace-work-column');
    const sideColumn = row?.querySelector('.workspace-side-column');
    expect(workColumn?.children[0]).toHaveClass('workspace-metrics-panel');
    expect(workColumn?.children[1]).toContainElement(screen.getByText('Поручения'));
    expect(workColumn?.children[2]).toContainElement(screen.getByText('Мои ознакомления'));
    expect(sideColumn?.children[0]).toHaveClass('workspace-registration-panel');
    expect(sideColumn?.children[1]).toContainElement(screen.getByText('Новое для меня'));
    const registrationTable = row?.querySelector('.workspace-registration-table');
    expect(registrationTable?.querySelectorAll('tbody tr')).toHaveLength(2);
    expect(registrationTable?.querySelectorAll('.workspace-registration-icon--incoming')).toHaveLength(1);
    expect(registrationTable?.querySelectorAll('.workspace-registration-icon--outgoing')).toHaveLength(1);
    expect(registrationTable?.querySelectorAll('.workspace-registration-chevron')).toHaveLength(2);
    fireEvent.click(screen.getByRole('button', { name: 'Входящий документ' }));
    expect(onOpenRegister).toHaveBeenCalledWith('incoming_letter', 'incoming');
});

test('acknowledgments stay in the first column when assignments are unavailable', async () => {
    setUser();
    api.GetOverview.mockResolvedValue({
        ...overview('', ''),
        assignmentModes: [], assignmentMode: '', assignmentCounts: null, assignments: [],
        acknowledgmentModes: ['control'], acknowledgmentMode: 'control',
    });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Контроль ознакомлений')).toBeInTheDocument();
    const row = screen.getByText('Контроль ознакомлений').closest('.workspace-overview-row');
    expect(row?.querySelector('.workspace-work-column')).toContainElement(screen.getByText('Контроль ознакомлений'));
    expect(row?.querySelector('.workspace-side-column')).toContainElement(screen.getByText('Новое для меня'));
    expect(row?.querySelector('.workspace-metrics-panel')).not.toBeInTheDocument();
});

test('assignment preview shows registration details in five columns and opens its document', async () => {
    setUser();
    api.GetOverview.mockResolvedValue({
        ...overview('', ''),
        assignments: [{
            id: 'assignment-1', documentId: 'document-1', documentKind: 'incoming_letter',
            documentNumber: 'ИТ/42', documentDate: '2026-09-28T12:00:00Z',
            content: 'Подготовить ответ', deadline: '2026-10-01T12:00:00Z', status: 'in_progress',
        }],
    });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Подготовить ответ')).toBeInTheDocument();
    const table = screen.getByRole('table');
    expect(within(table).getAllByRole('columnheader').map((header) => header.textContent)).toEqual([
        'Срок', 'Документ', 'Содержание поручения', 'Статус', 'Действие',
    ]);
    expect(within(table).getByText('ИТ/42')).toBeInTheDocument();
    expect(within(table).getByText('28.09.2026')).toBeInTheDocument();
    expect(within(table).getByText('01.10.2026')).toBeInTheDocument();
    expect(within(table).getByText('В работе')).toBeInTheDocument();
    fireEvent.click(within(table).getByRole('button', { name: 'Открыть' }));
    expect(screen.getByTestId('opened-document')).toHaveTextContent('document-1');
});

test('controller only sees the control mode and opens its scoped acknowledgment list', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode, false));
    api.ListAcknowledgments.mockResolvedValue({ items: [], totalCount: 0, page: 1, pageSize: 10 });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    expect(screen.getByText('( контроль )')).toBeInTheDocument();
    expect(within(screen.getByText('Поручения').closest('.ant-card') as HTMLElement).queryByRole('button', { name: 'контроль' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Ожидают внимания: 4'));
    expect(api.ListAcknowledgments).toHaveBeenCalledWith('control', 1, 10);
});

test('failed refresh hides stale task counts and shows the access error', async () => {
    setUser();
    api.GetOverview.mockResolvedValueOnce(overview('', '')).mockRejectedValueOnce(new Error('Доступ прекращён'));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Обновить'));
    await waitFor(() => expect(screen.queryByText('Поручения')).not.toBeInTheDocument());
    expect(screen.getByText('Не удалось выполнить действие. Повторите попытку или обратитесь к администратору, если ошибка повторяется.')).toBeInTheDocument();
});

test('resync rereads tasks and personal events from the server', async () => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
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
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    fireEvent.click(within(screen.getByText('Поручения').closest('.ant-card') as HTMLElement).getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenCalledTimes(3));
    expect(api.GetOverview).toHaveBeenLastCalledWith('', '');
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
});
