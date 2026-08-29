import React from 'react';
import { screen, waitFor } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';
import AssignmentSeriesModal from '../../src/components/AssignmentSeriesModal';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

const firstSeries = {
  id: 'series-1',
  documentId: 'document-1',
  executorId: 'executor-1',
  content: 'Параметры первой серии',
  intervalUnit: 'month',
  intervalValue: 1,
  dayRule: 'fixed',
  dayOfMonth: 1,
  coExecutorIds: [],
  active: true,
};

describe('AssignmentSeriesModal loading lifecycle', () => {
  test('does not retain or submit the previous series after the next load fails', async () => {
    const getSeries = vi.fn()
      .mockResolvedValueOnce(firstSeries)
      .mockRejectedValueOnce(new Error('series load failed'));
    installWailsMock({
      AssignmentService: {
        GetSeries: getSeries,
        GetSeriesHistory: vi.fn().mockResolvedValue([]),
        UpdateSeries: vi.fn(),
        CancelSeries: vi.fn(),
      },
      UserService: {
        GetExecutors: vi.fn().mockResolvedValue([{ id: 'executor-1', fullName: 'Исполнитель' }]),
      },
      AttachmentService: {
        GetAssignmentFiles: vi.fn().mockResolvedValue([]),
      },
    });

    const view = renderWithApp(<AssignmentSeriesModal
      open
      seriesId="series-1"
      documentId="document-1"
      onCancel={vi.fn()}
      onSuccess={vi.fn()}
    />);

    expect(await screen.findByDisplayValue('Параметры первой серии')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Сохранить' })).toBeInTheDocument();

    view.rerender(<AssignmentSeriesModal
      open
      seriesId="series-2"
      documentId="document-2"
      onCancel={vi.fn()}
      onSuccess={vi.fn()}
    />);

    await waitFor(() => expect(getSeries).toHaveBeenCalledWith('series-2'));
    await waitFor(() => expect(screen.queryByDisplayValue('Параметры первой серии')).not.toBeInTheDocument());
    expect(screen.queryByRole('button', { name: 'Сохранить' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Отменить серию' })).not.toBeInTheDocument();
  });
});
