import React from 'react';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';
import DocumentStatisticsTab from '../../src/features/statistics/DocumentStatisticsTab';
import AssignmentStatisticsTab from '../../src/features/statistics/AssignmentStatisticsTab';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/features/statistics/DocumentCharts', () => ({ default: () => <div>document charts</div> }));
vi.mock('../../src/features/statistics/AssignmentCharts', () => ({ default: () => <div>assignment charts</div> }));

describe('statistics tabs reject stale report responses', () => {
  test('document report keeps the newest grouping result', async () => {
    const first = deferred<any>();
    const second = deferred<any>();
    const getReport = vi.fn()
      .mockImplementationOnce(() => first.promise)
      .mockImplementationOnce(() => second.promise);
    installWailsMock({
      StatisticsService: {
        GetDocumentStatistics: vi.fn().mockResolvedValue({ year: 2026, totalYear: 0, documentsByKindMonthly: [], documentsByRegistrarMonthly: [] }),
        GetDocumentFilterOptions: vi.fn().mockResolvedValue({ kinds: [], nomenclature: [], users: [] }),
        GetDocumentReport: getReport,
      },
    });
    renderWithApp(<DocumentStatisticsTab />);
    await waitFor(() => expect(getReport).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByRole('radio', { name: 'Дело' }));
    await waitFor(() => expect(getReport).toHaveBeenCalledTimes(2));
    await act(async () => {
      second.resolve({ total: 22, rows: [{ key: 'current', name: 'Текущий результат', count: 22 }] });
    });
    expect(await screen.findByText('Текущий результат')).toBeInTheDocument();

    await act(async () => {
      first.resolve({ total: 11, rows: [{ key: 'old', name: 'Устаревший результат', count: 11 }] });
    });
    expect(screen.getByText('Текущий результат')).toBeInTheDocument();
    expect(screen.queryByText('Устаревший результат')).not.toBeInTheDocument();
  });

  test('assignment report keeps the newest filter result', async () => {
    const first = deferred<any>();
    const second = deferred<any>();
    const getReport = vi.fn()
      .mockImplementationOnce(() => first.promise)
      .mockImplementationOnce(() => second.promise);
    installWailsMock({
      StatisticsService: {
        GetAssignmentStatistics: vi.fn().mockResolvedValue({ year: 2026, monthlyTotals: [], monthlyByExecutor: [], overdueRating: [], statusCounts: [] }),
        GetAssignmentFilterOptions: vi.fn().mockResolvedValue({ users: [] }),
        GetAssignmentReport: getReport,
      },
    });
    renderWithApp(<AssignmentStatisticsTab />);
    await waitFor(() => expect(getReport).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByText('С нарушением сроков'));
    await waitFor(() => expect(getReport).toHaveBeenCalledTimes(2));
    await act(async () => {
      second.resolve({ total: 7, rows: [{ key: 'current', name: 'Актуальные поручения', count: 7 }] });
    });
    expect(await screen.findByText('Актуальные поручения')).toBeInTheDocument();

    await act(async () => {
      first.resolve({ total: 3, rows: [{ key: 'old', name: 'Старые поручения', count: 3 }] });
    });
    expect(screen.getByText('Актуальные поручения')).toBeInTheDocument();
    expect(screen.queryByText('Старые поручения')).not.toBeInTheDocument();
  });
});
