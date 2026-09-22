import React from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import { useAttachments } from '../../src/hooks/useAttachments';
import { installWailsMock, renderWithApp } from '../componentTestUtils';

vi.mock('../../src/hooks/useDocumentKindAccess', () => ({
  useDocumentKindAccess: () => ({ ready: true, hasAction: () => true }),
}));

const Harness = () => {
  const { files, uploadFile, contextHolder } = useAttachments({ documentId: 'doc-1', documentKind: 'incoming_letter' });
  return <>
    {contextHolder}
    <button onClick={() => void uploadFile()}>upload</button>
    <ul aria-label="Вложения">{files.map((file) => <li key={file.id}>{file.filename}</li>)}</ul>
  </>;
};

test('refreshes saved files and reports each outcome after partial success', async () => {
  const list = vi.fn().mockResolvedValueOnce([]).mockResolvedValue([{ id: 'file-1', filename: 'first.txt' }]);
  const upload = vi.fn().mockResolvedValue({ items: [
    { filename: 'first.txt', attachment: { id: 'file-1', filename: 'first.txt' } },
    { filename: 'second.txt', error: { code: 'VALIDATION_ERROR', message: 'Файл слишком большой', status: 400 } },
  ] });
  installWailsMock({ AttachmentService: { GetList: list, Upload: upload } });
  renderWithApp(<Harness />);
  await waitFor(() => expect(list).toHaveBeenCalledTimes(1));
  fireEvent.click(screen.getByRole('button', { name: 'upload' }));
  expect(await screen.findByText('first.txt')).toBeInTheDocument();
  expect(await screen.findByText('Загружено: 1. Ошибок: 1.')).toBeInTheDocument();
  expect(screen.getByText(/second.txt: Файл слишком большой/)).toBeInTheDocument();
  expect(upload).toHaveBeenCalledTimes(1);
  expect(list).toHaveBeenCalledTimes(2);
});

test.each([false, true])('refreshes after an uncertain upload outcome (bridge rejection: %s)', async (bridgeError) => {
  const list = vi.fn().mockResolvedValueOnce([]).mockResolvedValue([{ id: 'file-1', filename: 'saved.txt' }]);
  const upload = bridgeError ? vi.fn().mockRejectedValue(new Error('lost bridge response')) : vi.fn().mockResolvedValue({
    items: [{ filename: 'saved.txt', error: { code: 'INTERNAL_ERROR', message: 'response lost', status: 500 } }],
  });
  installWailsMock({ AttachmentService: { GetList: list, Upload: upload } });
  renderWithApp(<Harness />);
  await waitFor(() => expect(list).toHaveBeenCalledTimes(1));
  fireEvent.click(screen.getByRole('button', { name: 'upload' }));
  expect(await screen.findByText('saved.txt')).toBeInTheDocument();
  expect(list).toHaveBeenCalledTimes(2);
  expect(upload).toHaveBeenCalledTimes(1);
});

test('does not report success when the picker is cancelled', async () => {
  const list = vi.fn().mockResolvedValue([]);
  const upload = vi.fn().mockResolvedValue({ items: [] });
  installWailsMock({ AttachmentService: { GetList: list, Upload: upload } });
  renderWithApp(<Harness />);
  await waitFor(() => expect(list).toHaveBeenCalledTimes(1));
  fireEvent.click(screen.getByRole('button', { name: 'upload' }));
  await waitFor(() => expect(upload).toHaveBeenCalledTimes(1));
  expect(screen.queryByText('Результат загрузки файлов')).not.toBeInTheDocument();
  expect(list).toHaveBeenCalledTimes(1);
});
