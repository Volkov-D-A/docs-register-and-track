import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import BackupCatalog from '../../src/features/settings/BackupCatalog';
import { models } from '../../wailsjs/go/models';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

const copy = models.BackupCopy.createFrom({ id: 'copy-id', format: 3, createdAt: '2026-09-10T00:00:00Z', size: 1024, verification: 'unverified', issue: '', canDelete: true, restoreConfirmation: 'restore copy-id date', deleteConfirmation: 'delete copy-id date' });

test('restore requires successful verification and explicit replacement confirmation', async () => {
  const start = vi.fn().mockResolvedValueOnce({ job: { id: 'verification', kind: 'verify', state: 'queued' }, statusToken: 'read-only-token', expiresAt: '2099-01-01T00:00:00Z' }).mockResolvedValueOnce({ job: { id: 'restore', kind: 'restore', state: 'queued' }, statusToken: 'restore-token', expiresAt: '2099-01-01T00:00:00Z' });
  const poll = vi.fn().mockResolvedValue({ id: 'verification', kind: 'verify', state: 'completed' });
  installWailsMock({ SettingsService: { StartBackupOperation: start, GetBackupOperation: poll } });
  renderWithApp(<BackupCatalog copies={[copy]} loading={false} onChanged={vi.fn().mockResolvedValue(undefined)} />);
  fireEvent.click(screen.getByRole('button', { name: 'Восстановить' }));
  expect(screen.getByText(/Потребуется пароль пользователя из архива/)).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Заменить данные' })).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Проверить выбранную копию' }));
  await waitFor(() => expect(start).toHaveBeenCalledWith('verify', expect.objectContaining({ copyId: 'copy-id' })));
  const replace = await screen.findByRole('button', { name: 'Заменить данные' }, { timeout: 5000 });
  expect(replace).toBeDisabled();
  expect(poll).toHaveBeenCalledWith('verification', 'read-only-token');
  fireEvent.click(screen.getByRole('checkbox', { name: 'Подтверждаю замену текущих данных выбранной копией' }));
  fireEvent.click(replace);
  await waitFor(() => expect(start).toHaveBeenCalledWith('restore', expect.objectContaining({ copyId: 'copy-id', verificationId: 'verification', confirmation: copy.restoreConfirmation })));
}, 15000);

test('legacy deletion and incomplete recovery stay disabled', () => {
  renderWithApp(<BackupCatalog copies={[models.BackupCopy.createFrom({ ...copy, format: 2, canDelete: false, verification: 'incomplete' })]} loading={false} onChanged={vi.fn()} />);
  expect(screen.getByRole('button', { name: 'Удалить' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Восстановить' })).toBeDisabled();
});

test('operation completion arrives through a server event without repeated status requests', async () => {
  const job = { id: 'verify-live', kind: 'verify', state: 'queued', updatedAt: '2026-09-10T00:00:00Z' };
  const start = vi.fn().mockResolvedValue({ job, statusToken: 'capability', expiresAt: '2099-01-01T00:00:00Z' });
  const status = vi.fn().mockResolvedValue(job);
  const changed = vi.fn().mockResolvedValue(undefined);
  installWailsMock({ SettingsService: { StartBackupOperation: start, GetBackupOperation: status } });
  renderWithApp(<BackupCatalog copies={[copy]} loading={false} onChanged={changed} />);
  fireEvent.click(screen.getByRole('button', { name: 'Проверить' }));
  fireEvent.click(screen.getByRole('button', { name: 'Проверить выбранную копию' }));
  await waitFor(() => expect(status).toHaveBeenCalledTimes(1));
  fireEvent(window, new CustomEvent('server:event', { detail: { topic: 'operation', revision: 0, operation: { ...job, state: 'completed', updatedAt: '2026-09-10T00:00:01Z' } } }));
  await waitFor(() => expect(changed).toHaveBeenCalledTimes(1));
  expect(status).toHaveBeenCalledTimes(1);
});
