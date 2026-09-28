import React from 'react';
import { ConfigProvider } from 'antd';
import { afterEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import DocumentSearchPanel from '../../src/components/DocumentSearchPanel';
import { deferred, renderWithApp } from '../componentTestUtils';

const api = vi.hoisted(() => ({ search: vi.fn(), listeners: [] as Array<(event: { topic: string; visibilityChanged?: boolean }) => void> }));
vi.mock('../../wailsjs/go/services/DocumentQueryService', () => ({ Search: api.search }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string; visibilityChanged?: boolean }) => void) => {
    api.listeners.push(listener);
    return () => { api.listeners = api.listeners.filter((value) => value !== listener); };
} }));
afterEach(() => { api.search.mockReset(); api.listeners = []; });

const item = (id: string, number: string) => ({ id, kindCode: 'incoming_letter', registrationNumber: number, registrationDate: '2026-09-28', content: 'Ремонт дороги', resolution: 'Подготовить ответ', correspondent: 'Администрация', person: 'Иванов', relevance: 0.5 });
const result = (items = [item('first', '123')], totalCount = items.length) => ({ items, totalCount, page: 1, pageSize: 20 });
const renderSearch = (onOpenDocument: (id: string, kind: string) => void = vi.fn()) => renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}><DocumentSearchPanel onOpenDocument={onOpenDocument} /></ConfigProvider>,
);
const submit = (text: string) => {
    fireEvent.change(screen.getByRole('searchbox', { name: 'Поиск по документам' }), { target: { value: text } });
    fireEvent.click(screen.getByRole('button', { name: 'Найти' }));
};

test('search opens a separate modal, preserves server relevance order and opens a card', async () => {
    api.search.mockResolvedValue(result([item('high', '321'), item('low', '123')]));
    const open = vi.fn();
    renderSearch(open);
    submit('  ремонт   Иванов  ');
    const dialog = await screen.findByRole('dialog', { name: 'Поиск документов' });
    expect(api.search).toHaveBeenCalledWith(expect.objectContaining({ query: 'ремонт Иванов', page: 1, pageSize: 20 }));
    const first = await within(dialog).findByRole('button', { name: '№ 321' });
    const rows = dialog.querySelectorAll('tbody tr[data-row-key]');
    expect(rows[0]).toHaveAttribute('data-row-key', 'high');
    expect(rows[1]).toHaveAttribute('data-row-key', 'low');
    expect(within(dialog).getAllByText('Подготовить ответ')).toHaveLength(2);
    fireEvent.click(first);
    expect(open).toHaveBeenCalledWith('high', 'incoming_letter');
    expect(dialog).toBeInTheDocument();
});

test('blank input never calls the server and pagination repeats the submitted query', async () => {
    api.search.mockResolvedValueOnce(result([item('first', '123')], 21)).mockResolvedValueOnce(result([item('second', '456')], 21));
    renderSearch();
    submit('   ');
    expect(api.search).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    submit('ремонт');
    await screen.findByRole('button', { name: '№ 123' });
    fireEvent.click(screen.getByTitle('2'));
    await screen.findByRole('button', { name: '№ 456' });
    expect(api.search).toHaveBeenLastCalledWith(expect.objectContaining({ query: 'ремонт', page: 2, pageSize: 20 }));
});

test('error displays a retry and retry can return an empty result', async () => {
    api.search.mockRejectedValueOnce(new Error('Connection lost')).mockResolvedValueOnce(result([]));
    renderSearch();
    submit('ремонт');
    await screen.findByRole('alert');
    fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
    expect(await screen.findByText('Документы не найдены')).toBeInTheDocument();
    expect(api.search).toHaveBeenCalledTimes(2);
});

test('closing and starting a new search discards a late response from the previous query', async () => {
    const old = deferred<ReturnType<typeof result>>();
    api.search.mockReturnValueOnce(old.promise).mockResolvedValueOnce(result([item('new', '456')]));
    renderSearch();
    submit('старый');
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    submit('новый');
    await screen.findByRole('button', { name: '№ 456' });
    await act(async () => { old.resolve(result([item('old', '123')])); });
    expect(screen.queryByRole('button', { name: '№ 123' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '№ 456' })).toBeInTheDocument();
});

test.each(['access-changed', 'document-changed'])('access invalidation %s clears visible results and pending replies', async (topic) => {
    const pending = deferred<ReturnType<typeof result>>();
    api.search.mockResolvedValueOnce(result()).mockReturnValueOnce(pending.promise);
    renderSearch();
    submit('ремонт');
    await screen.findByRole('button', { name: '№ 123' });
    await act(async () => { api.listeners.forEach((listener) => listener({ topic, visibilityChanged: topic === 'document-changed' })); });
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    submit('ремонт');
    await act(async () => { api.listeners.forEach((listener) => listener({ topic, visibilityChanged: topic === 'document-changed' })); });
    await act(async () => { pending.resolve(result()); });
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(screen.queryByRole('button', { name: '№ 123' })).not.toBeInTheDocument();
});
