import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, waitFor, renderHook } from '@testing-library/react';
import { useDocumentListPage } from '../../src/hooks/useDocumentListPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { installWailsMock } from '../componentTestUtils';

const api = { documents: vi.fn() };
let previousSession: ReturnType<typeof useAuthStore.getState>;

beforeEach(() => {
    previousSession = useAuthStore.getState();
    useAuthStore.setState({ isAuthenticated: true, sessionRevision: 5 });
    api.documents.mockResolvedValue({ items: [], hasMore: false });
    installWailsMock({
        DocumentQueryService: { GetList: api.documents },
    });
});
afterEach(() => {
    useAuthStore.setState(previousSession);
    for (const mock of Object.values(api)) mock.mockReset();
});

test('register refreshes only its document kind and keeps the opened document', async () => {
    const { result } = renderHook(() => useDocumentListPage({
        kindCode: 'incoming_letter', filters: {}, deps: [], buildFilter: () => ({}),
    }));
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(1));
    act(() => result.current.openViewModal('doc-1'));
    act(() => fireEvent(window, new CustomEvent('server:event', { detail: {
        topic: 'document-changed', documentId: 'doc-2', documentKind: 'outgoing_letter', resource: 'document', revision: 5,
    } })));
    act(() => fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'documents', revision: 5 } })));
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 150)); });
    expect(api.documents).toHaveBeenCalledTimes(1);
    act(() => fireEvent(window, new CustomEvent('server:event', { detail: {
        topic: 'document-changed', documentId: 'doc-3', documentKind: 'incoming_letter', resource: 'document', revision: 5,
    } })));
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(2));
    expect(result.current.viewModalOpen).toBe(true);
    expect(result.current.viewDocId).toBe('doc-1');
});

test('register follows cursor history and resets it when page size changes', async () => {
    api.documents.mockResolvedValue({ items: [], hasMore: true, nextCursor: 'cursor-2' });
    const { result } = renderHook(() => useDocumentListPage({
        kindCode: 'incoming_letter', filters: {}, deps: [],
        buildFilter: ({ pageSize, search }) => ({ pageSize, search }),
    }));
    await waitFor(() => expect(result.current.hasMore).toBe(true));
    expect(api.documents.mock.calls[0]).toEqual(['incoming_letter', { pageSize: 10, search: '', cursor: '' }]);
    act(() => result.current.goToNextPage());
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(2));
    expect(api.documents.mock.calls[1][1].cursor).toBe('cursor-2');
    expect(result.current.page).toBe(2);
    act(() => result.current.goToPreviousPage());
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(3));
    expect(api.documents.mock.calls[2][1].cursor).toBe('');
    act(() => result.current.goToNextPage());
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(4));
    act(() => result.current.setPageSize(20));
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(5));
    expect(api.documents.mock.calls[4][1]).toEqual({ pageSize: 20, search: '', cursor: '' });
    expect(result.current.page).toBe(1);
    expect(result.current.canGoBack).toBe(false);
});
