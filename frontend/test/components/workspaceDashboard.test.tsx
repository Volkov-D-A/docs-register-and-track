import React from 'react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import DashboardPage from '../../src/pages/DashboardPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { deferred, renderWithApp } from '../componentTestUtils';

const api = vi.hoisted(() => ({ GetRecentDocuments: vi.fn(), GetOverview: vi.fn(), ListAcknowledgments: vi.fn(), GetCurrentUserEvents: vi.fn(), serverListeners: [] as Array<(event: { topic: string; resource?: string; documentId?: string; visibilityChanged?: boolean }) => void>, registrationKinds: [] as Array<{ code: string; label: string; pageKey: string }> }));
vi.mock('../../wailsjs/go/services/WorkspaceService', () => ({ GetRecentDocuments: api.GetRecentDocuments, GetOverview: api.GetOverview, ListAcknowledgments: api.ListAcknowledgments }));
vi.mock('../../wailsjs/go/services/UserEventService', () => ({ GetCurrentUserEvents: api.GetCurrentUserEvents }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string; resource?: string; documentId?: string; visibilityChanged?: boolean }) => void) => {
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
    user: { id: 'workspace-user', login: 'tester', fullName: 'Иванов Иван Иванович', lastName: 'Иванов', firstName: 'Иван', patronymic: 'Иванович', noPatronymic: false, isDocumentParticipant: true, systemPermissions: [] },
});

beforeEach(() => { api.GetRecentDocuments.mockResolvedValue({ available: false, items: [] }); });

afterEach(() => {
    api.GetRecentDocuments.mockReset();
    api.GetOverview.mockReset();
    api.ListAcknowledgments.mockReset();
    api.GetCurrentUserEvents.mockReset();
    api.serverListeners = [];
    api.registrationKinds = [];
    useAuthStore.setState({ user: null, isAuthenticated: false });
});

test('greeting follows profile changes without reloading workspace data', async () => {
    vi.spyOn(Date.prototype, 'getHours').mockReturnValue(12);
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByRole('heading', { name: 'Добрый день, Иван Иванович!' })).toBeInTheDocument();
    await screen.findByText('Поручения');
    const calls = api.GetOverview.mock.calls.length;
    act(() => {
        const user = useAuthStore.getState().user!;
        useAuthStore.setState({ user: { ...user, firstName: 'Анна', patronymic: 'Ивановна', noPatronymic: true } });
    });
    expect(screen.getByRole('heading', { name: 'Добрый день, Анна!' })).toBeInTheDocument();
    expect(api.GetOverview).toHaveBeenCalledTimes(calls);
});

test('one switch beside the greeting changes both task scopes and list navigation', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode));
    api.ListAcknowledgments.mockResolvedValue({ items: [], totalCount: 0, page: 1, pageSize: 10 });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const openAssignments = vi.fn();
    renderWithApp(<DashboardPage onOpenAssignments={openAssignments} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    const modes = screen.getByRole('group', { name: 'Режим рабочего стола' });
    expect(modes).toHaveTextContent('( исполнение \\ контроль )');
    const heading = screen.getByRole('heading');
    expect(heading.parentElement).toHaveClass('workspace-heading');
    expect(heading.nextElementSibling).toBe(modes);
    expect(screen.getAllByRole('button', { name: 'контроль' })).toHaveLength(1);
    for (const [title, link] of [['Поручения', 'Все поручения'], ['Ознакомления', 'Все ознакомления']]) {
        const card = screen.getByText(title).closest('.ant-card') as HTMLElement;
        expect(within(card).queryByRole('button', { name: 'контроль' })).not.toBeInTheDocument();
        expect(within(card).getByText(link).closest('.ant-card-head')).toBe(screen.getByText(title).closest('.ant-card-head'));
    }
    expect(screen.queryByText(/Ожидают внимания/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('В работе').closest('[role="button"]')!);
    expect(openAssignments).toHaveBeenLastCalledWith('execution', 'in_progress');
    fireEvent.click(within(modes).getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith('control', 'control'));
    await waitFor(() => expect(screen.getByText('Ожидают приёмки')).toBeInTheDocument());
    expect(within(modes).getByRole('button', { name: 'контроль' })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByText('Ожидают приёмки').closest('[role="button"]')!);
    expect(openAssignments).toHaveBeenLastCalledWith('control', 'acceptance');
    fireEvent.click(screen.getByRole('button', { name: 'Все поручения' }));
    expect(openAssignments).toHaveBeenLastCalledWith('control');
    fireEvent.click(screen.getByRole('button', { name: 'Все ознакомления' }));
    await waitFor(() => expect(api.ListAcknowledgments).toHaveBeenLastCalledWith('control', 1, 10));
    fireEvent.click(within(modes).getByRole('button', { name: 'исполнение' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith('execution', 'execution'));
    await waitFor(() => expect(screen.queryByText('Ожидают приёмки')).not.toBeInTheDocument());
    expect(within(modes).getByRole('button', { name: 'исполнение' })).toHaveAttribute('aria-pressed', 'true');
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
    expect(screen.getByRole('searchbox', { name: 'Поиск по документам' })).toBeInTheDocument();
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
    expect(workColumn?.children[2]).toContainElement(screen.getByText('Ознакомления'));
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
    expect(await screen.findByText('Ознакомления')).toBeInTheDocument();
    const row = screen.getByText('Ознакомления').closest('.workspace-overview-row');
    expect(row?.querySelector('.workspace-work-column')).toContainElement(screen.getByText('Ознакомления'));
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
    const table = within(screen.getByText('Поручения').closest('.ant-card') as HTMLElement).getByRole('table');
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
    expect(screen.queryByRole('group', { name: 'Режим рабочего стола' })).not.toBeInTheDocument();
    expect(within(screen.getByText('Поручения').closest('.ant-card') as HTMLElement).queryByRole('button', { name: 'контроль' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByText('Все ознакомления'));
    expect(api.ListAcknowledgments).toHaveBeenCalledWith('control', 1, 10);
});

test('failed refresh hides stale task counts and shows the access error', async () => {
    setUser();
    api.GetOverview.mockResolvedValueOnce(overview('', '')).mockRejectedValueOnce(new Error('Доступ прекращён'))
        .mockResolvedValueOnce(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    await act(async () => { api.serverListeners.forEach((listener) => listener({ topic: 'documents' })); });
    await waitFor(() => expect(screen.queryByText('Поручения')).not.toBeInTheDocument());
    const retry = screen.getByRole('button', { name: 'Повторить' });
    expect(within(retry.closest('.ant-card') as HTMLElement).getByText('Не удалось выполнить действие. Повторите попытку или обратитесь к администратору, если ошибка повторяется.')).toBeInTheDocument();
    fireEvent.click(retry);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
});

test.each(['resync', 'access-changed'])('%s rereads every workspace block from the server', async (topic) => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    await waitFor(() => expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(1));
    const documentRequests = api.GetRecentDocuments.mock.calls.length;
    expect(screen.queryByRole('button', { name: 'Обновить' })).not.toBeInTheDocument();
    await act(async () => { api.serverListeners.forEach((listener) => listener({ topic })); });
    await waitFor(() => expect(api.GetOverview).toHaveBeenCalledTimes(2));
    expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(2);
    expect(api.GetRecentDocuments).toHaveBeenCalledTimes(documentRequests + 1);
});

test('returning to the window refreshes all blocks once and preserves control mode', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    const { unmount } = renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    await screen.findByText('Поручения');
    fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith('control', 'control'));
    const counts = [api.GetOverview.mock.calls.length, api.GetCurrentUserEvents.mock.calls.length, api.GetRecentDocuments.mock.calls.length];
    visibility.mockReturnValue('hidden');
    fireEvent(document, new Event('visibilitychange'));
    fireEvent(window, new Event('focus'));
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 200)); });
    expect(api.GetOverview).toHaveBeenCalledTimes(counts[0]);
    visibility.mockReturnValue('visible');
    fireEvent(document, new Event('visibilitychange'));
    fireEvent(window, new Event('focus'));
    await waitFor(() => expect(api.GetRecentDocuments).toHaveBeenCalledTimes(counts[2] + 1));
    expect(api.GetOverview).toHaveBeenCalledTimes(counts[0] + 1);
    expect(api.GetOverview).toHaveBeenLastCalledWith('control', 'control');
    expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(counts[1] + 1);
    fireEvent(window, new Event('focus'));
    unmount();
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 200)); });
    expect(api.GetOverview).toHaveBeenCalledTimes(counts[0] + 1);
});

test('document invalidation updates the open acknowledgment list with an unchanged count', async () => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const acknowledgment = {
        id: 'ack-1', documentId: 'ack-doc', documentKind: 'incoming_letter',
        documentNumber: '43', documentContent: 'Документ', content: 'Старая резолюция',
    };
    api.ListAcknowledgments.mockResolvedValueOnce({ items: [acknowledgment], totalCount: 1 })
        .mockResolvedValueOnce({ items: [{ ...acknowledgment, content: 'Новая резолюция' }], totalCount: 1 });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    await screen.findByText('Ознакомления');
    fireEvent.click(screen.getByRole('button', { name: 'Все ознакомления' }));
    const drawer = await screen.findByRole('dialog');
    await within(drawer).findByText('Старая резолюция');
    await act(async () => { api.serverListeners.forEach((listener) => listener({ topic: 'documents' })); });
    await within(drawer).findByText('Новая резолюция');
    expect(within(drawer).queryByText('Старая резолюция')).not.toBeInTheDocument();
    expect(api.ListAcknowledgments).toHaveBeenCalledTimes(2);
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
    fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenCalledTimes(3));
    expect(api.GetOverview).toHaveBeenLastCalledWith('', '');
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: 'Режим рабочего стола' })).not.toBeInTheDocument();
});

test('acknowledgment preview and full list show document details and resolution and open the document', async () => {
    setUser();
    const acknowledgment = {
        id: 'ack-1', documentId: 'document-ack', documentKind: 'incoming_letter',
        documentNumber: 'ИТ/43', documentDate: '2026-09-28T12:00:00Z',
        documentContent: 'Об изменении графика', content: 'Принять к сведению',
    };
    api.GetOverview.mockResolvedValue({ ...overview('', ''), acknowledgments: [acknowledgment] });
    api.ListAcknowledgments.mockResolvedValue({ items: [acknowledgment], totalCount: 1, page: 1, pageSize: 10 });
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    const card = (await screen.findByText('Ознакомления')).closest('.ant-card') as HTMLElement;
    const checkTable = (table: HTMLElement) => {
        expect(within(table).getAllByRole('columnheader').map((header) => header.textContent)).toEqual([
            'Документ', 'Содержание', 'Резолюция', 'Действие',
        ]);
        for (const value of ['ИТ/43', '28.09.2026', 'Об изменении графика', 'Принять к сведению']) {
            expect(within(table).getByText(value)).toBeInTheDocument();
        }
        fireEvent.click(within(table).getByRole('button', { name: 'Открыть' }));
        expect(screen.getByTestId('opened-document')).toHaveTextContent('document-ack');
    };
    checkTable(within(card).getByRole('table'));
    fireEvent.click(within(card).getByRole('button', { name: 'Все ознакомления' }));
    const drawer = await screen.findByRole('dialog');
    await waitFor(() => expect(within(drawer).getByText('Принять к сведению')).toBeInTheDocument());
    expect(api.ListAcknowledgments).toHaveBeenCalledWith('execution', 1, 10);
    checkTable(within(drawer).getByRole('table'));
});

test.each(['assignments', 'acknowledgments'])('unified mode respects control access limited to %s', async (controlBlock) => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => ({
        ...overview(assignmentMode, acknowledgmentMode),
        assignmentModes: controlBlock === 'assignments' ? ['execution', 'control'] : ['execution'],
        acknowledgmentModes: controlBlock === 'acknowledgments' ? ['execution', 'control'] : ['execution'],
    }));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    expect(await screen.findByText('Поручения')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith(
        controlBlock === 'assignments' ? 'control' : '', controlBlock === 'acknowledgments' ? 'control' : '',
    ));
    const visibleTitle = controlBlock === 'assignments' ? 'Поручения' : 'Ознакомления';
    const hiddenTitle = controlBlock === 'assignments' ? 'Ознакомления' : 'Поручения';
    await waitFor(() => expect(screen.getByText(visibleTitle)).toBeInTheDocument());
    expect(screen.queryByText(hiddenTitle)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'исполнение' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith('execution', 'execution'));
    expect(await screen.findByText(hiddenTitle)).toBeInTheDocument();
});

test('personal events show only the preview and open documents without a full list drawer', async () => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    const event = {
        id: 'event-1', documentId: 'event-document', documentKind: 'incoming_letter', documentNumber: '238',
        documentDate: '2026-09-24T00:00:00', createdAt: '2026-09-28T12:47:00',
        eventType: 'assignment_created', entityType: 'assignment', title: 'Новое поручение', message: 'Вам назначено поручение',
    };
    api.GetCurrentUserEvents.mockResolvedValue({ items: [event], totalCount: 21 });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    const card = (await screen.findByText('Новое для меня')).closest('.ant-card') as HTMLElement;
    await within(card).findByText('Входящий № 238 от 24.09.2026');
    expect(card).toHaveClass('workspace-events-panel');
    expect(within(card).queryByRole('button', { name: /Все/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.GetCurrentUserEvents).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 5 }));
    expect(api.GetCurrentUserEvents.mock.calls.every(([filter]) => filter.page === 1 && filter.pageSize === 5)).toBe(true);
    fireEvent.click(within(card).getByRole('button', { name: /Вам назначено поручение/ }));
    expect(screen.getByTestId('opened-document')).toHaveTextContent('event-document');
});

test('recent documents stay below events and do not change with execution/control mode', async () => {
    setUser();
    api.GetOverview.mockImplementation(async (assignmentMode: string, acknowledgmentMode: string) => overview(assignmentMode, acknowledgmentMode));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    api.GetRecentDocuments.mockResolvedValue({ available: true, items: [{
        id: 'latest-document', documentKind: 'incoming_letter', documentNumber: '125',
        documentDate: '2026-09-20', registeredAt: '2026-09-24T12:00:00', description: 'Alpha', correspondents: [],
    }] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    await screen.findByText('Вх. № 125 от 20.09.2026');
    const card = screen.getByText('Последние документы').closest('.ant-card') as HTMLElement;
    const events = screen.getByText('Новое для меня').closest('.ant-card');
    expect(events?.nextElementSibling).toBe(card);
    expect(within(card).queryByRole('button', { name: /Все/ })).not.toBeInTheDocument();
    fireEvent.click(within(card).getByRole('button', { name: /Вх. № 125/ }));
    expect(screen.getByTestId('opened-document')).toHaveTextContent('latest-document');
    const requests = api.GetRecentDocuments.mock.calls.length;
    fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(api.GetOverview).toHaveBeenLastCalledWith('control', 'control'));
    expect(api.GetRecentDocuments).toHaveBeenCalledTimes(requests);
    expect(screen.getByText('Вх. № 125 от 20.09.2026')).toBeInTheDocument();
});

test('changing user closes the document and ignores the former user refresh response', async () => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    const item = { id: 'old-doc', documentKind: 'incoming_letter', documentNumber: 'OLD', documentDate: '2026-09-20', registeredAt: '2026-09-24T12:00:00', description: 'Old caption', correspondents: [] };
    const oldRefresh = deferred<{ available: boolean; items: typeof item[] }>();
    let first = true;
    api.GetRecentDocuments.mockImplementation(async () => {
        if (useAuthStore.getState().user?.id !== 'workspace-user') {
            return { available: true, items: [{ ...item, id: 'new-doc', documentNumber: 'NEW', description: 'New caption' }] };
        }
        if (first) { first = false; return { available: true, items: [item] }; }
        return oldRefresh.promise;
    });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    await screen.findByText('Вх. № OLD от 20.09.2026');
    fireEvent.click(screen.getByRole('button', { name: /Вх. № OLD/ }));
    expect(screen.getByTestId('opened-document')).toHaveTextContent('old-doc');
    await act(async () => { api.serverListeners.forEach((listener) => listener({ topic: 'documents' })); });
    await waitFor(() => expect(api.GetRecentDocuments).toHaveBeenCalledTimes(2));
    act(() => {
        const user = useAuthStore.getState().user!;
        useAuthStore.setState({ user: { ...user, id: 'another-user' } });
    });
    await screen.findByText('Вх. № NEW от 20.09.2026');
    await act(async () => { oldRefresh.resolve({ available: true, items: [item] }); });
    expect(screen.queryByText('Вх. № OLD от 20.09.2026')).not.toBeInTheDocument();
    expect(screen.queryByTestId('opened-document')).not.toBeInTheDocument();
});

test.each([
    ['user-events', undefined, 1, 0, 0],
    ['document-changed', 'files', 0, 0, 0],
    ['document-changed', 'links', 0, 0, 0],
    ['document-changed', 'assignments', 0, 1, 0],
    ['document-changed', 'acknowledgments', 0, 1, 0],
    ['document-changed', 'document', 0, 1, 1],
] as const)('%s %s refreshes only affected workspace data', async (topic, resource, eventDelta, overviewDelta, documentDelta) => {
    setUser();
    api.GetOverview.mockResolvedValue(overview('', ''));
    api.GetCurrentUserEvents.mockResolvedValue({ items: [] });
    renderWithApp(<DashboardPage onOpenAssignments={vi.fn()} onOpenRegister={vi.fn()} />);
    await screen.findByText('Поручения');
    await waitFor(() => expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(1));
    const before = [api.GetCurrentUserEvents.mock.calls.length, api.GetOverview.mock.calls.length, api.GetRecentDocuments.mock.calls.length];
    await act(async () => { api.serverListeners.forEach((listener) => listener({ topic, resource, documentId: 'doc-1' })); });
    expect(api.GetCurrentUserEvents).toHaveBeenCalledTimes(before[0] + eventDelta);
    expect(api.GetOverview).toHaveBeenCalledTimes(before[1] + overviewDelta);
    expect(api.GetRecentDocuments).toHaveBeenCalledTimes(before[2] + documentDelta);
});
