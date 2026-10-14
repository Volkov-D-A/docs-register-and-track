import React, { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { App, Button, Card, Checkbox, DatePicker, Popconfirm, Select, Space, Spin, Table, Typography } from 'antd';
import { DownloadOutlined, PlayCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { formatAppError } from '../utils/appError';
import { models } from '../../wailsjs/go/models';

type Template = 'organizations' | 'department_load' | 'execution_time' | 'overdue_rate';
const templates = [
  { value: 'organizations', label: 'Входящие и исходящие по организациям' },
  { value: 'department_load', label: 'Нагрузка подразделений по месяцам' },
  { value: 'execution_time', label: 'Среднее время исполнения' },
  { value: 'overdue_rate', label: 'Доля просрочки' },
];
const DepartmentLoadChart = lazy(() => import('../features/reports/DepartmentLoadChart'));

const ReportsPage: React.FC = () => {
  const { message } = App.useApp();
  const [template, setTemplate] = useState<Template>('organizations');
  const [dates, setDates] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs().startOf('year'), dayjs()]);
  const [organizations, setOrganizations] = useState<{ value: string; label: string }[]>([]);
  const [departments, setDepartments] = useState<{ value: string; label: string }[]>([]);
  const [filterOptions, setFilterOptions] = useState<models.ReportFilters>();
  const [organizationId, setOrganizationId] = useState<string>();
  const [departmentId, setDepartmentId] = useState<string>();
  const [kindCode, setKindCode] = useState<string>();
  const [nomenclatureId, setNomenclatureId] = useState<string>();
  const [userId, setUserId] = useState<string>();
  const [status, setStatus] = useState<string>();
  const [onlyWithDeadline, setOnlyWithDeadline] = useState(false);
  const [result, setResult] = useState<models.ReportResult>();
  const [loading, setLoading] = useState(false);
  const [schedules, setSchedules] = useState<models.ReportSchedule[]>([]);
  const [frequency, setFrequency] = useState<'weekly' | 'monthly'>('monthly');
  const [scheduleFormat, setScheduleFormat] = useState<'xlsx' | 'pdf'>('xlsx');
  const [selectedSchedule, setSelectedSchedule] = useState<string>();
  const [runs, setRuns] = useState<models.ReportRun[]>([]);
	const generation = useRef(0);
	const invalidateResult = () => { generation.current += 1; setResult(undefined); };

  const loadSchedules = useCallback(async () => {
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      setSchedules(await service.ListSchedules());
    } catch (error) { message.error(formatAppError(error)); }
  }, [message]);
  useEffect(() => { void loadSchedules(); }, [loadSchedules]);

  const loadRuns = async (scheduleId: string) => {
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      setRuns(await service.ListRuns(scheduleId));
    } catch (error) { message.error(formatAppError(error)); }
  };

  useEffect(() => {
    let active = true;
    void Promise.all([
      import('../../wailsjs/go/services/ReferenceService').then((service) => service.GetOrganizations()),
      import('../../wailsjs/go/services/DepartmentService').then((service) => service.GetAllDepartments()),
      import('../../wailsjs/go/services/ReportingService').then((service) => service.GetFilters()),
    ]).then(([orgs, depts, options]) => {
      if (!active) return;
      setOrganizations(orgs.map((item) => ({ value: item.id, label: item.name })));
      setDepartments(depts.map((item) => ({ value: item.id, label: item.name })));
      setFilterOptions(options);
    }).catch((error) => message.error(formatAppError(error)));
    return () => { active = false; };
  }, [message]);

  const request = useMemo(() => models.ReportRequest.createFrom({
    template, startDate: dates[0].format('YYYY-MM-DD'), endDate: dates[1].format('YYYY-MM-DD'),
    organizationId: template === 'organizations' ? organizationId || '' : '',
    departmentId: template !== 'organizations' ? departmentId || '' : '',
    kindCode: kindCode || '', nomenclatureId: nomenclatureId || '', userId: userId || '',
    status: template === 'organizations' ? '' : status || '',
    onlyWithDeadline: template !== 'organizations' && template !== 'overdue_rate' && onlyWithDeadline,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
  }), [template, dates, organizationId, departmentId, kindCode, nomenclatureId, userId, status, onlyWithDeadline]);

  const run = async () => {
    const currentGeneration = ++generation.current;
    setLoading(true);
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      const next = await service.Run(request);
      if (generation.current === currentGeneration) setResult(next);
    } catch (error) { message.error(formatAppError(error)); }
    finally { setLoading(false); }
  };

  const exportFile = async (format: 'xlsx' | 'pdf') => {
    setLoading(true);
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      const path = await service.Export(request, format);
      message.success(`Отчёт сохранён: ${path}`);
    } catch (error) { message.error(formatAppError(error)); }
    finally { setLoading(false); }
  };

  const createSchedule = async () => {
    setLoading(true);
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      await service.CreateSchedule(models.ReportScheduleRequest.createFrom({
        report: request, frequency, format: scheduleFormat,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Yekaterinburg',
      }));
      message.success('Расписание создано');
      await loadSchedules();
    } catch (error) { message.error(formatAppError(error)); }
    finally { setLoading(false); }
  };

  const disableSchedule = async (id: string) => {
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      await service.DisableSchedule(id);
      await loadSchedules();
    } catch (error) { message.error(formatAppError(error)); }
  };

  const downloadRun = async (id: string) => {
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      const path = await service.DownloadRun(id);
      message.success(`Отчёт сохранён: ${path}`);
    } catch (error) { message.error(formatAppError(error)); }
  };
  const retryRun = async (id: string) => {
    try {
      const service = await import('../../wailsjs/go/services/ReportingService');
      await service.RetryRun(id);
      if (selectedSchedule) await loadRuns(selectedSchedule);
    } catch (error) { message.error(formatAppError(error)); }
  };

  const columns = useMemo(() => {
    const name = { title: template === 'organizations' ? 'Организация' : 'Подразделение', dataIndex: 'name', key: 'name' };
    if (template === 'organizations') return [name,
      { title: 'Входящие', dataIndex: 'incoming', key: 'incoming' },
      { title: 'Исходящие', dataIndex: 'outgoing', key: 'outgoing' }];
    if (template === 'department_load') return [
      { title: 'Месяц', dataIndex: 'period', key: 'period' }, name,
      { title: 'Поручений', dataIndex: 'count', key: 'count' }];
    if (template === 'execution_time') return [name,
      { title: 'Поручений', dataIndex: 'count', key: 'count' },
      { title: 'Среднее, суток', dataIndex: 'metric', key: 'metric' }];
    return [name,
      { title: 'Поручений со сроком', dataIndex: 'count', key: 'count' },
      { title: 'С просрочкой', dataIndex: 'overdue', key: 'overdue' },
      { title: 'Доля, %', dataIndex: 'metric', key: 'metric' }];
  }, [template]);

  return <div style={{ padding: 24 }}>
    <Typography.Title level={3} style={{ marginTop: 0 }}>Отчёты</Typography.Title>
    <Card title="Конструктор отчёта">
      <Space wrap style={{ marginBottom: 16 }}>
        <Select aria-label="Вид отчёта" value={template} options={templates} style={{ width: 340 }} onChange={(value) => {
          setTemplate(value);
          if (value === 'organizations' && kindCode && kindCode !== 'incoming_letter' && kindCode !== 'outgoing_letter') setKindCode(undefined);
          invalidateResult();
        }} />
        <DatePicker.RangePicker value={dates} allowClear={false} onChange={(value) => { if (value?.[0] && value?.[1]) { setDates([value[0], value[1]]); invalidateResult(); } }} />
        {template === 'organizations'
          ? <Select allowClear showSearch optionFilterProp="label" placeholder="Все организации" options={organizations} value={organizationId} onChange={(value) => { setOrganizationId(value); invalidateResult(); }} style={{ width: 230 }} />
          : <Select allowClear showSearch optionFilterProp="label" placeholder="Все подразделения" options={departments} value={departmentId} onChange={(value) => { setDepartmentId(value); invalidateResult(); }} style={{ width: 230 }} />}
        <Button type="primary" icon={<PlayCircleOutlined />} loading={loading} onClick={() => void run()}>Построить</Button>
      </Space>
      <Space wrap style={{ display: 'flex', marginBottom: 16 }}>
        <Select allowClear showSearch optionFilterProp="label" placeholder="Все виды документов" value={kindCode} onChange={(value) => { setKindCode(value); invalidateResult(); }}
          options={(filterOptions?.kinds || []).filter((item) => template !== 'organizations' || item.value === 'incoming_letter' || item.value === 'outgoing_letter')}
          style={{ width: 220 }} />
        <Select allowClear showSearch optionFilterProp="label" placeholder="Все дела" value={nomenclatureId} onChange={(value) => { setNomenclatureId(value); invalidateResult(); }}
          options={filterOptions?.nomenclature || []} style={{ width: 250 }} />
        <Select allowClear showSearch optionFilterProp="label" placeholder={template === 'organizations' ? 'Все регистраторы' : 'Все исполнители'} value={userId} onChange={(value) => { setUserId(value); invalidateResult(); }}
          options={filterOptions?.users || []} style={{ width: 240 }} />
        {template !== 'organizations' && <Select allowClear placeholder="Все статусы" value={status} onChange={(value) => { setStatus(value); invalidateResult(); }} style={{ width: 170 }}
          options={[{ value: 'new', label: 'Новое' }, { value: 'in_progress', label: 'В работе' }, { value: 'completed', label: 'На приёмке' }, { value: 'finished', label: 'Завершено' }, { value: 'returned', label: 'Возвращено' }, { value: 'cancelled', label: 'Отменено' }]} />}
        {template !== 'organizations' && template !== 'overdue_rate' && <Checkbox checked={onlyWithDeadline} onChange={(event) => { setOnlyWithDeadline(event.target.checked); invalidateResult(); }}>Только со сроком</Checkbox>}
      </Space>
      {result && <>
        <Typography.Paragraph type="secondary">{result.definition}</Typography.Paragraph>
        <Typography.Paragraph type="secondary">{result.filters}. Версия определения: {result.version}.</Typography.Paragraph>
        <Typography.Paragraph type="secondary">Период: {result.startDate} — {result.endDate}. Сформировано: {dayjs(result.generatedAt).format('DD.MM.YYYY HH:mm')}. Часовой пояс: {result.timezone}.</Typography.Paragraph>
        <Typography.Paragraph strong>{template === 'organizations'
          ? `По организациям: входящих ${result.summary.incoming}, исходящих ${result.summary.outgoing}; уникальных документов ${result.summary.uniqueDocuments}`
          : template === 'department_load' ? `Всего поручений: ${result.summary.count}`
          : template === 'execution_time' ? `Всего поручений: ${result.summary.count}; среднее: ${result.summary.count ? `${result.summary.metric.toFixed(1)} суток` : 'нет данных'}`
          : `Со сроком: ${result.summary.count}; просрочено: ${result.summary.overdue}; доля: ${result.summary.count ? `${result.summary.metric.toFixed(1)} %` : 'нет данных'}; открытая просрочка: ${result.summary.openOverdue}`}</Typography.Paragraph>
        <Table size="small" rowKey={(row) => `${row.key}:${row.period || ''}`} columns={columns} dataSource={result.rows || []} loading={loading} pagination={{ pageSize: 20 }} />
        {template === 'department_load' && result.rows.length > 0 && <Suspense fallback={<Spin />}><DepartmentLoadChart rows={result.rows} /></Suspense>}
        <Space>
          <Button icon={<DownloadOutlined />} disabled={loading} onClick={() => void exportFile('xlsx')}>Excel</Button>
          <Button icon={<DownloadOutlined />} disabled={loading} onClick={() => void exportFile('pdf')}>PDF</Button>
        </Space>
      </>}
    </Card>
    <Card title="Регулярные отчёты" style={{ marginTop: 24 }}>
      <Typography.Paragraph type="secondary">Используются выбранный вид отчёта и фильтр. Недельный отчёт охватывает прошлую неделю, месячный — прошлый месяц. Запуск в 08:00 по часовому поясу компьютера.</Typography.Paragraph>
      <Space wrap style={{ marginBottom: 16 }}>
        <Select value={frequency} onChange={setFrequency} options={[{ value: 'weekly', label: 'Каждый понедельник' }, { value: 'monthly', label: 'Первого числа месяца' }]} style={{ width: 220 }} />
        <Select value={scheduleFormat} onChange={setScheduleFormat} options={[{ value: 'xlsx', label: 'Excel' }, { value: 'pdf', label: 'PDF' }]} style={{ width: 110 }} />
        <Button onClick={() => void createSchedule()} loading={loading}>Создать расписание</Button>
      </Space>
      <Table rowKey="id" size="small" dataSource={schedules} pagination={false} columns={[
        { title: 'Отчёт', render: (_value, row) => templates.find((item) => item.value === row.request.report.template)?.label || row.request.report.template },
        { title: 'Периодичность', render: (_value, row) => row.request.frequency === 'weekly' ? 'Еженедельно' : 'Ежемесячно' },
        { title: 'Следующий запуск', render: (_value, row) => row.enabled ? dayjs(row.nextRunAt).format('DD.MM.YYYY HH:mm') : 'Отключено' },
        { title: 'Действия', render: (_value, row) => <Space>
          <Button size="small" onClick={() => { setSelectedSchedule(row.id); void loadRuns(row.id); }}>История</Button>
          {row.enabled && <Popconfirm title="Отключить расписание?" onConfirm={() => void disableSchedule(row.id)}><Button size="small" danger>Отключить</Button></Popconfirm>}
        </Space> },
      ]} />
      {selectedSchedule && <Card size="small" title="История запусков" extra={<Button size="small" onClick={() => void loadRuns(selectedSchedule)}>Обновить</Button>} style={{ marginTop: 16 }}>
        <Table rowKey="id" size="small" dataSource={runs} pagination={{ pageSize: 10 }} columns={[
          { title: 'Запуск', render: (_value, row) => dayjs(row.plannedAt).format('DD.MM.YYYY HH:mm') },
          { title: 'Период', render: (_value, row) => `${row.startDate} — ${row.endDate}` },
          { title: 'Статус', dataIndex: 'status' },
          { title: 'Ошибка', dataIndex: 'error' },
          { title: 'Действие', render: (_value, row) => row.status === 'ready'
            ? <Button size="small" onClick={() => void downloadRun(row.id)}>Скачать</Button>
            : row.status === 'failed' ? <Button size="small" onClick={() => void retryRun(row.id)}>Повторить</Button> : null },
        ]} />
      </Card>}
    </Card>
  </div>;
};

export default ReportsPage;
