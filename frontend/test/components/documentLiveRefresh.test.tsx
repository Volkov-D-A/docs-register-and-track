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
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 150)); });
    expect(api.documents).toHaveBeenCalledTimes(1);
    act(() => fireEvent(window, new CustomEvent('server:event', { detail: {
        topic: 'document-changed', documentId: 'doc-3', documentKind: 'incoming_letter', resource: 'document', revision: 5,
    } })));
    await waitFor(() => expect(api.documents).toHaveBeenCalledTimes(2));
    expect(result.current.viewModalOpen).toBe(true);
    expect(result.current.viewDocId).toBe('doc-1');
});
