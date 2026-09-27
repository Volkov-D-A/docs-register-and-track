import React from 'react';
import { expect, test, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import AssignmentsPage from '../../src/pages/AssignmentsPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({ useDocumentKindAccess: () => ({
    ready: true, hasAction: () => true, hasAnyAction: (action: string) => action === 'assign',
}) }));
vi.mock('../../src/components/DocumentViewModal', () => ({ default: () => null }));
vi.mock('../../src/components/AssignmentModal', () => ({ default: () => null }));
vi.mock('../../src/components/AssignmentSeriesModal', () => ({ default: () => null }));

test('assignment card opens a server-scoped list with the same mode and metric', async () => {
    useAuthStore.setState({ isAuthenticated: true, user: {
        id: 'user-1', login: 'tester', fullName: 'Tester', isDocumentParticipant: true, systemPermissions: [],
    } });
    const getList = vi.fn().mockResolvedValue({ items: [], totalCount: 0 });
    installWailsMock({ WorkspaceService: { GetOverview: vi.fn().mockResolvedValue({ assignmentModes: ['execution', 'control'], assignmentMode: 'execution' }) }, AssignmentService: { GetList: getList }, UserService: { GetExecutors: vi.fn().mockResolvedValue([]) } });
    renderWithApp(<AssignmentsPage initialView={{ requestId: 1, mode: 'control', metric: 'overdue' }} />);
    await waitFor(() => expect(getList).toHaveBeenCalled());
    expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'control', metric: 'overdue' });
    expect(screen.getByText('Поручения под контролем')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Исполнение'));
    await waitFor(() => expect(getList.mock.lastCall?.[0]).toMatchObject({ mode: 'execution', metric: '' }));
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
    expect(screen.queryByText('Исполнение')).not.toBeInTheDocument();
});
