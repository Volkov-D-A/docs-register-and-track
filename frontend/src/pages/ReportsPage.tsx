import React, { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react';
import { App, Button, Card, DatePicker, Select, Space, Spin, Table, Typography } from 'antd';
import { DownloadOutlined, PlayCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { formatAppError } from '../utils/appError';
import { models } from '../../wailsjs/go/models';
import { onServerEvent } from '../events/serverEvents';

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
  const [periodBasis, setPeriodBasis] = useState('registration_date');
  const [result, setResult] = useState<models.ReportResult>();
  const [loading, setLoading] = useState(false);
  const [accessRevision, setAccessRevision] = useState(0);
	const generation = useRef(0);
  const filtersGeneration = useRef(0);
	const invalidateResult = () => { generation.current += 1; setResult(undefined); };

  useEffect(() => {
    const filterVersion = ++filtersGeneration.current;
    void import('../../wailsjs/go/services/ReportingService').then((service) => service.GetFilters()).then((options) => {
      if (filtersGeneration.current !== filterVersion) return;
      setOrganizations(options.organizations);
      setDepartments(options.departments);
      setFilterOptions(options);
    }).catch((error) => message.error(formatAppError(error)));
    return () => { filtersGeneration.current += 1; };
  }, [message, accessRevision]);

  useEffect(() => {
    const stop = onServerEvent((event) => {
      if (!['resync', 'access-changed', 'documents', 'document-changed'].includes(event.topic)) return;
      generation.current += 1; filtersGeneration.current += 1; setResult(undefined); setFilterOptions(undefined);
      setOrganizations([]); setDepartments([]);
      setOrganizationId(undefined); setDepartmentId(undefined); setNomenclatureId(undefined); setUserId(undefined);
      setAccessRevision((version) => version + 1);
    });
    return () => { stop(); generation.current += 1; };
  }, []);

  const request = useMemo(() => models.ReportRequest.createFrom({
    template, startDate: dates[0].format('YYYY-MM-DD'), endDate: dates[1].format('YYYY-MM-DD'),
    organizationId: template === 'organizations' ? organizationId || '' : '',
    departmentId: template !== 'organizations' ? departmentId || '' : '',
    kindCode: kindCode || '', nomenclatureId: nomenclatureId || '', userId: userId || '',
    status: template === 'organizations' ? '' : status || '',
    periodBasis,
    timezone: 'Asia/Yekaterinburg',
  }), [template, dates, organizationId, departmentId, kindCode, nomenclatureId, userId, status, periodBasis]);

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

  const columns = useMemo(() => {
    const name = { title: template === 'organizations' ? 'Организация' : 'Подразделение', dataIndex: 'name', key: 'name' };
    if (template === 'organizations') return [name,
      { title: 'Входящие', dataIndex: 'incoming', key: 'incoming' },
      { title: 'Исходящие', dataIndex: 'outgoing', key: 'outgoing' }];
    if (template === 'department_load') return [
      { title: 'Месяц', dataIndex: 'period', key: 'period' }, name,
      { title: 'Контролей', dataIndex: 'count', key: 'count' }];
    if (template === 'execution_time') return [name,
      { title: 'Контролей', dataIndex: 'count', key: 'count' },
      { title: 'Среднее, суток', dataIndex: 'metric', key: 'metric' }];
    return [name,
      { title: 'Контролей со сроком', dataIndex: 'count', key: 'count' },
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
        <Select aria-label="Основание периода" value={periodBasis} onChange={(value) => { setPeriodBasis(value); invalidateResult(); }} options={[{value:'registration_date',label:'Дата регистрации документа'},{value:'due_date',label:'Срок контроля'},{value:'executed_on',label:'Фактическая дата исполнения'}]} style={{width:260}} />
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
          options={(template === 'organizations' ? filterOptions?.users : filterOptions?.executors) || []} style={{ width: 240 }} />
        {template !== 'organizations' && <Select allowClear placeholder="Все статусы" value={status} onChange={(value) => { setStatus(value); invalidateResult(); }} style={{ width: 170 }}
          options={[{ value: 'active', label: 'Активен' }, { value: 'completed', label: 'Исполнен' }, { value: 'cancelled', label: 'Отменён' }]} />}

      </Space>
      {result && <>
        <Typography.Paragraph type="secondary">{result.definition}</Typography.Paragraph>
        <Typography.Paragraph type="secondary">{result.filters}. Версия определения: {result.version}.</Typography.Paragraph>
        <Typography.Paragraph type="secondary">Период: {result.startDate} — {result.endDate}. Сформировано: {dayjs(result.generatedAt).format('DD.MM.YYYY HH:mm')}. Часовой пояс: {result.timezone}.</Typography.Paragraph>
        <Typography.Paragraph strong>{template === 'organizations'
          ? `По организациям: входящих ${result.summary.incoming}, исходящих ${result.summary.outgoing}; уникальных документов ${result.summary.uniqueDocuments}`
          : template === 'department_load' ? `Всего контролей: ${result.summary.count}`
          : template === 'execution_time' ? `Всего контролей: ${result.summary.count}; среднее: ${result.summary.count ? `${result.summary.metric.toFixed(1)} суток` : 'нет данных'}`
          : `Со сроком: ${result.summary.count}; просрочено: ${result.summary.overdue}; доля: ${result.summary.count ? `${result.summary.metric.toFixed(1)} %` : 'нет данных'}; открытая просрочка: ${result.summary.openOverdue}`}</Typography.Paragraph>
        <Table size="small" rowKey={(row) => `${row.key}:${row.period || ''}`} columns={columns} dataSource={result.rows || []} loading={loading} pagination={{ pageSize: 20 }} />
        {template === 'department_load' && result.rows.length > 0 && <Suspense fallback={<Spin />}><DepartmentLoadChart rows={result.rows} /></Suspense>}
        <Space>
          <Button icon={<DownloadOutlined />} disabled={loading} onClick={() => void exportFile('xlsx')}>Excel</Button>
          <Button icon={<DownloadOutlined />} disabled={loading} onClick={() => void exportFile('pdf')}>PDF</Button>
        </Space>
      </>}
    </Card>
  </div>;
};

export default ReportsPage;
