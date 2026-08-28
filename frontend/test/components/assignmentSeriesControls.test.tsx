import React from 'react';
import { Table } from 'antd';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, test, vi } from 'vitest';
import { buildAssignmentColumns } from '../../src/components/assignmentListColumns';
import { renderWithApp } from '../componentTestUtils';

const recurringAssignment = {
  id: 'assignment-1',
  seriesId: 'series-1',
  createdAt: '2026-08-28T00:00:00Z',
  content: 'Регулярный отчёт',
  executorName: 'Исполнитель',
  status: 'new',
};

describe('assignment series controls', () => {
  test('does not expose series management to an executor', () => {
    const columns = buildAssignmentColumns({
      canManageAssignments: false,
      onEdit: vi.fn(),
      onDelete: vi.fn(),
      onManageSeries: vi.fn(),
    });
    renderWithApp(<Table rowKey="id" pagination={false} dataSource={[recurringAssignment]} columns={columns} />);
    expect(screen.queryByTitle('Управление серией')).not.toBeInTheDocument();
    expect(screen.getByText('Регулярный отчёт')).toBeInTheDocument();
  });

  test('shows the protected series action to an assignment manager', async () => {
    const onManageSeries = vi.fn();
    const columns = buildAssignmentColumns({
      canManageAssignments: true,
      onEdit: vi.fn(),
      onDelete: vi.fn(),
      onManageSeries,
    });
    renderWithApp(<Table rowKey="id" pagination={false} dataSource={[recurringAssignment]} columns={columns} />);
    await userEvent.click(screen.getByTitle('Управление серией'));
    expect(onManageSeries).toHaveBeenCalledWith(recurringAssignment);
    expect(screen.queryByTitle('Удалить поручение')).not.toBeInTheDocument();
  });
});
