import React from 'react';
import { afterEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import RecentDocumentsPanel from '../../src/components/RecentDocumentsPanel';
import { renderWithApp } from '../componentTestUtils';

const api = vi.hoisted(() => ({ get: vi.fn(), listeners: [] as Array<(event: { topic: string }) => void> }));
vi.mock('../../wailsjs/go/services/WorkspaceService', () => ({ GetRecentDocuments: api.get }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string }) => void) => {
    api.listeners.push(listener);
    return () => { api.listeners = api.listeners.filter((item) => item !== listener); };
} }));

afterEach(() => { api.get.mockReset(); api.listeners = []; });

const items = [
    { id: 'incoming', documentKind: 'incoming_letter', documentNumber: '125', documentDate: '2026-09-20', registeredAt: '2026-09-24T12:00:00', description: 'Alpha', correspondents: ['Alpha', 'Beta'] },
    { id: 'outgoing', documentKind: 'outgoing_letter', documentNumber: '84', documentDate: '2026-09-19', registeredAt: '2026-09-23T12:00:00', description: 'Администрация города', correspondents: [] },
    { id: 'order', documentKind: 'administrative_order', documentNumber: '41', documentDate: '2026-09-22', registeredAt: '2026-09-22T12:00:00', description: 'О назначении', correspondents: [] },
    { id: 'appeal', documentKind: 'citizen_appeal', documentNumber: '543', documentDate: '2026-09-21', registeredAt: '2026-09-21T12:00:00', description: 'Петров А.В.', correspondents: [] },
];

test('four document rows show registration dates separately, compact captions and open documents', async () => {
    api.get.mockResolvedValue({ available: true, items });
    const open = vi.fn();
    renderWithApp(<RecentDocumentsPanel refreshVersion={0} onOpenDocument={open} />);
    const list = await screen.findByRole('list', { name: 'Последние зарегистрированные документы' });
    const rows = within(list).getAllByRole('listitem');
    expect(rows).toHaveLength(4);
    expect(screen.getByText('Вх. № 125 от 20.09.2026')).toBeInTheDocument();
    expect(screen.getByText('24.09.2026')).toHaveAttribute('title', 'Зарегистрирован 24.09.2026 12:00');
    expect(screen.getByText('Alpha · ещё 1')).toHaveAttribute('title', 'Alpha; Beta');
    expect(screen.getByText('Исх. № 84 от 19.09.2026')).toBeInTheDocument();
    expect(screen.getByText('Администрация города')).toBeInTheDocument();
    expect(rows[2].querySelector('button')).toHaveClass('workspace-document-row--green');
    expect(rows[3].querySelector('button')).toHaveClass('workspace-document-row--purple');
    expect(screen.queryByRole('button', { name: /Все/ })).not.toBeInTheDocument();
    fireEvent.click(within(rows[0]).getByRole('button'));
    expect(open).toHaveBeenCalledWith('incoming', 'incoming_letter');
});

test.each([false, true])('availability %s distinguishes hidden card from an empty readable list', async (available) => {
    api.get.mockResolvedValue({ available, items: [] });
    renderWithApp(<RecentDocumentsPanel refreshVersion={0} onOpenDocument={vi.fn()} />);
    if (available) {
        expect(await screen.findByText('Доступных документов пока нет')).toBeInTheDocument();
    } else {
        await waitFor(() => expect(screen.queryByText('Последние документы')).not.toBeInTheDocument());
    }
});

test('failed invalidation clears stale rows and retry reloads independently', async () => {
    api.get.mockResolvedValueOnce({ available: true, items })
        .mockRejectedValueOnce(new Error('Connection lost'))
        .mockResolvedValueOnce({ available: true, items: [items[2]] });
    renderWithApp(<RecentDocumentsPanel refreshVersion={0} onOpenDocument={vi.fn()} />);
    await screen.findByText('Вх. № 125 от 20.09.2026');
    await act(async () => { api.listeners.forEach((listener) => listener({ topic: 'documents' })); });
    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(screen.queryByText('Вх. № 125 от 20.09.2026')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
    expect(await screen.findByText('Приказ № 41 от 22.09.2026')).toBeInTheDocument();
});

test('explicit refresh and reconnect reread data without needing a personal event', async () => {
    api.get.mockResolvedValue({ available: true, items });
    const { rerender } = renderWithApp(<RecentDocumentsPanel refreshVersion={0} onOpenDocument={vi.fn()} />);
    await screen.findByText('Вх. № 125 от 20.09.2026');
    rerender(<RecentDocumentsPanel refreshVersion={1} onOpenDocument={vi.fn()} />);
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(2));
    await act(async () => { api.listeners.forEach((listener) => listener({ topic: 'resync' })); });
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(3));
});
