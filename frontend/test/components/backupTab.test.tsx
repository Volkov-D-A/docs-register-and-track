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
  await waitFor(() => expect(screen.getByLabelText('Сервер SMB')).toHaveValue('freenas'));
  expect(screen.getByRole('button', { name: 'Настройки подключения' })).toHaveAttribute('aria-expanded', 'false');
  expect(screen.queryByText('Домен (если нужен)')).not.toBeInTheDocument();
  expect(screen.queryByText('История заданий')).not.toBeInTheDocument();
  fireEvent.click(screen.getByText('Настройки подключения'));
  expect(screen.getByRole('button', { name: 'Настройки подключения' })).toHaveAttribute('aria-expanded', 'true');
  expect(screen.getByLabelText('Новый пароль (текущий сохранён)')).toHaveValue('');
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
  expect(save.mock.calls[0][0]).toMatchObject({ password: '', clearPassword: false, settings: { smb: { host: 'freenas' } } });
  fireEvent.click(screen.getByRole('button', { name: 'Повторить отправку' }));
  await waitFor(() => expect(retry).toHaveBeenCalledWith('backup-id'));
}, 15000);

test('an unconfigured SMB connection is not queried until settings are saved', async () => {
  const settings = {
    smb: { host: '', share: '', directory: '', user: '', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1],
    retentionDays: 15, keepCopies: 3, passwordSet: false,
  };
  const getSettings = vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' });
  const catalog = vi.fn().mockResolvedValue([]);
  const save = vi.fn().mockResolvedValue(undefined);
  const jobs = vi.fn().mockResolvedValue([]);
  installWailsMock({ SettingsService: {
    GetBackupSettings: getSettings, SaveBackupSettings: save,
    ListBackups: jobs, ListBackupCopies: catalog,
  } });
  renderWithApp(<BackupTab />);
  await waitFor(() => expect(jobs).toHaveBeenCalled());
  expect(screen.getByText('Сначала заполните и сохраните настройки SMB-подключения и пароль.')).toBeInTheDocument();
  expect(catalog).not.toHaveBeenCalled();
  expect(screen.getByRole('button', { name: 'Обновить каталог' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Создать копию' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Проверить сохранённое подключение' })).toBeDisabled();
  expect(screen.queryByText('Каталог SMB недоступен')).not.toBeInTheDocument();
  fireEvent.change(screen.getByLabelText('Сервер SMB'), { target: { value: 'nas' } });
  fireEvent.change(screen.getByLabelText('Общая папка'), { target: { value: 'backups' } });
  fireEvent.change(screen.getByLabelText('Пользователь'), { target: { value: 'backup' } });
  fireEvent.change(screen.getByLabelText('Пароль'), { target: { value: 'secret' } });
  expect(catalog).not.toHaveBeenCalled();
  getSettings.mockResolvedValue({ settings: { ...settings, smb: { ...settings.smb, host: 'nas', share: 'backups', user: 'backup' }, passwordSet: true }, issue: '', nextRun: '' });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(catalog).toHaveBeenCalledTimes(1));
  expect(screen.getByRole('button', { name: 'Обновить каталог' })).toBeEnabled();
  expect(screen.queryByText('Сначала заполните и сохраните настройки SMB-подключения и пароль.')).not.toBeInTheDocument();
}, 15000);

test('backup progress does not refresh an unchanged catalog and maintenance preserves copies', async () => {
  const settings = {
    smb: { host: 'nas', share: 'backups', directory: '', user: 'backup', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1],
    retentionDays: 15, keepCopies: 3, passwordSet: true,
  };
  const job = { id: 'backup-live', state: 'queued', createdAt: '2026-09-14T00:00:00Z', updatedAt: '2026-09-14T00:00:00Z', archiveSize: 0 };
  const jobs = vi.fn().mockResolvedValue([job]);
  const catalog = vi.fn().mockResolvedValue([{ id: 'saved-copy', format: 3, createdAt: job.createdAt, size: 1024, verification: 'verified' }]);
  installWailsMock({ SettingsService: {
    GetBackupSettings: vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' }),
    ListBackups: jobs, ListBackupCopies: catalog,
    StartBackup: vi.fn().mockImplementation(async () => { jobs.mockResolvedValue([{ ...job, state: 'snapshotting' }]); }),
  } });
  renderWithApp(<BackupTab />);
  expect(await screen.findByText('saved-copy')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Создать копию' }));
  expect(await screen.findByText('Создание снимка')).toBeInTheDocument();
  expect(catalog).toHaveBeenCalledTimes(1);
  catalog.mockRejectedValueOnce(JSON.stringify({ code: 'MAINTENANCE', status: 503, message: 'Обслуживание' }));
  fireEvent.click(await screen.findByRole('button', { name: /Обновить каталог/ }));
  expect(await screen.findByText('Обновление каталога отложено до завершения обслуживания сервера.')).toBeInTheDocument();
  expect(screen.getByText('saved-copy')).toBeInTheDocument();
  expect(screen.queryByText('Каталог SMB недоступен')).not.toBeInTheDocument();
  await waitFor(() => expect(screen.queryByText('Обновление каталога отложено до завершения обслуживания сервера.')).not.toBeInTheDocument(), { timeout: 5000 });
  catalog.mockRejectedValueOnce(JSON.stringify({ code: 'INTERNAL_ERROR', status: 503, message: 'SMB unavailable' }));
  fireEvent.click(await screen.findByRole('button', { name: /Обновить каталог/ }));
  expect(await screen.findByText('Каталог SMB недоступен')).toBeInTheDocument();
}, 15000);
