import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import AcknowledgmentModal from '../../src/components/AcknowledgmentModal';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

test.each([false, true])('acknowledgment form omits content when editing=%s', async (editing) => {
  const createTask = vi.fn().mockResolvedValue({});
  const update = vi.fn().mockResolvedValue({});
  const success = vi.fn();
  installWailsMock({
    UserService: { GetExecutors: vi.fn().mockResolvedValue([{ id: 'recipient-1', fullName: 'Сотрудник' }]) },
    AssignmentService: { CreateTask: createTask, Update: update },
  });
  renderWithApp(<AcknowledgmentModal open documentId="doc-1" onCancel={vi.fn()} onSuccess={success}
    initialValues={editing ? {
      id: 'ack-1', content: 'Старый комментарий', deadline: '2026-12-31',
      users: [{ userId: 'recipient-1', userName: 'Сотрудник' }],
    } : undefined} />);
  expect(screen.queryByLabelText('Содержание / Комментарий')).not.toBeInTheDocument();
  expect(screen.queryByDisplayValue('Старый комментарий')).not.toBeInTheDocument();
  if (!editing) {
    fireEvent.mouseDown(screen.getByRole('combobox'));
    fireEvent.click(await screen.findByText('Сотрудник'));
  }
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));
  await waitFor(() => expect(success).toHaveBeenCalledOnce());
  if (editing) {
    expect(update).toHaveBeenCalledWith('ack-1', '', '', '2026-12-31', []);
    expect(createTask).not.toHaveBeenCalled();
  } else {
    expect(createTask).toHaveBeenCalledWith(expect.objectContaining({
      type: 'acknowledgment', documentId: 'doc-1', userIds: ['recipient-1'], deadline: '',
    }));
    expect(createTask.mock.lastCall?.[0].content).toBeUndefined();
    expect(update).not.toHaveBeenCalled();
  }
});
