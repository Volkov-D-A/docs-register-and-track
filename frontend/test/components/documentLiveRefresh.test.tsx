import React from 'react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, renderHook } from '@testing-library/react';
import { useDocumentListPage } from '../../src/hooks/useDocumentListPage';
import DocumentViewModal from '../../src/components/DocumentViewModal';
import { useAuthStore } from '../../src/store/useAuthStore';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({
    useDocumentKindAccess: () => ({ ready: true, loading: false, kinds: [], hasAction: (_kind: string, action: string) => action === 'assign' }),
}));
vi.mock('../../src/components/DocumentAssignmentWorkflowPanel', () => ({ default: () => null }));
vi.mock('../../src/components/DocumentAcknowledgmentWorkflowPanel', () => ({ default: () => null }));
vi.mock('../../src/components/documentDetails/IncomingDocumentDetails', () => ({ default: () => <div>Карточка документа</div> }));
vi.mock('../../src/components/FileListComponent', () => ({ default: () => null }));
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
        AssignmentService: { GetList: api.list },
        UserService: { GetExecutors: api.executors },
        UserEventService: { MarkDocumentRead: vi.fn().mockResolvedValue(undefined) },
    });
});
afterEach(() => {
    useAuthStore.setState(previousSession);
    for (const mock of Object.values(api)) mock.mockReset();
});

const emit = (topic: string, documentId = 'doc-1', resource = 'document', visibilityChanged = false) => {
    fireEvent(window, new CustomEvent('server:event', { detail: { topic, documentId, resource, visibilityChanged, revision: 5 } }));
};

const openForm = async () => {
    await screen.findByText('Карточка документа');
    fireEvent.click(screen.getByRole('tab', { name: 'Поручения' }));
    fireEvent.click(await screen.findByRole('button', { name: /Добавить поручение/ }));
    const input = await screen.findByRole('textbox', { name: 'Текст поручения' });
    fireEvent.change(input, { target: { value: 'Несохранённое поручение' } });
    return input;
};

test.each([['document-changed', 'document', false], ['resync', 'document', false], ['document-changed', 'assignments', true]] as const)('%s %s preserves layout and the open assignment form throughout background loading', async (topic, resource, visibilityChanged) => {
    const close = vi.fn();
    const view = renderWithApp(<DocumentViewModal open documentId="doc-1" documentKind="incoming_letter" onCancel={() => close()} />);
    const input = await openForm();
    // A dashboard rerender creates a new callback but must not reload the document.
    view.rerender(<DocumentViewModal open documentId="doc-1" documentKind="incoming_letter" onCancel={() => close()} />);
    expect(api.get).toHaveBeenCalledTimes(1);
    const pending = deferred<typeof card>();
    api.get.mockReturnValueOnce(pending.promise);
    act(() => emit(topic, 'doc-1', resource, visibilityChanged));
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2));
    const body = screen.getByRole('tab', { name: 'Поручения' }).closest('.ant-modal-body')!;
    expect(body.querySelector(':scope > .ant-spin')).toBeNull();
    expect(body.querySelector('.ant-table-wrapper .ant-spin-spinning')).toBeNull();
    expect(input).toBeInTheDocument();
    expect(input).toHaveValue('Несохранённое поручение');
    expect(screen.getByRole('button', { name: 'Создать' })).toBeInTheDocument();
    await act(async () => { pending.resolve(card); });
    expect(screen.getByRole('textbox', { name: 'Текст поручения' })).toBe(input);
    expect(input).toHaveValue('Несохранённое поручение');
    expect(close).not.toHaveBeenCalled();
});

test('ignores another document and unrelated resources, refreshes only the assignment list', async () => {
    renderWithApp(<DocumentViewModal open documentId="doc-1" documentKind="incoming_letter" onCancel={vi.fn()} />);
    const input = await openForm();
    await waitFor(() => expect(api.list).toHaveBeenCalledTimes(1));
    act(() => { emit('document-changed', 'doc-2', 'assignments'); emit('document-changed', 'doc-1', 'files'); });
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 150)); });
    expect(api.get).toHaveBeenCalledTimes(1);
    expect(api.list).toHaveBeenCalledTimes(1);
    act(() => { emit('document-changed', 'doc-1', 'assignments'); emit('document-changed', 'doc-1', 'assignments'); });
    await waitFor(() => expect(api.list).toHaveBeenCalledTimes(2));
    expect(api.get).toHaveBeenCalledTimes(1);
    expect(input).toHaveValue('Несохранённое поручение');
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
