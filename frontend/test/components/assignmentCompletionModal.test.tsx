import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import AssignmentCompletionModal from '../../src/components/AssignmentCompletionModal';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

test('uploads evidence to the current assignment before completing its report', async () => {
  const upload = vi.fn().mockResolvedValue([{ id: 'file-1', filename: 'report.txt' }]);
  const update = vi.fn().mockResolvedValue(undefined);
  const success = vi.fn();
  installWailsMock({
    SettingsService: { IsAssignmentCompletionAttachmentsEnabled: vi.fn().mockResolvedValue(true) },
    AttachmentService: { UploadForAssignment: upload },
    AssignmentService: { UpdateStatus: update },
  });
  renderWithApp(<AssignmentCompletionModal open assignmentId="assignment-1" documentId="doc-1"
    onCancel={vi.fn()} onSuccess={success} />);

  fireEvent.click(await screen.findByRole('button', { name: /Добавить файлы/ }));
  await waitFor(() => expect(upload).toHaveBeenCalledWith('assignment-1'));
  expect(update).not.toHaveBeenCalled();
  fireEvent.change(screen.getByPlaceholderText('Введите результат выполнения поручения...'), {
    target: { value: '  Выполнено  ' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Отметить исполненным' }));
  await waitFor(() => expect(update).toHaveBeenCalledWith('assignment-1', 'completed', 'Выполнено'));
  await waitFor(() => expect(success).toHaveBeenCalledOnce());
});
