import React from 'react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import DeadlineControls from '../../src/features/deadlines/DeadlineControls';
import { organizationToday } from '../../src/features/deadlines/deadlineEvents';
import { renderWithApp } from '../componentTestUtils';
const api = vi.hoisted(() => ({ List: vi.fn(), ListExecutors: vi.fn(), Create: vi.fn(), CreateSeries: vi.fn(), Complete: vi.fn(), Update: vi.fn(), GetSeries: vi.fn(), canControl: false, canCreate: false, listeners: [] as Array<(event: { topic: string }) => void> }));
vi.mock('../../wailsjs/go/services/DeadlineControlService', () => api);
vi.mock('../../src/hooks/useDocumentKindAccess', () => ({ useDocumentKindAccess: () => ({ ready: true, hasAction: () => api.canCreate, hasAnyAction: () => api.canControl }) }));
vi.mock('../../src/events/serverEvents', () => ({ onServerEvent: (listener: (event: { topic: string }) => void) => { api.listeners.push(listener); return () => { api.listeners = api.listeners.filter((item) => item !== listener); }; } }));
const control = { id: 'control-1', documentId: 'doc-1', documentKind: 'incoming_letter', subject: 'Проверить ответ', executorId: 'person-1', coExecutorIds: [], status: 'active', dueDate: '2026-10-10', version: 7, canManage: true };
beforeEach(() => {
  api.canCreate = false;
  api.canControl = false;
  api.List.mockResolvedValue({ items: [control], total: 1 });
  api.ListExecutors.mockResolvedValue([{ id: 'person-1', name: 'Исполнитель', isActive: true }]);
  api.Complete.mockResolvedValue({ ...control, status: 'completed' });
});
afterEach(() => { vi.clearAllMocks(); api.listeners = []; });
test('card readers see controls without management actions', async () => {
  api.List.mockResolvedValue({ items: [{ ...control, canManage: false }], total: 1 });
  renderWithApp(<DeadlineControls documentId="doc-1" documentKind="incoming_letter" />);
  expect(await screen.findByText('Проверить ответ')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Изменить' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Исполнен' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Установить контроль' })).not.toBeInTheDocument();
});
test('completion warns about finality and submits the edited actual date with the current version', async () => {
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Исполнен' }));
  expect(await screen.findByText('Завершение контроля нельзя отменить')).toBeInTheDocument();
  const input = screen.getByLabelText('Фактическая дата исполнения');
  expect(input).toHaveValue(organizationToday());
  expect(input).toHaveAttribute('max', organizationToday());
  expect(api.Complete).not.toHaveBeenCalled();
  fireEvent.change(input, { target: { value: '2026-01-01' } });
  fireEvent.click(screen.getByRole('button', { name: 'Завершить окончательно' }));
  await waitFor(() => expect(api.Complete).toHaveBeenCalledWith('control-1', expect.objectContaining({ version: 7, executedOn: '2026-01-01' })));
});
test('access invalidation clears cached controls while the replacement request is pending', async () => {
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  expect(await screen.findByText('Проверить ответ')).toBeInTheDocument();
  api.List.mockReturnValue(new Promise(() => {}));
  act(() => api.listeners.forEach((listener) => listener({ topic: 'resync' })));
  expect(screen.queryByText('Проверить ответ')).not.toBeInTheDocument();
  await waitFor(() => expect(api.List).toHaveBeenCalledTimes(2));
});

test('deadline presets and reset send matching server filters', async () => {
  renderWithApp(<DeadlineControls />);
  await screen.findByText('Проверить ответ');
  fireEvent.click(screen.getByRole('button', { name: 'Сегодня' }));
  await waitFor(() => expect(api.List).toHaveBeenLastCalledWith(expect.objectContaining({ dueFrom: organizationToday(), dueTo: organizationToday(), overdueOnly: false, page: 1 })));
  fireEvent.click(screen.getByRole('button', { name: 'Просроченные' }));
  await waitFor(() => expect(api.List).toHaveBeenLastCalledWith(expect.objectContaining({ dueFrom: '', dueTo: '', overdueOnly: true })));
  fireEvent.click(screen.getByRole('button', { name: 'Сбросить фильтры' }));
  await waitFor(() => expect(api.List).toHaveBeenLastCalledWith(expect.objectContaining({ status: 'active', overdueOnly: false, dueFrom: '', dueTo: '' })));
});

test('ordinary control updates preserve an open completion form', async () => {
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Исполнен' }));
  fireEvent.change(screen.getByLabelText('Комментарий'), { target: { value: 'Подготовлен ответ' } });
  act(() => api.listeners.forEach((listener) => listener({ topic: 'document-changed', resource: 'controls' } as { topic: string })));
  await waitFor(() => expect(api.List).toHaveBeenCalledTimes(2));
  expect(screen.getByLabelText('Комментарий')).toHaveValue('Подготовлен ответ');
});


test('document card shows every state without filters or manual refresh', async () => {
  api.List.mockResolvedValue({ items: [control, { ...control, id: 'completed', subject: 'Завершённый контроль', status: 'completed' }], total: 2 });
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  expect(await screen.findByText('Завершённый контроль')).toBeInTheDocument();
  expect(api.List).toHaveBeenCalledWith(expect.objectContaining({ documentId: 'doc-1', status: '', overdueOnly: false, dueFrom: '', dueTo: '' }));
  expect(screen.queryByRole('region', { name: 'Фильтры контроля сроков' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Сегодня' })).not.toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Обновить' })).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Изменить' })).toHaveTextContent('');
});

test('single create button selects regular control inside the dialog and preserves shared fields', async () => {
  api.canCreate = true;
  renderWithApp(<DeadlineControls documentId="doc-1" documentKind="incoming_letter" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Установить контроль' }));
  expect(screen.queryByRole('button', { name: 'Создать регулярный контроль' })).not.toBeInTheDocument();
  expect(screen.getByLabelText('Срок')).toHaveValue(organizationToday().split('-').reverse().join('.'));
  expect(screen.getByLabelText('Срок')).toHaveAttribute('placeholder', 'ДД.ММ.ГГГГ');
  fireEvent.change(screen.getByLabelText('Предмет контроля'), { target: { value: 'Ежемесячный отчёт' } });
  expect(screen.queryByLabelText('Периодичность')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('radio', { name: 'Регулярный' }));
  expect(screen.getByLabelText('Периодичность')).toBeInTheDocument();
  expect(screen.getByLabelText('Предмет контроля')).toHaveValue('Ежемесячный отчёт');
  fireEvent.click(screen.getByRole('radio', { name: 'Обычный' }));
  expect(screen.queryByLabelText('Периодичность')).not.toBeInTheDocument();
  expect(screen.getByLabelText('Предмет контроля')).toHaveValue('Ежемесячный отчёт');
});


test('editing displays Russian dates and sends ISO dates to the service', async () => {
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Изменить' }));
  const input = screen.getByLabelText('Срок');
  expect(input).toHaveValue('10.10.2026');
  fireEvent.change(input, { target: { value: '25.10.2026' } });
  fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' });
  fireEvent.blur(input);
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  await waitFor(() => expect(api.Update).toHaveBeenCalledWith('control-1', expect.objectContaining({ dueDate: '2026-10-25', version: 7 })));
});


test('standalone controls retain the old heading and expandable period selector', async () => {
  renderWithApp(<DeadlineControls />);
  expect(screen.getByRole('heading', { name: 'Контроль сроков' })).toBeInTheDocument();
  const period = screen.getByRole('button', { name: 'Выбрать период' });
  expect(period).toHaveAttribute('aria-expanded', 'false');
  fireEvent.click(period);
  expect(period).toHaveAttribute('aria-expanded', 'true');
  expect(await screen.findByPlaceholderText('Срок с')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Сегодня' }));
  expect(period).toHaveAttribute('aria-expanded', 'false');
  expect(screen.queryByPlaceholderText('Срок с')).not.toBeInTheDocument();
});


test('stopped series filter belongs to control mode and resets on returning to my deadlines', async () => {
  api.canControl = true;
  renderWithApp(<DeadlineControls />);
  expect(screen.queryByRole('button', { name: 'Остановленные серии' })).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'контроль' }));
  fireEvent.click(screen.getByRole('button', { name: 'Остановленные серии' }));
  await waitFor(() => expect(api.List).toHaveBeenLastCalledWith(expect.objectContaining({ manageOnly: true, mineOnly: false, seriesStoppedOnly: true, status: '' })));
  fireEvent.click(screen.getByRole('button', { name: 'мои сроки' }));
  expect(screen.queryByRole('button', { name: 'Остановленные серии' })).not.toBeInTheDocument();
  await waitFor(() => expect(api.List).toHaveBeenLastCalledWith(expect.objectContaining({ mineOnly: true, seriesStoppedOnly: false, status: 'active' })));
});

test('document readers see stopped series label while the current control stays active', async () => {
  api.List.mockResolvedValue({ items: [{ ...control, seriesId: 'series-1', canManage: false }], total: 1 });
  api.GetSeries.mockResolvedValue({ id: 'series-1', active: false });
  renderWithApp(<DeadlineControls documentId="doc-1" />);
  expect(await screen.findByText('Серия остановлена')).toBeInTheDocument();
  expect(screen.getByText('Активен')).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: 'Остановленные серии' })).not.toBeInTheDocument();
});


test.each(['Обычный', 'Регулярный'])('creation of %s control requires a nonblank subject', async (type) => {
  api.canCreate = true;
  renderWithApp(<DeadlineControls documentId="doc-1" documentKind="incoming_letter" />);
  fireEvent.click(await screen.findByRole('button', { name: 'Установить контроль' }));
  fireEvent.click(screen.getByRole('radio', { name: type }));
  fireEvent.change(screen.getByLabelText('Предмет контроля'), { target: { value: '   ' } });
  fireEvent.click(screen.getByRole('button', { name: 'Сохранить' }));
  expect(await screen.findByText('Укажите предмет контроля')).toBeInTheDocument();
  expect(api.Create).not.toHaveBeenCalled();
  expect(api.CreateSeries).not.toHaveBeenCalled();
});


test('controls do not poll every minute but reload when the organization day changes', async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-10-19T18:57:00Z'));
  api.List.mockClear();
  let unmount = () => {};
  try {
    await act(async () => { ({ unmount } = renderWithApp(<DeadlineControls documentId="doc-1" />)); });
    expect(api.List).toHaveBeenCalledTimes(1);
    await act(async () => { vi.advanceTimersByTime(120_000); });
    expect(api.List).toHaveBeenCalledTimes(1);
    await act(async () => { vi.advanceTimersByTime(60_000); });
    expect(api.List).toHaveBeenCalledTimes(2);
    await act(async () => { api.listeners.forEach((listener) => listener({ topic: 'document-changed', resource: 'controls' } as { topic: string })); vi.advanceTimersByTime(100); });
    expect(api.List).toHaveBeenCalledTimes(3);
    await act(async () => { api.listeners.forEach((listener) => listener({ topic: 'document-changed', resource: 'document' } as { topic: string })); vi.advanceTimersByTime(100); });
    expect(api.List).toHaveBeenCalledTimes(4);
  } finally { unmount(); vi.useRealTimers(); }
});
