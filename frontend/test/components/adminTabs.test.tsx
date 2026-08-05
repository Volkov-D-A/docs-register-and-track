import React from 'react';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, test, vi } from 'vitest';
import StorageTab from '../../src/features/settings/StorageTab';
import MigrationsTab from '../../src/features/settings/MigrationsTab';
import OutboxTab from '../../src/features/settings/OutboxTab';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

describe('critical administration tabs', () => {
  test('storage reconciliation prevents overlap and renders actionable discrepancies', async () => {
    const pending = deferred<{ missingObjects: string[]; orphanObjects: string[] }>();
    const reconcile = vi.fn().mockReturnValue(pending.promise);
    installWailsMock({ AttachmentService: { ReconcileStorage: reconcile } });
    renderWithApp(<StorageTab />);
    const button = screen.getByRole('button', { name: /Сверить с MinIO/ });

    fireEvent.click(button);
    fireEvent.click(button);
    await waitFor(() => expect(reconcile).toHaveBeenCalledTimes(1));
    await act(async () => {
      pending.resolve({ missingObjects: ['missing/report.pdf'], orphanObjects: ['orphan/scan.pdf'] });
    });

    expect(await screen.findByText('Ссылки без файлов: 1')).toBeInTheDocument();
    expect(screen.getByText('missing/report.pdf')).toBeInTheDocument();
    expect(screen.getByText('Файлы без ссылок: 1')).toBeInTheDocument();
    expect(screen.getByText('orphan/scan.pdf')).toBeInTheDocument();
  });

  test('migration execution requires confirmation and refreshes status', async () => {
    const getStatus = vi.fn()
      .mockResolvedValueOnce({ currentVersion: 10, availableCount: 11, latestAvailableVersion: 11, upToDate: false, schemaTooNew: false, dirty: false })
      .mockResolvedValueOnce({ currentVersion: 11, availableCount: 11, latestAvailableVersion: 11, upToDate: true, schemaTooNew: false, dirty: false });
    const runMigrations = vi.fn().mockResolvedValue(undefined);
    installWailsMock({ SettingsService: { GetMigrationStatus: getStatus, RunMigrations: runMigrations } });
    const user = userEvent.setup();
    renderWithApp(<MigrationsTab />);

    expect(await screen.findByText('Требуется обновление')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Применить миграции/ }));
    expect(runMigrations).not.toHaveBeenCalled();
    await user.click(await screen.findByRole('button', { name: 'Запустить' }));

    await waitFor(() => expect(runMigrations).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(getStatus).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('Актуальна')).toBeInTheDocument();
  });

  test('outbox terminal event can be requeued and the view reloads', async () => {
    const eventID = '123e4567-e89b-12d3-a456-426614174000';
    const getStats = vi.fn().mockResolvedValue({ Pending: 0, Processing: 0, Failed: 1, Processed: 5 });
    const getFailed = vi.fn().mockResolvedValue([{
      id: eventID,
      eventType: 'admin_audit',
      deduplicationKey: 'audit:test',
      attempts: 10,
      lastError: 'database unavailable',
      createdAt: '2026-08-05T10:00:00Z',
      failedAt: '2026-08-05T11:00:00Z',
    }]);
    const requeue = vi.fn().mockResolvedValue(undefined);
    installWailsMock({ OutboxAdminService: { GetStats: getStats, GetFailed: getFailed, Requeue: requeue } });
    const user = userEvent.setup();
    renderWithApp(<OutboxTab />);

    expect(await screen.findByText('database unavailable')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Повторить/ }));
    const confirmations = await screen.findAllByRole('button', { name: /Повторить/ });
    await user.click(confirmations.at(-1)!);

    await waitFor(() => expect(requeue).toHaveBeenCalledWith(eventID));
    await waitFor(() => expect(getStats).toHaveBeenCalledTimes(2));
    expect(getFailed).toHaveBeenCalledTimes(2);
  });
});
