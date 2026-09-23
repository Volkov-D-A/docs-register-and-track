import React from 'react';
import { ConfigProvider } from 'antd';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { afterEach, expect, test, vi } from 'vitest';
import BackupTab from '../../src/features/settings/BackupTab';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

afterEach(() => vi.useRealTimers());

test('backup settings preserve a stored password and retry a staged archive', async () => {
  const settings = {
    smb: { host: 'freenas', share: 'backups', directory: 'docflow', user: 'backup', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1, 2, 3, 4, 5],
    retentionDays: 15, keepCopies: 3, passwordSet: true,
  };
  const save = vi.fn().mockResolvedValue(undefined);
  const retry = vi.fn().mockResolvedValue(undefined);
  const check = vi.fn().mockResolvedValue(undefined);
  installWailsMock({ SettingsService: {
    GetBackupSettings: vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' }),
    SaveBackupSettings: save,
    ListBackups: vi.fn().mockResolvedValue([{ id: 'backup-id', state: 'staged', createdAt: '2026-09-10T00:00:00Z', updatedAt: '2026-09-10T00:00:00Z', attempts: 5, archiveSize: 1024 }]),
    RetryBackup: retry, CheckBackupConnection: check,
    ListBackupCopies: vi.fn().mockResolvedValue([]),
  } });
  renderWithApp(<ConfigProvider theme={{ token: { motion: false } }}><BackupTab /></ConfigProvider>);
  expect(screen.queryByRole('button', { name: 'Сохранить' })).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Журнал операций резервирования' })).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Настройки резервного копирования' }));
  const settingsDialog = await screen.findByRole('dialog', { name: 'Настройки резервного копирования' });
  await waitFor(() => expect(within(settingsDialog).getByLabelText('Сервер SMB')).toHaveValue('freenas'));
  await waitFor(() => expect(within(settingsDialog).getByRole('heading', { name: 'Настройки подключения' })).toBeVisible());
  expect(within(settingsDialog).getByRole('heading', { name: 'Настройки расписания' })).toBeVisible();
  expect(within(settingsDialog).getByLabelText('Сервер SMB')).toBeVisible();
  expect(within(settingsDialog).getByLabelText('Время')).toBeVisible();
  expect(screen.queryByText('Домен (если нужен)')).not.toBeInTheDocument();
  expect(screen.queryByText('История заданий')).not.toBeInTheDocument();
  expect(screen.getByLabelText('Новый пароль (текущий сохранён)')).toHaveValue('');
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
  expect(save.mock.calls[0][0]).toMatchObject({ password: '', clearPassword: false, settings: { smb: { host: 'freenas' } } });
  await waitFor(() => expect(within(settingsDialog).getByRole('button', { name: 'Проверить сохранённое подключение' })).toBeEnabled());
  fireEvent.click(within(settingsDialog).getByRole('button', { name: 'Проверить сохранённое подключение' }));
  await waitFor(() => expect(check).toHaveBeenCalledTimes(1));
  fireEvent.click(within(settingsDialog).getByRole('button', { name: /close/i }));
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  expect(screen.queryByText('Ожидает отправки')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Создать копию' }));
  const dialog = await screen.findByRole('dialog');
  expect(within(dialog).getByText('Ожидает отправки')).toBeInTheDocument();
  fireEvent.click(within(dialog).getByRole('button', { name: 'Повторить отправку' }));
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
  renderWithApp(<ConfigProvider theme={{ token: { motion: false } }}><BackupTab /></ConfigProvider>);
  await waitFor(() => expect(jobs).toHaveBeenCalled());
  expect(screen.getByText('Сначала заполните и сохраните настройки SMB-подключения и пароль.')).toBeInTheDocument();
  expect(catalog).not.toHaveBeenCalled();
  expect(screen.getByRole('button', { name: 'Обновить каталог' })).toBeDisabled();
  expect(screen.getByRole('button', { name: 'Создать копию' })).toBeDisabled();
  fireEvent.click(screen.getByRole('button', { name: 'Настройки резервного копирования' }));
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

test('backup progress does not refresh an unchanged catalog', async () => {
  const settings = {
    smb: { host: 'nas', share: 'backups', directory: '', user: 'backup', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1],
    retentionDays: 15, keepCopies: 3, passwordSet: true,
  };
  const job = { id: 'backup-live', state: 'queued', createdAt: '2026-09-14T00:00:00Z', updatedAt: '2026-09-14T00:00:00Z', archiveSize: 0 };
  const jobs = vi.fn().mockResolvedValue([]);
  const catalog = vi.fn().mockResolvedValue([{ id: 'saved-copy', format: 3, createdAt: job.createdAt, size: 1024, verification: 'verified' }]);
  installWailsMock({ SettingsService: {
    GetBackupSettings: vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' }),
    ListBackups: jobs, ListBackupCopies: catalog,
    StartBackup: vi.fn().mockImplementation(async () => {
      const started = { ...job, state: 'snapshotting' };
      jobs.mockResolvedValue([started]);
      return started;
    }),
  } });
  renderWithApp(<ConfigProvider theme={{ token: { motion: false } }}><BackupTab /></ConfigProvider>);
  expect(await screen.findByText('saved-copy')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Создать копию' }));
  const dialog = await screen.findByRole('dialog');
  fireEvent.click(within(dialog).getByRole('button', { name: 'Начать создание копии' }));
  expect(await within(dialog).findByText('Создание снимка')).toBeInTheDocument();
  expect(within(dialog).getByRole('button', { name: 'Начать создание копии' })).toBeDisabled();
  fireEvent.click(within(dialog).getByRole('button', { name: /close/i }));
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  expect(screen.queryByText('Создание снимка')).not.toBeInTheDocument();
  expect(screen.queryByText('Текущее задание')).not.toBeInTheDocument();
  expect(catalog).toHaveBeenCalledTimes(1);
});

test('maintenance keeps saved copies until catalog retry succeeds', async () => {
  const settings = {
    smb: { host: 'nas', share: 'backups', directory: '', user: 'backup', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1],
    retentionDays: 15, keepCopies: 3, passwordSet: true,
  };
  const savedCopy = { id: 'saved-copy', format: 3, createdAt: '2026-09-14T00:00:00Z', size: 1024, verification: 'verified' };
  const catalog = vi.fn().mockResolvedValue([savedCopy]);
  installWailsMock({ SettingsService: {
    GetBackupSettings: vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' }),
    ListBackups: vi.fn().mockResolvedValue([]), ListBackupCopies: catalog,
  } });
  renderWithApp(<ConfigProvider theme={{ token: { motion: false } }}><BackupTab /></ConfigProvider>);
  expect(await screen.findByText('saved-copy')).toBeInTheDocument();
  vi.useFakeTimers();
  catalog.mockRejectedValueOnce(JSON.stringify({ code: 'MAINTENANCE', status: 503, message: 'Обслуживание' }));
  fireEvent.click(screen.getByRole('button', { name: /Обновить каталог/ }));
  await act(async () => { await Promise.resolve(); });
  expect(screen.getByText('Обновление каталога отложено до завершения обслуживания сервера.')).toBeInTheDocument();
  expect(screen.getByText('saved-copy')).toBeInTheDocument();
  expect(screen.queryByText('Каталог SMB недоступен')).not.toBeInTheDocument();
  await act(async () => { await vi.advanceTimersByTimeAsync(3000); });
  expect(screen.queryByText('Обновление каталога отложено до завершения обслуживания сервера.')).not.toBeInTheDocument();
  expect(catalog).toHaveBeenCalledTimes(3);
  expect(screen.getByText('saved-copy')).toBeInTheDocument();
  catalog.mockRejectedValueOnce(JSON.stringify({ code: 'INTERNAL_ERROR', status: 503, message: 'SMB unavailable' }));
  fireEvent.click(screen.getByRole('button', { name: /Обновить каталог/ }));
  await act(async () => { await Promise.resolve(); });
  expect(screen.getByText('Каталог SMB недоступен')).toBeInTheDocument();
});

test('a new creation dialog does not show the previous completed backup', async () => {
  const settings = {
    smb: { host: 'nas', share: 'backups', directory: '', user: 'backup', domain: '' },
    enabled: false, time: '02:00', timezone: 'Asia/Yekaterinburg', weekdays: [1],
    retentionDays: 15, keepCopies: 3, passwordSet: true,
  };
  const previous = { id: 'previous-copy', state: 'completed', archiveSize: 1024 };
  const created = { ...previous, id: 'new-copy' };
  const jobs = vi.fn().mockResolvedValue([previous]);
  const start = vi.fn().mockImplementation(async () => {
    jobs.mockResolvedValue([created, previous]);
    return created;
  });
  installWailsMock({ SettingsService: {
    GetBackupSettings: vi.fn().mockResolvedValue({ settings, issue: '', nextRun: '' }),
    ListBackups: jobs, ListBackupCopies: vi.fn().mockResolvedValue([]), StartBackup: start,
  } });
  renderWithApp(<ConfigProvider theme={{ token: { motion: false } }}><BackupTab /></ConfigProvider>);
  await waitFor(() => expect(jobs).toHaveBeenCalled());
  fireEvent.click(screen.getByRole('button', { name: 'Создать копию' }));
  const dialog = await screen.findByRole('dialog');
  expect(within(dialog).queryByText(/previous-copy/)).not.toBeInTheDocument();
  expect(within(dialog).queryByText('Завершено')).not.toBeInTheDocument();
  expect(start).not.toHaveBeenCalled();
  fireEvent.click(within(dialog).getByRole('button', { name: 'Начать создание копии' }));
  expect(await within(dialog).findByText('ID: new-copy')).toBeInTheDocument();
  expect(within(dialog).getByText('Завершено')).toBeInTheDocument();
  expect(within(dialog).queryByRole('button', { name: 'Начать создание копии' })).not.toBeInTheDocument();
  fireEvent.click(within(dialog).getByRole('button', { name: /close/i }));
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  await waitFor(() => expect(screen.getByRole('button', { name: 'Создать копию' })).toBeEnabled());
  fireEvent.click(screen.getByRole('button', { name: 'Создать копию' }));
  const reopened = await screen.findByRole('dialog');
  expect(within(reopened).queryByText(/new-copy|previous-copy|Завершено/)).not.toBeInTheDocument();
  expect(within(reopened).getByRole('button', { name: 'Начать создание копии' })).toBeEnabled();
}, 15000);
