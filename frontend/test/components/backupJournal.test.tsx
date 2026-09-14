import React from 'react';
import { ConfigProvider } from 'antd';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import BackupJournal from '../../src/features/settings/BackupJournal';
import { models } from '../../wailsjs/go/models';
import { renderWithApp } from '../componentTestUtils';

const job = (id: string, kind = '') => models.BackupJob.createFrom({
  id, kind, copyId: kind ? 'scheduled-copy' : '', state: 'completed',
  createdAt: '2026-09-14T00:00:00Z', updatedAt: '2026-09-14T00:01:00Z',
  stages: [{ state: kind === 'delete' ? 'deleting' : 'snapshotting', startedAt: '2026-09-14T00:00:00Z' }, { state: 'completed', startedAt: '2026-09-14T00:01:00Z' }],
});

test('the journal opens persisted backups and catalog operations with their stages', async () => {
  const refresh = vi.fn().mockResolvedValue(undefined);
  renderWithApp(<ConfigProvider theme={{ token: { motion: false } }}><BackupJournal jobs={[job('scheduled-copy'), job('deletion', 'delete')]} onRefresh={refresh} /></ConfigProvider>);
  fireEvent.click(screen.getByRole('button', { name: 'Журнал операций резервирования' }));
  const journal = await screen.findByRole('dialog', { name: 'Журнал операций резервирования' });
  await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
  fireEvent.click(within(journal).getByRole('button', { name: 'Создание копии' }));
  const details = (await screen.findByText('ID операции: scheduled-copy')).closest('[role="dialog"]') as HTMLElement;
  expect(within(details).getByText('ID операции: scheduled-copy')).toBeInTheDocument();
  expect(within(details).getByText(/Создание снимка/)).toBeInTheDocument();
  fireEvent.click(within(details).getByRole('button', { name: /close/i }));
  await waitFor(() => expect(screen.queryByText('ID операции: scheduled-copy')).not.toBeInTheDocument());
  fireEvent.click(within(journal).getByRole('button', { name: 'Удаление копии' }));
  const deletion = (await screen.findByText('ID операции: deletion')).closest('[role="dialog"]') as HTMLElement;
  expect(within(deletion).getByText('ID операции: deletion')).toBeInTheDocument();
  expect(within(deletion).getByText(/Удаление ·/)).toBeInTheDocument();
  expect(within(deletion).queryByRole('checkbox')).not.toBeInTheDocument();
});

test('journal details follow server events and reject older operation updates', async () => {
  const current = { ...job('verification', 'verify'), state: 'queued', stages: [] };
  renderWithApp(<BackupJournal jobs={[current]} onRefresh={vi.fn().mockResolvedValue(undefined)} />);
  fireEvent.click(screen.getByRole('button', { name: 'Журнал операций резервирования' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Проверка копии' }));
  const details = (await screen.findByText('ID операции: verification')).closest('[role="dialog"]') as HTMLElement;
  const completed = { ...current, state: 'completed', updatedAt: '2026-09-14T00:02:00Z' };
  fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'operation', revision: 0, operation: completed } }));
  expect(await within(details).findByText('Завершено')).toBeInTheDocument();
  fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'operation', revision: 0, operation: current } }));
  expect(within(details).getByText('Завершено')).toBeInTheDocument();
});

test('journal refresh failures are visible without discarding stored history', async () => {
  renderWithApp(<BackupJournal jobs={[job('saved')]} onRefresh={vi.fn().mockRejectedValue(new Error('Unavailable'))} />);
  fireEvent.click(screen.getByRole('button', { name: 'Журнал операций резервирования' }));
  expect(await screen.findByText('Не удалось обновить журнал')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Создание копии' })).toBeInTheDocument();
});
