import React from 'react';
import dayjs from 'dayjs';
import { afterEach, expect, test, vi } from 'vitest';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import AssignmentsPage from '../../src/pages/AssignmentsPage';
import DashboardPage from '../../src/pages/DashboardPage';
import { useAssignmentModeStore } from '../../src/store/useAssignmentModeStore';
import { useAuthStore } from '../../src/store/useAuthStore';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({ useDocumentKindAccess: () => ({
    ready: true, hasAction: () => true, hasAnyAction: (action: string) => action === 'assign',
}) }));
vi.mock('../../src/hooks/useCurrentAccessSummary', () => ({ useCurrentAccessSummary: () => ({ ready: true, registrationKinds: [] }) }));
vi.mock('../../src/components/WorkspaceUserEventsPanel', () => ({ default: () => null }));
vi.mock('../../src/components/RecentDocumentsPanel', () => ({ default: () => null }));
vi.mock('../../src/components/DocumentSearchPanel', () => ({ default: () => null }));
afterEach(() => { useAuthStore.setState({ user: null, isAuthenticated: false }); });

vi.mock('../../src/components/DocumentViewModal', () => ({ default: () => null }));
vi.mock('../../src/components/AssignmentModal', () => ({ default: () => null }));
vi.mock('../../src/components/AssignmentSeriesModal', () => ({ default: () => null }));

test.each([
    ['new', 'new', 'Новые'],
    ['in_progress', 'in_progress', 'В работе'],
    ['returned', 'returned', 'На доработке'],
    ['acceptance', 'completed', 'На приёмке'],
] as const)('%s counter populates the visible status filter and can be cleared', async (metric, status, label) => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'user-1', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'] }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control', metric }} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'control', statuses: [status], types: [] }));
    expect(getList.mock.lastCall?.[0]).not.toHaveProperty('metric');
    expect(within(screen.getByRole('group', { name: 'Статус поручения' })).getByRole('button', { name: label })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByRole('button', { name: 'Сбросить фильтры' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ statuses: [], overdueOnly: false, dateFrom: '', dateTo: '' }));
});

test.each(['overdue', 'due_soon'] as const)('%s counter populates visible deadline filters that remain editable', async (metric) => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'user-1', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'] }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control', metric }} />);
    await waitFor(() => expect(getList).toHaveBeenCalled());
    expect(getList.mock.lastCall?.[0]).not.toHaveProperty('metric');
    const today = dayjs().format('YYYY-MM-DD');
    const dateTo = dayjs().add(3, 'day').format('YYYY-MM-DD');
    if (metric === 'overdue') {
        expect(getList.mock.lastCall?.[0]).toMatchObject({ overdueOnly: true, dateFrom: '', dateTo: '' });
        expect(screen.getByRole('button', { name: 'Просроченные' })).toHaveAttribute('aria-pressed', 'true');
    } else {
        expect(getList.mock.lastCall?.[0]).toMatchObject({ overdueOnly: false, dateFrom: today, dateTo });
        expect(screen.getByRole('button', { name: 'До 3 дней' })).toHaveAttribute('aria-pressed', 'true');
    }
    fireEvent.click(screen.getByRole('button', { name: 'Сегодня' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ overdueOnly: false, dateFrom: today, dateTo: today }));
    fireEvent.click(screen.getByRole('button', { name: 'Сбросить фильтры' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ overdueOnly: false, dateFrom: '', dateTo: '' }));
});

test('control-only user opens the server-selected list mode', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'controller', login: 'controller', fullName: 'Controller', isDocumentParticipant: false, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({
        WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['control'], assignmentMode: 'control' }) },
        AssignmentService: { GetList: getList },
        UserService: { GetExecutors: vi.fn().mockResolvedValue([]) },
    });
    renderWithApp(<AssignmentsPage initialView={null} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'control' }));
    expect(screen.getByText('Поручения под контролем')).toBeInTheDocument();
    expect(screen.queryByRole('radio', { name: 'Исполнение' })).not.toBeInTheDocument();
});


test.each(['execution', 'control'] as const)('acknowledgment navigation selects the recipient task type in %s mode', async (mode) => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'user-1', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({
        WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'] }) },
        AssignmentService: { GetList: getList },
        UserService: { GetExecutors: vi.fn().mockResolvedValue([]) },
    });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode, type: 'acknowledgment' }} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode, types: ['acknowledgment'] }));
});


test('work mode survives navigation between dashboard and assignments in both directions', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'shared-mode-user', login: 'tester', fullName: 'Tester', firstName: 'Tester', patronymic: '', noPatronymic: true, isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    const getOverview = vi.fn().mockImplementation(async (mode: string) => ({
        assignmentModes: ['execution', 'control'], assignmentMode: mode || 'execution',
        assignmentCounts: {}, assignments: [],
    }));
    installWailsMock({ WorkspaceService: { GetOverview: getOverview }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    const openAssignments = vi.fn();
    const dashboard = renderWithApp(<DashboardPage onOpenAssignments={openAssignments} onOpenRegister={vi.fn()} />);
    await screen.findByText('Поручения');
    fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
    await waitFor(() => expect(getOverview).toHaveBeenLastCalledWith('control'));
    fireEvent.click(screen.getByRole('button', { name: 'Все поручения' }));
    expect(openAssignments).toHaveBeenCalledWith('control');
    dashboard.unmount();

    const assignments = renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control' }} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'control' }));
    fireEvent.click(screen.getByRole('radio', { name: 'Исполнение' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'execution' }));
    assignments.unmount();

    const returnedDashboard = renderWithApp(<DashboardPage onOpenAssignments={openAssignments} onOpenRegister={vi.fn()} />);
    await waitFor(() => expect(getOverview).toHaveBeenLastCalledWith('execution'));
    expect(await screen.findByRole('button', { name: 'исполнение' })).toHaveAttribute('aria-pressed', 'true');
    returnedDashboard.unmount();

    // An earlier dashboard navigation must not overwrite a later selection on remount.
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control' }} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'execution' }));
    expect(screen.getByText('Мои поручения')).toBeInTheDocument();
});

test('an unavailable saved mode falls back to an allowed mode', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'changed-access-user', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    useAssignmentModeStore.getState().setMode('control');
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution'], assignmentMode: 'execution' }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control' }} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'execution' }));
    expect(useAssignmentModeStore.getState().mode).toBe('execution');
    expect(getList.mock.calls.every(([filter]) => filter.mode === 'execution')).toBe(true);
});

test('work mode survives profile updates and resets on sign-out or account change', () => {
    const user = { id: 'first-user', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [] };
    useAuthStore.setState({ user, isAuthenticated: true });
    useAssignmentModeStore.getState().setMode('control');
    useAuthStore.setState({ user: { ...user, fullName: 'Updated' } });
    expect(useAssignmentModeStore.getState().mode).toBe('control');
    useAuthStore.setState({ user: { ...user, id: 'second-user' } });
    expect(useAssignmentModeStore.getState().mode).toBe('');
    useAssignmentModeStore.getState().setMode('control');
    useAuthStore.setState({ user: null, isAuthenticated: false });
    expect(useAssignmentModeStore.getState().mode).toBe('');
});


test('filter buttons combine status and type and reset also clears search and finished tasks', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'filter-user', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'] }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control' }} />);
    await waitFor(() => expect(getList).toHaveBeenCalled());
    expect(screen.queryByRole('combobox', { name: 'Статус поручения' })).not.toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: 'Тип поручения' })).not.toBeInTheDocument();
    fireEvent.click(within(screen.getByRole('group', { name: 'Статус поручения' })).getByRole('button', { name: 'Новые' }));
    fireEvent.click(within(screen.getByRole('group', { name: 'Тип поручения' })).getByRole('button', { name: 'Ознакомление' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ statuses: ['new'], types: ['acknowledgment'] }));
    const search = screen.getByRole('searchbox', { name: 'Поиск поручений' });
    fireEvent.change(search, { target: { value: 'письмо' } });
    fireEvent.keyDown(search, { key: 'Enter', code: 'Enter', charCode: 13 });
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ search: 'письмо' }));
    expect(screen.getByRole('button', { name: 'Показать завершённые' })).toHaveAttribute('aria-pressed', 'false');
    fireEvent.click(screen.getByRole('button', { name: 'Показать завершённые' }));
    expect(screen.getByRole('button', { name: 'Показать завершённые' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.queryByRole('button', { name: 'Завершённые' })).not.toBeInTheDocument();
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ statuses: ['new'], showFinished: true }));
    fireEvent.click(screen.getByRole('button', { name: 'Показать завершённые' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ statuses: ['new'], showFinished: false }));
    fireEvent.click(screen.getByRole('button', { name: 'Сбросить фильтры' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ statuses: [], types: [], search: '', showFinished: false }));
    expect(search).toHaveValue('');
    expect(screen.queryByRole('button', { name: 'Завершённые' })).not.toBeInTheDocument();
});

test('status and type toggle off independently while preserving search and deadline filters', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'toggle-user', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'] }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control', metric: 'new', type: 'acknowledgment' }} />);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ statuses: ['new'], types: ['acknowledgment'] }));
    const search = screen.getByRole('searchbox', { name: 'Поиск поручений' });
    fireEvent.change(search, { target: { value: 'письмо' } });
    fireEvent.keyDown(search, { key: 'Enter', code: 'Enter', charCode: 13 });
    fireEvent.click(screen.getByRole('button', { name: 'Сегодня' }));
    const retained = { search: 'письмо', dateFrom: dayjs().format('YYYY-MM-DD'), dateTo: dayjs().format('YYYY-MM-DD') };
    const status = within(screen.getByRole('group', { name: 'Статус поручения' })).getByRole('button', { name: 'Новые' });
    const type = within(screen.getByRole('group', { name: 'Тип поручения' })).getByRole('button', { name: 'Ознакомление' });
    fireEvent.click(status);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ ...retained, statuses: [], types: ['acknowledgment'] }));
    expect(status).toHaveAttribute('aria-pressed', 'false');
    expect(type).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(type);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ ...retained, statuses: [], types: [] }));
    expect(type).toHaveAttribute('aria-pressed', 'false');
    fireEvent.click(status);
    fireEvent.click(type);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ ...retained, statuses: ['new'], types: ['acknowledgment'] }));
    fireEvent.click(within(screen.getByRole('group', { name: 'Статус поручения' })).getByRole('button', { name: 'В работе' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ ...retained, statuses: ['new', 'in_progress'], types: ['acknowledgment'] }));
    expect(status).toHaveAttribute('aria-pressed', 'true');
    const execution = within(screen.getByRole('group', { name: 'Тип поручения' })).getByRole('button', { name: 'Исполнение' });
    fireEvent.click(execution);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ ...retained, statuses: ['new', 'in_progress'], types: ['acknowledgment', 'execution'] }));
    expect(type).toHaveAttribute('aria-pressed', 'true');
    expect(execution).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(status);
    fireEvent.click(type);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ ...retained, statuses: ['in_progress'], types: ['execution'] }));
});

test('custom deadline period opens a calendar and shows the chosen dates on its button', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'period-user', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'] }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control' }} />);
    await waitFor(() => expect(getList).toHaveBeenCalled());
    expect(screen.queryByPlaceholderText('Срок от')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Выбрать период' }));
    expect(await screen.findByPlaceholderText('Срок от')).toBeInTheDocument();
    const start = dayjs().add(1, 'month').startOf('month').add(9, 'day');
    const end = start.add(5, 'day');
    fireEvent.click((await screen.findAllByTitle(start.format('YYYY-MM-DD'))).find((cell) => cell.classList.contains('ant-picker-cell-in-view'))!);
    fireEvent.click((await screen.findAllByTitle(end.format('YYYY-MM-DD'))).find((cell) => cell.classList.contains('ant-picker-cell-in-view'))!);
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ dateFrom: start.format('YYYY-MM-DD'), dateTo: end.format('YYYY-MM-DD'), overdueOnly: false }));
    expect(screen.getByRole('button', { name: `${start.format('DD.MM.YYYY')} — ${end.format('DD.MM.YYYY')}` })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.queryByPlaceholderText('Срок от')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Любой' }));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ dateFrom: '', dateTo: '', overdueOnly: false }));
});

test.each([1, 12])('acknowledgment rows display %i recipients compactly and retain expanded details', async (count) => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'reader', login: 'reader', fullName: 'Reader', isDocumentParticipant: true, systemPermissions: [],
    } });
    const users = Array.from({ length: count }, (_, index) => ({
        userId: `reader-${index}`, userName: `Адресат ${index + 1}`,
        confirmedAt: index === 0 ? '2026-09-29T10:00:00Z' : undefined,
    }));
    const getList = vi.fn().mockResolvedValue({ items: [{
        id: 'read-task', documentId: 'doc-1', documentKind: 'incoming_letter',
        type: 'acknowledgment', status: 'new', createdAt: '2026-09-29', users,
    }], totalCount: 1 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution'] }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={null} />);
    expect(await screen.findByText(count === 1 ? 'Адресат 1' : '12 адресатов')).toBeInTheDocument();
    expect(screen.queryByText(/Адресат 1:/)).not.toBeInTheDocument();
    expect(screen.getByText(`Ожидает (1/${count})`)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Expand row' }));
    expect(screen.getByText(/Адресат 1: ознакомлен/)).toBeInTheDocument();
    if (count > 1) expect(screen.getByText('Адресат 12: ожидает ознакомления')).toBeInTheDocument();
});
