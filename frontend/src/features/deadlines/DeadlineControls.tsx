import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, App, Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import { models } from '../../../wailsjs/go/models';
import * as service from '../../../wailsjs/go/services/DeadlineControlService';
import { useDocumentKindAccess } from '../../hooks/useDocumentKindAccess';
import { useAuthStore } from '../../store/useAuthStore';
import { onServerEvent } from '../../events/serverEvents';
import { formatAppError, normalizeAppError } from '../../utils/appError';
import { CoalescedRequest } from '../../utils/coalescedRequest';
import { deadlinesChanged, onDeadlinesChanged, organizationToday, type DeadlineView } from './deadlineEvents';

type Props = { documentId?: string; documentKind?: string; seriesId?: string; initialView?: DeadlineView; onOpenDocument?: (id: string, kind: string) => void };
type FormValues = { subject: string; executorId: string; coExecutorIds: string[]; dueDate: string; changeReason: string; executedOn: string; comment: string; reason: string; intervalUnit: string; intervalValue: number; dayRule: string; dayOfMonth: number };
type Editor = { mode: 'create' | 'edit' | 'series-create' | 'series-edit' | 'complete' | 'cancel'; control?: models.DocumentDeadlineControl; series?: models.DocumentDeadlineSeries };
const statuses = [{ value: 'active', label: 'Активные' }, { value: 'completed', label: 'Исполненные' }, { value: 'cancelled', label: 'Отменённые' }, { value: 'stopped', label: 'Остановленные серии' }, { value: '', label: 'Все состояния' }];
const statusLabel: Record<string, string> = { active: 'Активен', completed: 'Исполнен', cancelled: 'Отменён' };

export default function DeadlineControls({ documentId = '', documentKind = '', seriesId = '', initialView, onOpenDocument }: Props) {
  const { message, modal } = App.useApp();
  const userId = useAuthStore((state) => state.user?.id);
  const { ready, hasAction, hasAnyAction } = useDocumentKindAccess();
  const [view, setView] = useState(initialView?.seriesStoppedOnly ? 'stopped' : (initialView?.status ?? 'active'));
  const [mine, setMine] = useState(initialView?.mineOnly ?? (!documentId && !seriesId));
  const [overdue, setOverdue] = useState(initialView?.overdueOnly ?? false);
  const [dueFrom, setDueFrom] = useState(initialView?.dueFrom || '');
  const [dueTo, setDueTo] = useState(initialView?.dueTo || '');
  const [page, setPage] = useState(1);
  const [result, setResult] = useState<models.DeadlineControlPage>();
  const [executors, setExecutors] = useState<models.DeadlineExecutor[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editor, setEditor] = useState<Editor>();
  const [series, setSeries] = useState<models.DocumentDeadlineSeries>();
  const [form] = Form.useForm<FormValues>();
  const request = useRef(new CoalescedRequest<{ list: models.DeadlineControlPage; executors: models.DeadlineExecutor[] }>());
  const today = organizationToday();
  const canCreate = ready && !!documentId && hasAction(documentKind, 'control_deadlines');
  const canSeeQueue = ready && hasAnyAction('control_deadlines');
  const selectedExecutor = Form.useWatch('executorId', form);
  const intervalUnit = Form.useWatch('intervalUnit', form);
  const dayRule = Form.useWatch('dayRule', form);
  const isSeriesEditor = editor?.mode === 'series-create' || editor?.mode === 'series-edit';
  const isDetailsEditor = editor?.mode === 'create' || editor?.mode === 'edit' || isSeriesEditor;
  const load = useCallback(async () => {
    setLoading(true);
    await request.current.refresh(async () => {
      const [list, people] = await Promise.all([
        service.List(models.DeadlineControlFilter.createFrom({ documentId, seriesId, page, pageSize: 20, mineOnly: mine, manageOnly: !documentId && !seriesId && !mine, status: view === 'stopped' ? '' : view, seriesStoppedOnly: view === 'stopped', overdueOnly: overdue, dueFrom, dueTo })),
        service.ListExecutors(true),
      ]);
      return { list, executors: people };
    }, { onSuccess: ({ list, executors: people }) => { setResult(list); setExecutors(people); }, onError: (error) => { setResult(undefined); setExecutors([]); message.error(formatAppError(error)); }, onSettled: () => setLoading(false) });
  }, [documentId, seriesId, page, mine, view, overdue, dueFrom, dueTo, message]);
  useEffect(() => {
    const current = request.current;
    setResult(undefined); void load();
    return () => current.invalidate();
  }, [load, userId]);
  useEffect(() => {
    const stopLocal = onDeadlinesChanged(() => { void load(); });
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stopServer = onServerEvent((event) => {
      if (event.topic === 'resync' || event.topic === 'access-changed' || event.topic === 'documents' || event.resource === 'controls') {
        setResult(undefined); setEditor(undefined); setSeries(undefined);
        clearTimeout(timer); timer = setTimeout(() => { void load(); }, 100);
      }
    });
    const clock = setInterval(() => { void load(); }, 60_000);
    return () => { stopLocal(); stopServer(); clearTimeout(timer); clearInterval(clock); };
  }, [load]);
  const openEditor = (mode: Editor['mode'], control?: models.DocumentDeadlineControl, template?: models.DocumentDeadlineSeries) => {
    form.resetFields();
    form.setFieldsValue({ subject: template?.subject ?? control?.subject ?? '', executorId: template?.executorId ?? control?.executorId ?? '', coExecutorIds: template?.coExecutorIds ?? control?.coExecutorIds ?? [], dueDate: template?.firstDueDate ?? control?.dueDate ?? today, executedOn: today, comment: '', changeReason: '', reason: '', intervalUnit: template?.recurrence.intervalUnit ?? 'month', intervalValue: template?.recurrence.intervalValue ?? 1, dayRule: template?.recurrence.dayRule ?? 'fixed', dayOfMonth: template?.recurrence.dayOfMonth || Number(today.slice(-2)) });
    setEditor({ mode, control, series: template });
  };
  const afterChange = () => { setEditor(undefined); setSeries(undefined); deadlinesChanged(); };
  const save = async (values: FormValues) => {
    if (!editor) return;
    setSaving(true);
    try {
      const control = models.DeadlineControlRequest.createFrom({ documentId: editor.control?.documentId || editor.series?.documentId || documentId, subject: values.subject || '', executorId: values.executorId, coExecutorIds: values.coExecutorIds || [], dueDate: values.dueDate, changeReason: values.changeReason || '', version: editor.control?.version || 0 });
      if (editor.mode === 'create') await service.Create(control);
      if (editor.mode === 'edit' && editor.control) await service.Update(editor.control.id, control);
      if (isSeriesEditor) {
        const monthly = values.intervalUnit === 'month' || values.intervalUnit === 'year';
        const req = models.DeadlineSeriesRequest.createFrom({ control, version: editor.series?.version || 0, recurrence: { intervalUnit: values.intervalUnit, intervalValue: values.intervalValue, dayRule: monthly ? values.dayRule : 'same_day', dayOfMonth: monthly && values.dayRule === 'fixed' ? values.dayOfMonth : 0 } });
        if (editor.series) await service.UpdateSeries(editor.series.id, req); else await service.CreateSeries(req);
      }
      if (editor.mode === 'complete' && editor.control) await service.Complete(editor.control.id, models.CompleteDeadlineRequest.createFrom({ version: editor.control.version, executedOn: values.executedOn, comment: values.comment || '' }));
      if (editor.mode === 'cancel' && editor.control) await service.Cancel(editor.control.id, models.CancelDeadlineRequest.createFrom({ version: editor.control.version, reason: values.reason }));
      message.success('Контроль обновлён'); afterChange();
    } catch (error) {
      message.error(formatAppError(error));
      if (['CONFLICT', 'FORBIDDEN'].includes(normalizeAppError(error).code)) setEditor(undefined);
      await load();
    } finally { setSaving(false); }
  };
  const openSeries = async (id: string) => { try { setSeries(await service.GetSeries(id)); } catch (error) { message.error(formatAppError(error)); } };
  const stopSeries = () => {
    if (!series) return;
    form.resetFields(); form.setFieldsValue({ reason: '' });
    modal.confirm({ title: 'Остановить регулярный контроль?', content: 'Текущий контроль останется активным. Следующая итерация не будет создана.', onOk: async () => { await service.StopSeries(series.id, models.DeadlineSeriesActionRequest.createFrom({ version: series.version, reason: 'Остановлено пользователем', expectedNextDueDate: '' })); afterChange(); } });
  };
  const resumeSeries = async () => {
    if (!series) return;
    try {
      const preview = await service.PreviewResume(series.id);
      modal.confirm({ title: 'Возобновить регулярный контроль?', content: <><p>{preview.createsIteration ? 'Будет создан следующий контроль.' : 'Продолжится текущий контроль.'}</p><p>Срок: <strong>{preview.dueDate}</strong>{preview.dueDate < today ? ' — уже прошёл' : ''}.</p></>, onOk: async () => { await service.ResumeSeries(series.id, models.DeadlineSeriesActionRequest.createFrom({ version: preview.version, expectedNextDueDate: preview.dueDate, reason: '' })); afterChange(); } });
    } catch (error) { message.error(formatAppError(error)); }
  };
  const name = (id: string) => executors.find((executor) => executor.id === id)?.name || id;
  const seriesCanManage = !!series && result?.items.some((item) => item.seriesId === series.id && item.canManage);
  return <>
    <Space wrap style={{ marginBottom: 16 }}>
      {canCreate && <><Button type="primary" onClick={() => openEditor('create')}>Установить контроль</Button><Button onClick={() => openEditor('series-create')}>Создать регулярный контроль</Button></>}
      {!documentId && !seriesId && <Select value={mine ? 'mine' : 'queue'} onChange={(value) => { setMine(value === 'mine'); setPage(1); }} options={[{ value: 'mine', label: 'Мои сроки' }, ...(canSeeQueue ? [{ value: 'queue', label: 'Контроль сроков' }] : [])]} style={{ width: 190 }} />}
      <Select value={view} options={statuses} onChange={(value) => { setView(value); setPage(1); }} style={{ width: 210 }} />
      <Button type={overdue ? 'primary' : 'default'} onClick={() => { setOverdue(!overdue); setPage(1); }}>Просроченные</Button>
      <Input aria-label="Срок с" type="date" value={dueFrom} onChange={(event) => { setDueFrom(event.target.value); setPage(1); }} style={{ width: 155 }} />
      <Input aria-label="Срок по" type="date" value={dueTo} onChange={(event) => { setDueTo(event.target.value); setPage(1); }} style={{ width: 155 }} />
      <Button onClick={() => void load()} loading={loading}>Обновить</Button>
    </Space>
    <Table<models.DocumentDeadlineControl> rowKey="id" loading={loading} dataSource={result?.items || []} pagination={{ current: page, total: result?.total || 0, pageSize: 20, showSizeChanger: false, onChange: setPage }} columns={[
      ...(!documentId ? [{ title: 'Документ', render: (_value: unknown, item: models.DocumentDeadlineControl) => <Button type="link" onClick={() => onOpenDocument?.(item.documentId, item.documentKind)}>{item.documentNumber}</Button> }] : []),
      { title: 'Предмет', dataIndex: 'subject' },
      { title: 'Исполнители', render: (_value, item) => <>{name(item.executorId)}{item.coExecutorIds?.length > 0 && <Typography.Paragraph type="secondary">Соисполнители: {item.coExecutorIds.map(name).join(', ')}</Typography.Paragraph>}</> },
      { title: 'Срок', render: (_value, item) => <><Typography.Text type={item.status === 'active' && item.dueDate < today ? 'danger' : undefined}>{item.dueDate}</Typography.Text>{item.executedOn && <div>Исполнено: {item.executedOn}</div>}</> },
      { title: 'Состояние', render: (_value, item) => <Tag color={item.status === 'completed' ? 'green' : item.status === 'cancelled' ? 'default' : 'blue'}>{statusLabel[item.status]}</Tag> },
      { title: 'Действия', render: (_value, item) => <Space wrap>
        {item.canManage && item.status === 'active' && <><Button onClick={() => openEditor('edit', item)}>Изменить</Button><Button onClick={() => openEditor('complete', item)}>Исполнен</Button><Button danger onClick={() => openEditor('cancel', item)}>Отменить</Button></>}
        {item.seriesId && !seriesId && <Button onClick={() => void openSeries(item.seriesId || '')}>Серия · {item.iteration}</Button>}
      </Space> },
    ]} />
    <Modal open={!!editor} title={editor?.mode === 'complete' ? 'Окончательно завершить контроль' : editor?.mode === 'cancel' ? 'Отменить контроль' : isSeriesEditor ? 'Регулярный контроль' : 'Контроль срока'} onCancel={() => { if (!saving) setEditor(undefined); }} onOk={() => form.submit()} okText={editor?.mode === 'complete' ? 'Завершить окончательно' : 'Сохранить'} confirmLoading={saving} destroyOnHidden>
      {editor?.mode === 'complete' && <Alert showIcon type="warning" title="Завершение контроля нельзя отменить" description={editor.control?.seriesId ? 'Если серия активна, будет создан следующий контроль.' : 'Вернуть контроль в активное состояние будет невозможно.'} style={{ marginBottom: 16 }} />}
      {editor?.mode === 'cancel' && editor.control?.seriesId && <Alert type="warning" title="Отмена текущей итерации остановит серию" style={{ marginBottom: 16 }} />}
      {editor?.mode === 'series-edit' && <Alert type="info" title="Изменения относятся к будущим итерациям. Текущий контроль изменяется отдельно." style={{ marginBottom: 16 }} />}
      <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
        {isDetailsEditor && <>
          <Form.Item name="subject" label="Предмет контроля"><Input.TextArea maxLength={2000} /></Form.Item>
          <Form.Item name="executorId" label="Исполнитель" rules={[{ required: true }]}><Select showSearch optionFilterProp="label" options={executors.map((item) => ({ value: item.id, label: item.name + (item.institution ? ` · ${item.institution}` : ''), disabled: !item.isActive }))} /></Form.Item>
          <Form.Item name="coExecutorIds" label="Соисполнители"><Select mode="multiple" showSearch optionFilterProp="label" options={executors.filter((item) => item.id !== selectedExecutor).map((item) => ({ value: item.id, label: item.name, disabled: !item.isActive }))} /></Form.Item>
          <Form.Item name="dueDate" label={isSeriesEditor ? 'Первый плановый срок' : 'Срок'} rules={[{ required: true }]}><Input type="date" disabled={editor?.mode === 'series-edit'} /></Form.Item>
          {editor?.mode === 'edit' && <Form.Item name="changeReason" label="Причина переноса срока"><Input.TextArea maxLength={2000} /></Form.Item>}
          {isSeriesEditor && <>
            <Form.Item name="intervalUnit" label="Периодичность" rules={[{ required: true }]}><Select options={[{ value: 'day', label: 'Дни' }, { value: 'week', label: 'Недели' }, { value: 'month', label: 'Месяцы' }, { value: 'year', label: 'Годы' }]} /></Form.Item>
            <Form.Item name="intervalValue" label="Интервал" rules={[{ required: true }]}><InputNumber min={1} max={3650} /></Form.Item>
            {(intervalUnit === 'month' || intervalUnit === 'year') && <><Form.Item name="dayRule" label="Календарное правило"><Select options={[{ value: 'fixed', label: 'Заданное число месяца' }, { value: 'last_day', label: 'Последний день месяца' }]} /></Form.Item>{dayRule === 'fixed' && <Form.Item name="dayOfMonth" label="Число месяца" rules={[{ required: true }]}><InputNumber min={1} max={31} /></Form.Item>}</>}
          </>}
        </>}
        {editor?.mode === 'complete' && <><Form.Item name="executedOn" label="Фактическая дата исполнения" rules={[{ required: true }]}><Input type="date" max={today} /></Form.Item><Form.Item name="comment" label="Комментарий"><Input.TextArea maxLength={2000} /></Form.Item></>}
        {editor?.mode === 'cancel' && <Form.Item name="reason" label="Причина отмены" rules={[{ required: true, whitespace: true }]}><Input.TextArea maxLength={2000} /></Form.Item>}
      </Form>
    </Modal>
    <Modal open={!!series} title="Регулярный контроль и история" footer={null} width={1200} onCancel={() => setSeries(undefined)} destroyOnHidden>
      {series && <>
        <Typography.Paragraph>{series.active ? 'Серия активна' : `Серия остановлена: ${series.stopReason}`}. Текущая итерация: {series.currentIteration}.</Typography.Paragraph>
        {seriesCanManage && <Space style={{ marginBottom: 16 }}><Button onClick={() => openEditor('series-edit', undefined, series)}>Изменить шаблон</Button>{series.active ? <Button onClick={stopSeries}>Остановить серию</Button> : <Button onClick={() => void resumeSeries()}>Возобновить серию</Button>}</Space>}
        <DeadlineControls key={series.id} seriesId={series.id} initialView={{ status: '', mineOnly: false }} onOpenDocument={onOpenDocument} />
      </>}
    </Modal>
  </>;
}
