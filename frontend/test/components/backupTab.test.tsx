import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import BackupTab from '../../src/features/settings/BackupTab';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

test('backup settings preserve a stored password and retry a staged archive', async () => {
  const settings = {
    smb: { host: 'freenas', share: 'backups', directory: 'docflow', user: 'backup', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1, 2, 3, 4, 5],
    retentionDays: 15, keepCopies: 3, passwordSet: true,
  };
  const save = vi.fn().mockResolvedValue(undefined);
  const retry = vi.fn().mockResolvedValue(undefined);
  installWailsMock({ SettingsService: {
    GetBackupSettings: vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' }),
    SaveBackupSettings: save,
    ListBackups: vi.fn().mockResolvedValue([{ id: 'backup-id', state: 'staged', createdAt: '2026-09-10T00:00:00Z', updatedAt: '2026-09-10T00:00:00Z', attempts: 5, archiveSize: 1024 }]),
    RetryBackup: retry,
    ListBackupCopies: vi.fn().mockResolvedValue([]),
  } });
  renderWithApp(<BackupTab />);
  expect(await screen.findByDisplayValue('freenas')).toBeInTheDocument();
  expect(screen.getByLabelText('Новый пароль (текущий сохранён)')).toHaveValue('');
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
  expect(save.mock.calls[0][0]).toMatchObject({ password: '', clearPassword: false, settings: { smb: { host: 'freenas' } } });
  fireEvent.click(screen.getByRole('button', { name: 'Повторить отправку' }));
  await waitFor(() => expect(retry).toHaveBeenCalledWith('backup-id'));
}, 15000);
