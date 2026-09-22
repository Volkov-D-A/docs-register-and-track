import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import AssignmentCompletionModal from '../../src/components/AssignmentCompletionModal';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

test('uploads evidence to the current assignment before completing its report', async () => {
  const upload = vi.fn().mockResolvedValue({ items: [{ filename: 'report.txt', attachment: { id: 'file-1', filename: 'report.txt' } }] });
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

 test('shows successful files and failures without losing the completion report', async () => {
  const upload = vi.fn().mockResolvedValue({ items: [
    { filename: 'first.txt', attachment: { id: 'file-1', filename: 'first.txt' } },
    { filename: 'second.txt', error: { code: 'VALIDATION_ERROR', message: 'Файл слишком большой', status: 400 } },
  ] });
  installWailsMock({
    SettingsService: { IsAssignmentCompletionAttachmentsEnabled: vi.fn().mockResolvedValue(true) },
    AttachmentService: { UploadForAssignment: upload },
  });
  renderWithApp(<AssignmentCompletionModal open assignmentId="assignment-1" documentId="doc-1"
    initialReport="Мой отчёт" onCancel={vi.fn()} onSuccess={vi.fn()} />);
  fireEvent.click(await screen.findByRole('button', { name: /Добавить файлы/ }));
  expect(await screen.findByText('Загружено: 1. Ошибок: 1.')).toBeInTheDocument();
  expect(screen.getByText('first.txt: загружен')).toBeInTheDocument();
  expect(screen.getByText(/second.txt: Файл слишком большой/)).toBeInTheDocument();
  expect(screen.getByDisplayValue('Мой отчёт')).toBeInTheDocument();
});
