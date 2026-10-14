import React from 'react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import DeadlineControls from '../../src/features/deadlines/DeadlineControls';
import { organizationToday } from '../../src/features/deadlines/deadlineEvents';
import { renderWithApp } from '../componentTestUtils';
const api = vi.hoisted(() => ({ List: vi.fn(), ListExecutors: vi.fn(), Complete: vi.fn(), listeners: [] as Array<(event: { topic: string }) => void> }));
vi.mock('../../wailsjs/go/services/DeadlineControlService', () => api);
vi.mock('../../src/hooks/useDocumentKindAccess', () => ({ useDocumentKindAccess: () => ({ ready: true, hasAction: () => false, hasAnyAction: () => false }) }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string }) => void) => { api.listeners.push(listener); return () => { api.listeners = api.listeners.filter((item) => item !== listener); }; } }));
const control = { id: 'control-1', documentId: 'doc-1', documentKind: 'incoming_letter', subject: 'Проверить ответ', executorId: 'person-1', coExecutorIds: [], status: 'active', dueDate: '2026-10-10', version: 7, canManage: true };
beforeEach(() => {
  api.List.mockResolvedValue({ items: [control], total: 1 });
  api.ListExecutors.mockResolvedValue([{ id: 'person-1', name: 'Исполнитель', isActive: true }]);
  api.Complete.mockResolvedValue({ ...control, status: 'completed' });
});
afterEach(() => { vi.clearAllMocks(); api.listeners = []; });
test('card readers see controls without management actions', async () => {
  api.List.mockResolvedValue({ items: [{ ...control, canManage: false }], total: 1 });
  renderWithApp(<DeadlineControls documentId="doc-1" documentKind="incoming_letter" />);
  expect(await screen.findByText('Проверить ответ')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Изменить' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Исполнен' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Установить контроль' })).not.toBeInTheDocument();
});
test('completion warns about finality and submits the edited actual date with the current version', async () => {
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Исполнен' }));
  expect(await screen.findByText('Завершение контроля нельзя отменить')).toBeInTheDocument();
  const input = screen.getByLabelText('Фактическая дата исполнения');
  expect(input).toHaveValue(organizationToday());
  expect(input).toHaveAttribute('max', organizationToday());
  expect(api.Complete).not.toHaveBeenCalled();
  fireEvent.change(input, { target: { value: '2026-01-01' } });
  fireEvent.click(screen.getByRole('button', { name: 'Завершить окончательно' }));
  await waitFor(() => expect(api.Complete).toHaveBeenCalledWith('control-1', expect.objectContaining({ version: 7, executedOn: '2026-01-01' })));
});
test('access invalidation clears cached controls while the replacement request is pending', async () => {
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  expect(await screen.findByText('Проверить ответ')).toBeInTheDocument();
  api.List.mockReturnValue(new Promise(() => {}));
  act(() => api.listeners.forEach((listener) => listener({ topic: 'resync' })));
  expect(screen.queryByText('Проверить ответ')).not.toBeInTheDocument();
  await waitFor(() => expect(api.List).toHaveBeenCalledTimes(2));
});
