import React from 'react';
import { act, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, test, vi } from 'vitest';
import SystemStatisticsTab from '../../src/features/statistics/SystemStatisticsTab';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

const systemStats = {
  userCount: 2,
  totalDocuments: 4,
  dbSize: '10 MB',
  storageObjects: 0,
  storageSize: '0 B',
  storageRefreshInProgress: false,
  generatedAt: '2026-09-01T12:00:00Z',
  service: { version: '1.0.6', state: 'ready', uptimeSeconds: 3660, schemaCurrentVersion: 12, schemaRequiredVersion: 12 },
  usage: { activeUsers15m: 3, activeSessions: 4 },
  api: { requestsSinceStart: 100, clientErrorsSinceStart: 2, serverErrorsSinceStart: 1, deadlineExceededSinceStart: 0, p95Milliseconds: 18, inFlight: 1, sampleWindow: 256 },
  database: { poolInUse: 2, poolOpen: 4, poolMax: 20, waitCountSinceStart: 1, operationsSinceStart: 200, operationErrorsSinceStart: 0, operationP95Milliseconds: 8 },
  outbox: { pending: 1, processing: 0, failed: 0, processedSinceStart: 20, retriesSinceStart: 1 },
  attachments: {},
};

afterEach(() => vi.useRealTimers());

describe('SystemStatisticsTab storage lifecycle', () => {
  test('renders service diagnostics returned by the server', async () => {
    installWailsMock({
      StatisticsService: {
        GetSystemStatistics: vi.fn().mockResolvedValue(systemStats),
        GetStorageStatisticsStatus: vi.fn().mockResolvedValue({ state: 'idle', storageObjects: 0, storageSize: '0 B' }),
        RetryStorageStatisticsRefresh: vi.fn(),
      },
    });

    renderWithApp(<SystemStatisticsTab />);

    expect(await screen.findByText('1.0.6')).toBeInTheDocument();
    expect(screen.getByText('Активные пользователи, 15 мин.')).toBeInTheDocument();
    expect(screen.getByText('p95, окно до 256 запросов')).toBeInTheDocument();
    expect(screen.getByText('Фоновая очередь')).toBeInTheDocument();
  });

  test('polling replaces a pending snapshot with the completed result', async () => {
    vi.useFakeTimers();
    const getStorageStatus = vi.fn()
      .mockResolvedValueOnce({ state: 'pending', storageObjects: 1, storageSize: '1 MB' })
      .mockResolvedValueOnce({ state: 'idle', storageObjects: 8, storageSize: '8 MB', refreshedAt: '2026-08-05T12:00:00Z' });
    installWailsMock({
      StatisticsService: {
        GetSystemStatistics: vi.fn().mockResolvedValue(systemStats),
        GetStorageStatisticsStatus: getStorageStatus,
        RetryStorageStatisticsRefresh: vi.fn(),
      },
    });
    renderWithApp(<SystemStatisticsTab />);

    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(getStorageStatus).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/Выполняется фоновая сверка MinIO/)).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1500);
    });
    expect(getStorageStatus).toHaveBeenCalledTimes(2);
    expect(screen.getByText('8 MB')).toBeInTheDocument();
    expect(screen.queryByText(/Выполняется фоновая сверка MinIO/)).not.toBeInTheDocument();
  });

  test('a failed refresh can be retried from the component', async () => {
    const retry = vi.fn().mockResolvedValue({ state: 'idle', storageObjects: 9, storageSize: '9 MB', refreshedAt: '2026-08-05T12:00:00Z' });
    installWailsMock({
      StatisticsService: {
        GetSystemStatistics: vi.fn().mockResolvedValue(systemStats),
        GetStorageStatisticsStatus: vi.fn().mockResolvedValue({ state: 'failed', storageObjects: 1, storageSize: '1 MB', lastError: 'MinIO недоступен' }),
        RetryStorageStatisticsRefresh: retry,
      },
    });
    const user = userEvent.setup();
    renderWithApp(<SystemStatisticsTab />);

    expect(await screen.findByText('Сверка MinIO завершилась с ошибкой')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Повторить сверку/ }));
    await waitFor(() => expect(retry).toHaveBeenCalledTimes(1));
    expect(await screen.findByText('9 MB')).toBeInTheDocument();
    expect(screen.queryByText('Сверка MinIO завершилась с ошибкой')).not.toBeInTheDocument();
  });
});
