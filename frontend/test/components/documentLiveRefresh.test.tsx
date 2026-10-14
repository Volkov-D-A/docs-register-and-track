import React from 'react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, waitFor, renderHook } from '@testing-library/react';
import { useDocumentListPage } from '../../src/hooks/useDocumentListPage';
import { useAuthStore } from '../../src/store/useAuthStore';
import { installWailsMock } from '../componentTestUtils';

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({
    useDocumentKindAccess: () => ({ ready: true, loading: false, kinds: [], hasAction: (_kind: string, action: string) => action === 'assign' }),
}));
vi.mock('../../src/components/documentDetails/IncomingDocumentDetails', () => ({ default: () => <div>Карточка документа</div> }));
vi.mock('../../src/components/RelatedDocumentModal', () => ({ default: () => null }));

const card = { id: 'doc-1', kindCode: 'incoming_letter', registrationNumber: '1', incomingLetter: {} };
const api = { get: vi.fn(), list: vi.fn(), executors: vi.fn(), documents: vi.fn() };
let previousSession: ReturnType<typeof useAuthStore.getState>;

beforeEach(() => {
    previousSession = useAuthStore.getState();
    useAuthStore.setState({ isAuthenticated: true, sessionRevision: 5 });
    api.get.mockResolvedValue(card);
    api.list.mockResolvedValue({ items: [], totalCount: 0 });
    api.executors.mockResolvedValue([]);
    api.documents.mockResolvedValue({ items: [], hasMore: false });
    installWailsMock({
        DocumentQueryService: { GetByID: api.get, GetList: api.documents },
        UserService: { GetExecutors: api.executors },
        UserEventService: { MarkDocumentRead: vi.fn().mockResolvedValue(undefined) },
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
