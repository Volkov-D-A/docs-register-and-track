import React from 'react';
import { beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import DeadlineDashboardPage from '../../src/pages/DeadlineDashboardPage';
import { deadlinesChanged, organizationToday } from '../../src/features/deadlines/deadlineEvents';
import { renderWithApp } from '../componentTestUtils';
const api = vi.hoisted(() => ({ List: vi.fn(), ListExecutors: vi.fn(), listeners: [] as Array<(event: { topic: string; resource?: string }) => void> }));
vi.mock('../../wailsjs/go/services/DeadlineControlService', () => api);
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string; resource?: string }) => void) => { api.listeners.push(listener); return () => { api.listeners = api.listeners.filter((item) => item !== listener); }; } }));
vi.mock('../../src/hooks/useCurrentAccessSummary', () => ({ useCurrentAccessSummary: () => ({ hasAnyAction: () => true, registrationKinds: [] }) }));
vi.mock('../../src/components/RecentDocumentsPanel', () => ({ default: () => null }));
vi.mock('../../src/components/DocumentSearchPanel', () => ({ default: () => null }));
vi.mock('../../src/components/WorkspaceUserEventsPanel', () => ({ default: () => null }));
vi.mock('../../src/components/DocumentViewModal', () => ({ default: () => null }));
beforeEach(() => { api.ListExecutors.mockResolvedValue([]); });
test('dashboard navigation uses the same scope and dates as its counter', async () => {
  api.List.mockResolvedValue({ total: 3, items: [] });
  const open = vi.fn();
  renderWithApp(<DeadlineDashboardPage onOpenControls={open} onOpenRegister={vi.fn()} />);
  fireEvent.click(await screen.findByRole('button', { name: 'Сегодня' }));
  expect(open).toHaveBeenCalledWith({ mineOnly: true, manageOnly: false, status: 'active', dueFrom: organizationToday(), dueTo: organizationToday() });
  expect(api.List).toHaveBeenCalledWith(expect.objectContaining({ ...open.mock.calls[0][0], page: 1, pageSize: 1 }));
});

test('dashboard restores the active controls table and navigates to all active controls', async () => {
  api.List.mockResolvedValue({ total: 1, items: [{ id: 'control-1', documentId: 'doc-1', documentKind: 'incoming_letter', documentNumber: 'ВХ-42', subject: 'Подготовить ответ', dueDate: organizationToday() }] });
  const open = vi.fn();
  renderWithApp(<DeadlineDashboardPage onOpenControls={open} onOpenRegister={vi.fn()} />);
  expect(await screen.findByText('Подготовить ответ')).toBeInTheDocument();
  expect(screen.getByText('ВХ-42')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Все сроки' }));
  expect(open).toHaveBeenCalledWith({ mineOnly: true, manageOnly: false, status: 'active' });
  expect(api.List).toHaveBeenCalledWith(expect.objectContaining({ status: 'active', pageSize: 5 }));
});


test('stopped series metric and query are available only in control mode', async () => {
  api.List.mockClear();
  api.List.mockResolvedValue({ total: 0, items: [] });
  const open = vi.fn();
  renderWithApp(<DeadlineDashboardPage onOpenControls={open} onOpenRegister={vi.fn()} />);
  await waitFor(() => expect(api.List).toHaveBeenCalledTimes(5));
  expect(screen.queryByRole('button', { name: 'Остановленные серии' })).not.toBeInTheDocument();
  expect(api.List.mock.calls.some(([filter]) => filter.seriesStoppedOnly)).toBe(false);
  fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Остановленные серии' }));
  expect(open).toHaveBeenCalledWith({ mineOnly: false, manageOnly: true, status: '', seriesStoppedOnly: true });
  await waitFor(() => expect(api.List).toHaveBeenCalledWith(expect.objectContaining({ manageOnly: true, seriesStoppedOnly: true })));
});


test('control dashboard shows primary and coexecutors, my deadlines omit the column', async () => {
  api.ListExecutors.mockClear();
  api.List.mockResolvedValue({ total: 1, items: [{ id: 'control-1', executorId: 'person-1', coExecutorIds: ['person-2'], dueDate: organizationToday(), subject: 'Ответ' }] });
  api.ListExecutors.mockResolvedValue([{ id: 'person-1', name: 'Иванов Иван' }, { id: 'person-2', name: 'Петров Пётр' }]);
  renderWithApp(<DeadlineDashboardPage onOpenControls={vi.fn()} onOpenRegister={vi.fn()} />);
  await screen.findByText('Ответ');
  expect(screen.queryByRole('columnheader', { name: 'Исполнители' })).not.toBeInTheDocument();
  expect(api.ListExecutors).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
  expect(await screen.findByText('Иванов Иван')).toBeInTheDocument();
  expect(screen.getByText('Соисполнители: Петров Пётр')).toBeInTheDocument();
  expect(api.ListExecutors).toHaveBeenCalledWith(true);
  fireEvent.click(screen.getByRole('button', { name: 'мои сроки' }));
  expect(screen.queryByRole('columnheader', { name: 'Исполнители' })).not.toBeInTheDocument();
});


test('dashboard does not poll every minute and reloads on organization midnight', async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-10-19T18:57:00Z'));
  api.List.mockClear();
  api.List.mockResolvedValue({ total: 0, items: [] });
  let unmount = () => {};
  try {
    await act(async () => { ({ unmount } = renderWithApp(<DeadlineDashboardPage onOpenControls={vi.fn()} onOpenRegister={vi.fn()} />)); });
    expect(api.List).toHaveBeenCalledTimes(5);
    expect(screen.queryByRole('button', { name: 'Обновить' })).not.toBeInTheDocument();
    await act(async () => { vi.advanceTimersByTime(120_000); });
    expect(api.List).toHaveBeenCalledTimes(5);
    await act(async () => { vi.advanceTimersByTime(60_000); });
    expect(api.List).toHaveBeenCalledTimes(10);
    expect(api.List).toHaveBeenCalledWith(expect.objectContaining({ dueFrom: '2026-10-20', dueTo: '2026-10-20' }));
    await act(async () => { deadlinesChanged(); });
    expect(api.List).toHaveBeenCalledTimes(15);
    await act(async () => { api.listeners.forEach((listener) => listener({ topic: 'document-changed', resource: 'controls' })); vi.advanceTimersByTime(100); });
    expect(api.List).toHaveBeenCalledTimes(20);
    await act(async () => { api.listeners.forEach((listener) => listener({ topic: 'document-changed', resource: 'document' })); vi.advanceTimersByTime(100); });
    expect(api.List).toHaveBeenCalledTimes(25);
  } finally { unmount(); vi.useRealTimers(); }
});

test('dashboard offers retry only after a failed load', async () => {
  api.List.mockRejectedValueOnce(new Error('Нет связи')).mockResolvedValue({ total: 0, items: [] });
  renderWithApp(<DeadlineDashboardPage onOpenControls={vi.fn()} onOpenRegister={vi.fn()} />);
  fireEvent.click(await screen.findByRole('button', { name: 'Повторить' }));
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Повторить' })).not.toBeInTheDocument());
  expect(screen.queryByRole('button', { name: 'Обновить' })).not.toBeInTheDocument();
});
