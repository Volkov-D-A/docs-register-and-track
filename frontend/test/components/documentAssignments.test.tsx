import React from 'react';
import { App } from 'antd';
import { act, fireEvent, renderHook, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { useAuthStore } from '../../src/store/useAuthStore';
import { useAssignments } from '../../src/hooks/useAssignments';
import DocumentAssignmentWorkflowPanel from '../../src/components/DocumentAssignmentWorkflowPanel';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({
  useDocumentKindAccess: () => ({ ready: true, hasAction: () => false }),
}));
vi.mock('../../src/components/AssignmentCompletionModal', () => ({ default: () => null }));

const wrapper = ({ children }: { children: React.ReactNode }) => <App>{children}</App>;
const assignments = (start: number, count: number) => Array.from({ length: count }, (_, index) => ({
  id: `assignment-${start + index}`, documentId: 'doc-1', status: 'finished', canAct: false,
  content: `Поручение ${start + index}`,
}));
const page = (items: ReturnType<typeof assignments>, totalCount: number, pageNumber = 1) => ({
  items, totalCount, page: pageNumber, pageSize: 100, hasMore: false,
});
const setup = (getList: ReturnType<typeof vi.fn>) => installWailsMock({ AssignmentService: { GetList: getList } });

test('shows an executor action beyond the first 100 assignments', async () => {
  const actionable = { ...assignments(101, 1)[0], status: 'in_progress', canAct: true };
  const getList = vi.fn().mockResolvedValueOnce(page(assignments(1, 100), 101))
    .mockResolvedValueOnce(page([actionable], 101, 2));
  setup(getList);
  renderWithApp(<DocumentAssignmentWorkflowPanel documentId="doc-1" documentKind="incoming_letter" />);
  expect(await screen.findByRole('button', { name: 'Исполнить' })).toBeInTheDocument();
  expect(screen.getByText('Поручение 101')).toBeInTheDocument();
  expect(getList).toHaveBeenNthCalledWith(2, expect.objectContaining({
    documentId: 'doc-1', page: 2, pageSize: 100, showFinished: true,
  }));
  expect(getList).toHaveBeenCalledTimes(2);
});

test('loads every page and retains all assignments', async () => {
  const getList = vi.fn().mockImplementation(({ page: number }) => Promise.resolve(
    page(assignments((number - 1) * 100 + 1, number === 3 ? 1 : 100), 201, number),
  ));
  setup(getList);
  const { result } = renderHook(() => useAssignments({ documentId: 'doc-1', documentKind: 'incoming_letter' }), { wrapper });
  await waitFor(() => expect(result.current.data).toHaveLength(201));
  expect(result.current.loadWarning).toBeNull();
  expect(result.current.loading).toBe(false);
  expect(getList).toHaveBeenCalledTimes(3);
});

test('shows the safety limit warning even if no loaded assignment has actions', async () => {
  const getList = vi.fn().mockImplementation(({ page: number }) => Promise.resolve(
    page(assignments((number - 1) * 100 + 1, 100), 1001, number),
  ));
  setup(getList);
  renderWithApp(<DocumentAssignmentWorkflowPanel documentId="doc-1" documentKind="incoming_letter" />);
  expect(await screen.findByText(/Список поручений неполный.*1000/)).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Обновить' })).toBeInTheDocument();
  expect(getList).toHaveBeenCalledTimes(10);
});

test('preserves loaded pages after a failure and clears the warning after retry', async () => {
  const getList = vi.fn().mockResolvedValueOnce(page(assignments(1, 100), 101))
    .mockRejectedValueOnce(new Error('second page unavailable'))
    .mockResolvedValueOnce(page(assignments(1, 100), 101))
    .mockResolvedValueOnce(page(assignments(101, 1), 101, 2));
  setup(getList);
  const { result } = renderHook(() => useAssignments({ documentId: 'doc-1', documentKind: 'incoming_letter' }), { wrapper });
  await waitFor(() => expect(result.current.loadWarning).toContain('Список поручений неполный'));
  expect(result.current.data).toHaveLength(100);
  await act(async () => { await result.current.load(); });
  expect(result.current.data).toHaveLength(101);
  expect(result.current.loadWarning).toBeNull();
});

test('stops on a repeated page without duplicating assignments', async () => {
  const getList = vi.fn().mockResolvedValue(page(assignments(1, 100), 101));
  setup(getList);
  const { result } = renderHook(() => useAssignments({ documentId: 'doc-1', documentKind: 'incoming_letter' }), { wrapper });
  await waitFor(() => expect(result.current.loadWarning).toContain('не содержит новых записей'));
  expect(result.current.data).toHaveLength(100);
  expect(getList).toHaveBeenCalledTimes(2);
});

test('discards a previous document and stops requesting its remaining pages', async () => {
  const pending = deferred<ReturnType<typeof page>>();
  const getList = vi.fn().mockReturnValueOnce(pending.promise).mockResolvedValue(page([], 0));
  setup(getList);
  const { result, rerender } = renderHook(({ documentId }) => useAssignments({ documentId, documentKind: 'incoming_letter' }), {
    wrapper, initialProps: { documentId: 'doc-1' },
  });
  await waitFor(() => expect(getList).toHaveBeenCalledTimes(1));
  rerender({ documentId: 'doc-2' });
  await act(async () => { pending.resolve(page(assignments(1, 100), 101)); });
  await waitFor(() => expect(result.current.loading).toBe(false));
  expect(result.current.data).toEqual([]);
  expect(result.current.loadWarning).toBeNull();
  expect(getList.mock.calls.map(([filter]) => [filter.documentId, filter.page])).toEqual([['doc-1', 1], ['doc-2', 1]]);
});

test('marks results incomplete if the total changes between pages', async () => {
  const getList = vi.fn().mockResolvedValueOnce(page(assignments(1, 100), 101))
    .mockResolvedValueOnce(page(assignments(101, 1), 102, 2));
  setup(getList);
  const { result } = renderHook(() => useAssignments({ documentId: 'doc-1', documentKind: 'incoming_letter' }), { wrapper });
  await waitFor(() => expect(result.current.loadWarning).toContain('Список изменился'));
  expect(result.current.data).toHaveLength(101);
  expect(getList).toHaveBeenCalledTimes(2);
});

test.each([false, true])('SSE refresh keeps the workflow layout stable (actionable: %s)', async (actionable) => {
  const previous = useAuthStore.getState();
  useAuthStore.setState({ isAuthenticated: true, sessionRevision: 7 });
  const item = { ...assignments(1, 1)[0], status: actionable ? 'in_progress' : 'finished', canAct: actionable };
  const pending = deferred<ReturnType<typeof page>>();
  const getList = vi.fn().mockResolvedValueOnce(page([item], 1)).mockReturnValueOnce(pending.promise);
  setup(getList);
  const view = renderWithApp(<DocumentAssignmentWorkflowPanel documentId="doc-1" documentKind="incoming_letter" />);
  try {
    await waitFor(() => expect(getList).toHaveBeenCalledTimes(1));
    if (actionable) await screen.findByRole('button', { name: 'Исполнить' });
    else await waitFor(() => expect(view.container.querySelector('.document-assignment-workflow')).toBeNull());
    const original = view.container.querySelector('.document-assignment-workflow');
    const row = view.container.querySelector('.document-assignment-workflow__item');
    act(() => fireEvent(window, new CustomEvent('server:event', { detail: {
      topic: 'document-changed', documentId: 'doc-1', resource: 'assignments', revision: 7,
    } })));
    await waitFor(() => expect(getList).toHaveBeenCalledTimes(2));
    expect(view.container.querySelector('.document-assignment-workflow')).toBe(original);
    expect(view.container.querySelector('.document-assignment-workflow__item')).toBe(row);
    expect(view.container.querySelector('.ant-spin')).toBeNull();
    if (actionable) expect(screen.getByRole('button', { name: 'Исполнить' })).toBeInTheDocument();
    await act(async () => { pending.resolve(page([item], 1)); });
  } finally { view.unmount(); useAuthStore.setState(previous); }
});


test('acknowledgment confirms directly without an execution report or acceptance', async () => {
  const item = { ...assignments(1, 1)[0], type: 'acknowledgment', status: 'new', canAct: true, content: 'Старый комментарий' };
  const getList = vi.fn().mockResolvedValueOnce(page([item], 1)).mockResolvedValue(page([{ ...item, canAct: false }], 1));
  const updateStatus = vi.fn().mockResolvedValue({ ...item, canAct: false });
  installWailsMock({ AssignmentService: { GetList: getList, UpdateStatus: updateStatus } });
  renderWithApp(<DocumentAssignmentWorkflowPanel documentId="doc-1" documentKind="incoming_letter" />);
  await screen.findByText('Ознакомиться с документом');
  expect(screen.queryByText('Старый комментарий')).not.toBeInTheDocument();
  fireEvent.click(await screen.findByRole('button', { name: 'Ознакомлен' }));
  await waitFor(() => expect(updateStatus).toHaveBeenCalledWith(item.id, 'finished', ''));
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Ознакомлен' })).not.toBeInTheDocument());
  expect(screen.queryByRole('button', { name: 'Взять в работу' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Исполнить' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Завершить' })).not.toBeInTheDocument();
  expect(getList).toHaveBeenCalledWith(expect.objectContaining({ types: [] }));
});
