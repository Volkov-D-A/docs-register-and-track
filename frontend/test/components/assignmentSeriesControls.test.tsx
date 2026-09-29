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


describe('acknowledgment assignment controls', () => {
  const acknowledgment = { id: 'ack-1', type: 'acknowledgment', creatorId: 'creator', status: 'new', createdAt: '2026-09-29', content: 'Ознакомиться', users: [
    { userId: 'a', userName: 'Первый', confirmedAt: '2026-09-29T10:00:00Z' }, { userId: 'b', userName: 'Второй' },
  ] };
  test('assign permission manages acknowledgment assignments too', () => {
    const columns = buildAssignmentColumns({ canManageAssignments: true, currentUserId: 'creator', onEdit: vi.fn(), onDelete: vi.fn(), onManageSeries: vi.fn() });
    renderWithApp(<Table rowKey="id" pagination={false} dataSource={[acknowledgment]} columns={columns} />);
    expect(screen.getByText('Ожидает (1/2)')).toBeInTheDocument();
    expect(screen.getByTitle('Редактировать поручение')).toBeInTheDocument();
    expect(screen.getByTitle('Удалить поручение')).toBeInTheDocument();
  });
  test('without assign permission acknowledgment management is hidden', () => {
    const columns = buildAssignmentColumns({ canManageAssignments: false, currentUserId: 'creator', onEdit: vi.fn(), onDelete: vi.fn(), onManageSeries: vi.fn() });
    renderWithApp(<Table rowKey="id" pagination={false} dataSource={[acknowledgment]} columns={columns} />);
    expect(screen.queryByTitle('Редактировать поручение')).not.toBeInTheDocument();
    expect(screen.queryByTitle('Удалить поручение')).not.toBeInTheDocument();
  });
  test('assignment controller can edit but only the creator can delete', () => {
    const columns = buildAssignmentColumns({ canManageAssignments: true, currentUserId: 'other', onEdit: vi.fn(), onDelete: vi.fn(), onManageSeries: vi.fn() });
    renderWithApp(<Table rowKey="id" pagination={false} dataSource={[acknowledgment]} columns={columns} />);
    expect(screen.getByTitle('Редактировать поручение')).toBeInTheDocument();
    expect(screen.queryByTitle('Удалить поручение')).not.toBeInTheDocument();
  });
});
