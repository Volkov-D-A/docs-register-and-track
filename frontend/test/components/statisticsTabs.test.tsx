import React from 'react';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';
import DocumentStatisticsTab from '../../src/features/statistics/DocumentStatisticsTab';
import DeadlineStatisticsTab from '../../src/features/statistics/DeadlineStatisticsTab';
import { deferred, installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/features/statistics/DocumentCharts', () => ({ default: () => <div>document charts</div> }));
vi.mock('../../src/features/statistics/DeadlineCharts', () => ({ default: () => <div>assignment charts</div> }));

const events = vi.hoisted(() => ({ listeners: [] as Array<(event: { topic: string }) => void> }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string }) => void) => { events.listeners.push(listener); return () => { events.listeners = events.listeners.filter((item) => item !== listener); }; } }));

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

  test('deadline report keeps the newest filter result', async () => {
    const first = deferred<any>();
    const second = deferred<any>();
    const getReport = vi.fn()
      .mockImplementationOnce(() => first.promise)
      .mockImplementationOnce(() => second.promise);
    installWailsMock({
      StatisticsService: {
        GetDeadlineStatistics: vi.fn().mockResolvedValue({ year: 2026, monthlyTotals: [], monthlyByExecutor: [], overdueRating: [], statusCounts: [] }),
        GetDeadlineFilterOptions: vi.fn().mockResolvedValue({ users: [] }),
        GetDeadlineReport: getReport,
      },
    });
    renderWithApp(<DeadlineStatisticsTab />);
    await waitFor(() => expect(getReport).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByText('С нарушением сроков'));
    await waitFor(() => expect(getReport).toHaveBeenCalledTimes(2));
    await act(async () => {
      second.resolve({ total: 7, rows: [{ key: 'current', name: 'Актуальные контроли', count: 7 }] });
    });
    expect(await screen.findByText('Актуальные контроли')).toBeInTheDocument();

    await act(async () => {
      first.resolve({ total: 3, rows: [{ key: 'old', name: 'Старые контроли', count: 3 }] });
    });
    expect(screen.getByText('Актуальные контроли')).toBeInTheDocument();
    expect(screen.queryByText('Старые контроли')).not.toBeInTheDocument();
  });
});

test('statistics clear visible totals and ignore old responses when access is revoked', async () => {
 const pending=deferred<any>();const next=deferred<any>();
 const report=vi.fn().mockResolvedValueOnce({total:3,rows:[{key:'visible',name:'Доступный исполнитель',count:3}]}).mockImplementationOnce(()=>pending.promise).mockImplementation(()=>next.promise);
 installWailsMock({StatisticsService:{GetDeadlineStatistics:vi.fn().mockResolvedValue({year:2026,monthlyTotals:[],monthlyByExecutor:[],overdueRating:[],statusCounts:[]}),GetDeadlineFilterOptions:vi.fn().mockResolvedValue({users:[]}),GetDeadlineReport:report}});
 renderWithApp(<DeadlineStatisticsTab/>);
 expect(await screen.findByText('Доступный исполнитель')).toBeInTheDocument();
 fireEvent.click(screen.getByText('С нарушением сроков'));await waitFor(()=>expect(report).toHaveBeenCalledTimes(2));
 act(()=>events.listeners.forEach((listener)=>listener({topic:'resync'})));
 expect(screen.queryByText('Доступный исполнитель')).not.toBeInTheDocument();
 await act(async()=>pending.resolve({total:10,rows:[{key:'stale',name:'Отозванный исполнитель',count:10}]}));
 expect(screen.queryByText('Отозванный исполнитель')).not.toBeInTheDocument();
});
