import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, App, Button, DatePicker, Form, Input, InputNumber, Modal, Radio, Select, Space, Table, Tag, Tooltip, Typography } from 'antd';
import { CalendarOutlined, ClearOutlined, CheckOutlined, CloseOutlined, EditOutlined, HistoryOutlined, PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import 'dayjs/locale/ru';
import ruDatePicker from 'antd/es/date-picker/locale/ru_RU';
import { models } from '../../../wailsjs/go/models';
import * as service from '../../../wailsjs/go/services/DeadlineControlService';
import { useDocumentKindAccess } from '../../hooks/useDocumentKindAccess';
import { useOrganizationDay } from '../../hooks/useOrganizationDay';
import { useAuthStore } from '../../store/useAuthStore';
import { onServerEvent } from '../../events/serverEvents';
import { formatAppError, normalizeAppError } from '../../utils/appError';
import { CoalescedRequest } from '../../utils/coalescedRequest';
import { addCalendarDays, deadlinesChanged, onDeadlinesChanged, type DeadlineView } from './deadlineEvents';

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
  const [periodOpen, setPeriodOpen] = useState(false);
  const periodButton = useRef<HTMLButtonElement>(null);
  const [pageSize, setPageSize] = useState(20);
  const [page, setPage] = useState(1);
  const [result, setResult] = useState<models.DeadlineControlPage>();
  const [stoppedSeries, setStoppedSeries] = useState<string[]>([]);
  const [executors, setExecutors] = useState<models.DeadlineExecutor[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editor, setEditor] = useState<Editor>();
  const [series, setSeries] = useState<models.DocumentDeadlineSeries>();
  const [form] = Form.useForm<FormValues>();
  const request = useRef(new CoalescedRequest<{ list: models.DeadlineControlPage; executors: models.DeadlineExecutor[]; stoppedSeries: string[] }>());
  const today = useOrganizationDay();
  const canCreate = ready && !!documentId && hasAction(documentKind, 'control_deadlines');
  const canSeeQueue = ready && hasAnyAction('control_deadlines');
  const queue = !mine && canSeeQueue;
  const activeView = view === 'stopped' && !queue ? 'active' : view;
  const selectedExecutor = Form.useWatch('executorId', form);
  const intervalUnit = Form.useWatch('intervalUnit', form);
  const dayRule = Form.useWatch('dayRule', form);
  const isSeriesEditor = editor?.mode === 'series-create' || editor?.mode === 'series-edit';
  const isDetailsEditor = editor?.mode === 'create' || editor?.mode === 'edit' || isSeriesEditor;
  const load = useCallback(async () => {
    setLoading(true);
    await request.current.refresh(async () => {
      const [list, people] = await Promise.all([
        service.List(models.DeadlineControlFilter.createFrom({ documentId, seriesId, page, pageSize, mineOnly: mine, manageOnly: !documentId && !seriesId && !mine, status: documentId ? '' : activeView === 'stopped' ? '' : activeView, seriesStoppedOnly: !documentId && queue && activeView === 'stopped', overdueOnly: !documentId && overdue, dueFrom: documentId ? '' : dueFrom, dueTo: documentId ? '' : dueTo })),
        service.ListExecutors(true),
      ]);
      const seriesIDs = documentId ? [...new Set(list.items.map((item) => item.seriesId).filter((id): id is string => !!id))] : [];
      const templates = await Promise.all(seriesIDs.map((id) => service.GetSeries(id)));
      return { list, executors: people, stoppedSeries: templates.filter((template) => !template.active).map((template) => template.id) };
    }, { onSuccess: ({ list, executors: people, stoppedSeries }) => { setResult(list); setExecutors(people); setStoppedSeries(stoppedSeries); }, onError: (error) => { setResult(undefined); setExecutors([]); setStoppedSeries([]); message.error(formatAppError(error)); }, onSettled: () => setLoading(false) });
  }, [documentId, seriesId, page, pageSize, mine, activeView, queue, overdue, dueFrom, dueTo, message]);
  useEffect(() => {
    const current = request.current;
    setResult(undefined); void load();
    return () => current.invalidate();
  }, [load, userId, today]);
  useEffect(() => {
    const stopLocal = onDeadlinesChanged(() => { void load(); });
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stopServer = onServerEvent((event) => {
      if (['resync', 'access-changed', 'documents', 'document-changed'].includes(event.topic)) {
        if (event.topic === 'resync' || event.topic === 'access-changed') { setResult(undefined); setEditor(undefined); setSeries(undefined); }
        clearTimeout(timer); timer = setTimeout(() => { void load(); }, 100);
      }
    });
    return () => { stopLocal(); stopServer(); clearTimeout(timer); };
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
  const setDeadline = (preset: string) => {
    setPeriodOpen(false);
    setOverdue(preset === 'overdue');
    setDueFrom(preset === 'today' || preset === 'soon' ? today : '');
    setDueTo(preset === 'today' ? today : preset === 'soon' ? addCalendarDays(today, 3) : '');
    setPage(1);
  };
  return <>
    {canCreate && <Button style={{ marginBottom: 16 }} type="primary" icon={<PlusOutlined aria-hidden />} onClick={() => openEditor('create')}>Установить контроль</Button>}
    {!documentId && !seriesId && <div className="deadline-page-heading"><div className="workspace-heading">
      <Typography.Title level={4} style={{ margin: 0 }}>Контроль сроков</Typography.Title>
      {canSeeQueue && <span className="workspace-modes" role="group" aria-label="Режим контроля сроков">
        {'( '}<Button type="link" size="small" aria-pressed={mine} className={`workspace-mode-link${mine ? ' workspace-mode-link--active' : ''}`} onClick={() => { setMine(true); if (view === 'stopped') setView('active'); setPage(1); }}>мои сроки</Button>
        {' / '}<Button type="link" size="small" aria-pressed={!mine} className={`workspace-mode-link${!mine ? ' workspace-mode-link--active' : ''}`} onClick={() => { setMine(false); setPage(1); }}>контроль</Button>{' )'}
      </span>}
    </div></div>}
    {!documentId && !seriesId && <section className="deadline-filters" aria-label="Фильтры контроля сроков">
      <div className="deadline-filters-toggles" role="group" aria-label="Состояние контроля">
        {statuses.filter((option) => option.value !== 'stopped' || queue).map((option) => <Button key={option.value} className={`deadline-filter-button deadline-filter-button--${option.value === 'active' ? 'new' : option.value === 'completed' ? 'in_progress' : option.value === 'stopped' ? 'returned' : 'neutral'}`} icon={activeView === option.value ? <CheckOutlined aria-hidden /> : undefined} aria-pressed={activeView === option.value} onClick={() => { setView(option.value); setPage(1); }}>{option.label}</Button>)}
        <Button className="deadline-filters-reset" type="text" icon={<ClearOutlined aria-hidden />} onClick={() => { setView('active'); setDeadline('all'); }}>Сбросить фильтры</Button>
      </div>
      <div className="deadline-filter-row" role="group" aria-label="Срок контроля">
        <span className="deadline-filter-label">Срок</span>
        {([{ value: 'all', label: 'Любой' }, { value: 'overdue', label: 'Просроченные' }, { value: 'today', label: 'Сегодня' }, { value: 'soon', label: 'До 3 дней' }]).map((option) => <Button key={option.value} className="deadline-date-link" type="link" aria-pressed={option.value === 'overdue' ? overdue : option.value === 'all' ? !overdue && !dueFrom && !dueTo : !overdue && dueFrom === today && dueTo === (option.value === 'today' ? today : addCalendarDays(today, 3))} onClick={() => setDeadline(option.value)}>{option.label}</Button>)}
        <Button ref={periodButton} className="deadline-period-button" icon={<CalendarOutlined aria-hidden />} aria-expanded={periodOpen} aria-pressed={!!dueFrom && !!dueTo && !(dueFrom === today && (dueTo === today || dueTo === addCalendarDays(today, 3)))} onClick={() => setPeriodOpen(!periodOpen)}>
          {dueFrom && dueTo ? `${dayjs(dueFrom).format('DD.MM.YYYY')} — ${dayjs(dueTo).format('DD.MM.YYYY')}` : 'Выбрать период'}
        </Button>
        {periodOpen && <DatePicker.RangePicker className="deadline-period-picker" locale={ruDatePicker} autoFocus open aria-label="Период контроля" format="DD.MM.YYYY" placeholder={['Срок с', 'Срок по']} value={dueFrom && dueTo ? [dayjs(dueFrom).locale('ru'), dayjs(dueTo).locale('ru')] : null}
          onOpenChange={(open) => { if (!open) { setPeriodOpen(false); periodButton.current?.focus(); } }}
          onChange={(dates) => { setDueFrom(dates?.[0]?.format('YYYY-MM-DD') || ''); setDueTo(dates?.[1]?.format('YYYY-MM-DD') || ''); setOverdue(false); setPage(1); setPeriodOpen(false); periodButton.current?.focus(); }} />}
      </div>
    </section>}
    <Table<models.DocumentDeadlineControl> className={!documentId && !seriesId ? 'deadline-controls-table' : undefined} size="small" tableLayout="fixed" scroll={{ x: 1000 }} rowKey="id" loading={loading}
      rowClassName={(item) => item.status === 'active' && item.dueDate < today ? 'deadline-overdue' : ''}
      onRow={(item) => ({ tabIndex: onOpenDocument ? 0 : undefined, style: { cursor: onOpenDocument ? 'pointer' : undefined }, onClick: (event) => { if (!(event.target as HTMLElement).closest('button, a, input, [role="button"]')) onOpenDocument?.(item.documentId, item.documentKind); }, onKeyDown: (event) => { if (event.target === event.currentTarget && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); onOpenDocument?.(item.documentId, item.documentKind); } } })}
      expandable={{ rowExpandable: (item) => !!item.comment || !!item.cancellationReason, expandedRowRender: (item) => <Typography.Paragraph>{item.cancellationReason ? `Причина отмены: ${item.cancellationReason}` : `Комментарий: ${item.comment}`}</Typography.Paragraph> }} dataSource={result?.items || []} pagination={{ current: page, total: result?.total || 0, pageSize, showSizeChanger: true, pageSizeOptions: [10, 20, 50], onChange: (value, size) => { setPage(value); setPageSize(size); } }} columns={[
      ...(!documentId ? [{ title: 'Документ', render: (_value: unknown, item: models.DocumentDeadlineControl) => <Button type="link" onClick={() => onOpenDocument?.(item.documentId, item.documentKind)}>{item.documentNumber || 'Без номера'}</Button> }] : []),
      { title: 'Предмет', render: (_value, item) => <>{item.subject}{item.seriesId && stoppedSeries.includes(item.seriesId) && <div><Tag color="orange">Серия остановлена</Tag></div>}</> },
      { title: 'Исполнители', render: (_value, item) => <>{name(item.executorId)}{item.coExecutorIds?.length > 0 && <Typography.Paragraph type="secondary">Соисполнители: {item.coExecutorIds.map(name).join(', ')}</Typography.Paragraph>}</> },
      { title: 'Срок', render: (_value, item) => <><Typography.Text type={item.status === 'active' && item.dueDate < today ? 'danger' : undefined}>{dayjs(item.dueDate).format('DD.MM.YYYY')}</Typography.Text>{item.executedOn && <div>Исполнено: {dayjs(item.executedOn).format('DD.MM.YYYY')}</div>}</> },
      { title: 'Состояние', width: 120, render: (_value, item) => <Tag color={item.status === 'completed' ? 'green' : item.status === 'cancelled' ? 'default' : 'blue'}>{statusLabel[item.status]}</Tag> },
      { title: 'Действия', width: 150, render: (_value, item) => <Space wrap>
        {item.canManage && item.status === 'active' && <><Tooltip title="Изменить"><Button type="text" aria-label="Изменить" icon={<EditOutlined />} onClick={() => openEditor('edit', item)} /></Tooltip><Tooltip title="Исполнен"><Button type="text" aria-label="Исполнен" icon={<CheckOutlined />} onClick={() => openEditor('complete', item)} /></Tooltip><Tooltip title="Отменить"><Button type="text" danger aria-label="Отменить" icon={<CloseOutlined />} onClick={() => openEditor('cancel', item)} /></Tooltip></>}
        {item.seriesId && !seriesId && <Tooltip title={`Серия · ${item.iteration}`}><Button type="text" aria-label={`Серия · ${item.iteration}`} icon={<HistoryOutlined />} onClick={() => void openSeries(item.seriesId || '')} /></Tooltip>}
      </Space> },
    ]} />
    <Modal open={!!editor} title={editor?.mode === 'complete' ? 'Окончательно завершить контроль' : editor?.mode === 'cancel' ? 'Отменить контроль' : isSeriesEditor ? 'Регулярный контроль' : 'Контроль срока'} onCancel={() => { if (!saving) setEditor(undefined); }} onOk={() => form.submit()} okText={editor?.mode === 'complete' ? 'Завершить окончательно' : 'Сохранить'} confirmLoading={saving} destroyOnHidden>
      {editor?.mode === 'complete' && <Alert showIcon type="warning" title="Завершение контроля нельзя отменить" description={editor.control?.seriesId ? 'Если серия активна, будет создан следующий контроль.' : 'Вернуть контроль в активное состояние будет невозможно.'} style={{ marginBottom: 16 }} />}
      {editor?.mode === 'cancel' && editor.control?.seriesId && <Alert type="warning" title="Отмена текущей итерации остановит серию" style={{ marginBottom: 16 }} />}
      {editor?.mode === 'series-edit' && <Alert type="info" title="Изменения относятся к будущим итерациям. Текущий контроль изменяется отдельно." style={{ marginBottom: 16 }} />}
      <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
        {(editor?.mode === 'create' || editor?.mode === 'series-create') && <Form.Item label="Тип контроля">
          <Radio.Group value={editor.mode} onChange={(event) => setEditor({ ...editor, mode: event.target.value })} optionType="button" buttonStyle="solid" options={[{ value: 'create', label: 'Обычный' }, { value: 'series-create', label: 'Регулярный' }]} disabled={saving} />
        </Form.Item>}
        {isDetailsEditor && <>
          <Form.Item name="subject" label="Предмет контроля" rules={[{ required: true, whitespace: true, message: 'Укажите предмет контроля' }]}><Input.TextArea maxLength={2000} /></Form.Item>
          <Form.Item name="executorId" label="Исполнитель" rules={[{ required: true }]}><Select showSearch optionFilterProp="label" options={executors.map((item) => ({ value: item.id, label: item.name + (item.institution ? ` · ${item.institution}` : ''), disabled: !item.isActive }))} /></Form.Item>
          <Form.Item name="coExecutorIds" label="Соисполнители"><Select mode="multiple" showSearch optionFilterProp="label" options={executors.filter((item) => item.id !== selectedExecutor).map((item) => ({ value: item.id, label: item.name, disabled: !item.isActive }))} /></Form.Item>
          <Form.Item name="dueDate" label={isSeriesEditor ? 'Первый плановый срок' : 'Срок'} rules={[{ required: true }]} getValueProps={(value: string) => ({ value: value ? dayjs(value).locale('ru') : null })} normalize={(value: dayjs.Dayjs | null) => value?.format('YYYY-MM-DD') || ''}><DatePicker locale={ruDatePicker} format="DD.MM.YYYY" placeholder="ДД.ММ.ГГГГ" disabled={editor?.mode === 'series-edit'} /></Form.Item>
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
