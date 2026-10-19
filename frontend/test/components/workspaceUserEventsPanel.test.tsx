import React from 'react';
import { beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { dto } from '../../wailsjs/go/models';
import AppSidebar from '../../src/components/layout/AppSidebar';
import WorkspaceUserEventsPanel from '../../src/components/WorkspaceUserEventsPanel';
import { useUserEventsStore } from '../../src/store/useUserEventsStore';
import { renderWithApp } from '../componentTestUtils';
const api = vi.hoisted(() => ({ GetCurrentUserEvents: vi.fn(), MarkAllRead: vi.fn() }));
vi.mock('../../wailsjs/go/services/UserEventService', () => api);
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: () => () => {} }));
const items = Array.from({ length: 7 }, (_, index) => dto.UserEvent.createFrom({
    id: `event-${index}`, title: `Событие ${index}`, message: `Сообщение ${index}`, createdAt: '2026-10-19T09:00:00Z',
    documentId: `doc-${index}`, documentKind: 'incoming_letter', documentNumber: String(index),
}));
beforeEach(() => {
    vi.clearAllMocks();
    api.MarkAllRead.mockResolvedValue(undefined);
    api.GetCurrentUserEvents.mockResolvedValue({ items: items.map((item) => ({ ...item, readAt: '2026-10-19T10:00:00Z' })), totalCount: 27 });
    useUserEventsStore.setState({ events: items, unreadCount: 7, error: '', refresh: vi.fn(async () => {
        useUserEventsStore.setState({ events: items.map((item) => dto.UserEvent.createFrom({ ...item, readAt: '2026-10-19T10:00:00Z' })), unreadCount: 0 });
    }) });
});
const panel = (open = vi.fn()) => renderWithApp(<WorkspaceUserEventsPanel currentTime={new Date()} refreshVersion={0} onOpenDocument={open} />);

test('preview shows five events and counts unread events outside it', () => {
    panel();
    expect(screen.getByText('Сообщение 4')).toBeInTheDocument();
    expect(screen.queryByText('Сообщение 5')).not.toBeInTheDocument();
    expect(screen.getByTitle('Непрочитанных вне списка: 2')).toBeInTheDocument();
    expect(api.MarkAllRead).not.toHaveBeenCalled();
});

test('opening marks all read, clears the indicator and supports pages and documents', async () => {
    const open = vi.fn();
    panel(open);
    fireEvent.click(screen.getByRole('button', { name: 'Все события' }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    await waitFor(() => expect(useUserEventsStore.getState().unreadCount).toBe(0));
    expect(api.MarkAllRead).toHaveBeenCalledTimes(1);
    expect(await screen.findByText('Сообщение 6')).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('2'));
    await waitFor(() => expect(api.GetCurrentUserEvents).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, pageSize: 20 })));
    expect(api.MarkAllRead).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByText('Сообщение 6'));
    expect(open).toHaveBeenCalledWith('doc-6', 'incoming_letter');
    fireEvent.click(screen.getByRole('button', { name: 'Все события' }));
    await waitFor(() => expect(api.MarkAllRead).toHaveBeenCalledTimes(2));
});

test('failed mark-all keeps events unread and allows retry', async () => {
    api.MarkAllRead.mockRejectedValueOnce(new Error('Нет связи'));
    panel();
    fireEvent.click(screen.getByRole('button', { name: 'Все события' }));
    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(useUserEventsStore.getState().unreadCount).toBe(7);
    expect(api.GetCurrentUserEvents).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
    await waitFor(() => expect(useUserEventsStore.getState().unreadCount).toBe(0));
    expect(api.MarkAllRead).toHaveBeenCalledTimes(2);
});

test('badge subtracts only unread preview entries', () => {
    useUserEventsStore.setState({ events: [dto.UserEvent.createFrom({ ...items[0], readAt: '2026-10-19T10:00:00Z' }), ...items.slice(1)], unreadCount: 6 });
    panel();
    expect(screen.getByTitle('Непрочитанных вне списка: 2')).toBeInTheDocument();
});


test('dashboard menu indicator follows the shared unread count outside the dashboard', () => {
    const { container } = renderWithApp(<AppSidebar appTheme="light" token={{}} brandName="Docflow" currentPage="controls" sections={dto.AccessSections.createFrom({ dashboard: true })} expanded onExpandedChange={vi.fn()} onPageChange={vi.fn()} />);
    expect(container.querySelector('.ant-badge-dot')).toBeInTheDocument();
    act(() => useUserEventsStore.setState({ unreadCount: 0 }));
    expect(container.querySelector('.ant-badge-dot[data-show="true"]')).not.toBeInTheDocument();
});
