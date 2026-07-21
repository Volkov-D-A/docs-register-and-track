import React, { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Col, DatePicker, Radio, Row, Select, Spin, Table, Typography } from 'antd';
import { BarChartOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { formatAppError } from '../../utils/appError';
import { DocumentYearCard, ReportTotal } from './statisticsShared';
import { chartHeight, reportColumns } from './statisticsConstants';

const DocumentCharts = lazy(() => import('./DocumentCharts'));
const { Text } = Typography;
const thisYearRange = (): [dayjs.Dayjs, dayjs.Dayjs] => [dayjs().startOf('year'), dayjs().endOf('year')];

const DocumentStatisticsTab: React.FC = () => {
  const { message } = App.useApp();
  const [stats, setStats] = useState<any>(null);
  const [filters, setFilters] = useState<any>({ kinds: [], nomenclature: [], users: [] });
  const [report, setReport] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [reportLoading, setReportLoading] = useState(false);
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(thisYearRange());
  const [groupBy, setGroupBy] = useState('kind');
  const [kind, setKind] = useState<string | undefined>();
  const [nomenclature, setNomenclature] = useState<string | undefined>();
  const [user, setUser] = useState<string | undefined>();

  const loadOverview = useCallback(async () => {
    setLoading(true);
    try {
      const { GetDocumentStatistics, GetDocumentFilterOptions } = await import('../../../wailsjs/go/services/StatisticsService');
      const [nextStats, nextFilters] = await Promise.all([GetDocumentStatistics(), GetDocumentFilterOptions()]);
      setStats(nextStats);
      setFilters(nextFilters || { kinds: [], nomenclature: [], users: [] });
    } catch (err: unknown) { message.error(formatAppError(err)); } finally { setLoading(false); }
  }, [message]);

  const loadReport = useCallback(async () => {
    setReportLoading(true);
    try {
      const { GetDocumentReport } = await import('../../../wailsjs/go/services/StatisticsService');
      setReport(await GetDocumentReport(range[0].format('YYYY-MM-DD'), range[1].format('YYYY-MM-DD'), groupBy, kind || '', nomenclature || '', user || ''));
    } catch (err: unknown) { message.error(formatAppError(err)); } finally { setReportLoading(false); }
  }, [groupBy, kind, message, nomenclature, range, user]);

  useEffect(() => { void loadOverview(); }, [loadOverview]);
  useEffect(() => { void loadReport(); }, [loadReport]);

  const kindTotals = useMemo(() => {
    const totals = new Map<string, { key: string; name: string; count: number }>();
    for (const item of stats?.documentsByKindMonthly || []) {
      const key = item.categoryKey || item.categoryName;
      const current = totals.get(key) || { key, name: item.categoryName || key, count: 0 };
      current.count += item.value || 0;
      totals.set(key, current);
    }
    return Array.from(totals.values()).sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
  }, [stats]);
  const kindConfig: any = { data: stats?.documentsByKindMonthly || [], xField: 'period', yField: 'value', colorField: 'categoryName', group: true, height: chartHeight, autoFit: true };
  const registrarConfig: any = { data: stats?.documentsByRegistrarMonthly || [], xField: 'period', yField: 'value', colorField: 'categoryName', height: chartHeight, autoFit: true };

  return <Spin spinning={loading && !stats}><div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
    <Row gutter={[16, 16]}><Col xs={24} lg={12} xl={10}><DocumentYearCard year={stats?.year || dayjs().year()} total={stats?.totalYear || 0} items={kindTotals} /></Col></Row>
    <Row gutter={[16, 16]}><Suspense fallback={<Col span={24}><Spin /></Col>}><DocumentCharts kindConfig={kindConfig} registrarConfig={registrarConfig} hasKindData={!!stats?.documentsByKindMonthly?.length} hasRegistrarData={!!stats?.documentsByRegistrarMonthly?.length} /></Suspense></Row>
    <Card title="Отчет за период" variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
      <Row gutter={[12, 12]} align="bottom" style={{ marginBottom: 16 }}>
        <Col xs={24} lg={8}><Text type="secondary">Период</Text><DatePicker.RangePicker value={range} onChange={(dates) => { if (dates?.[0] && dates?.[1]) setRange([dates[0], dates[1]]); }} allowClear={false} style={{ width: '100%', marginTop: 4 }} /></Col>
        <Col xs={24} lg={8}><Text type="secondary">Группировка</Text><Radio.Group value={groupBy} onChange={(event) => setGroupBy(event.target.value)} optionType="button" buttonStyle="solid" style={{ display: 'flex', marginTop: 4 }} options={[{ label: 'Виды', value: 'kind' }, { label: 'Дело', value: 'nomenclature' }, { label: 'Пользователи', value: 'user' }]} /></Col>
        <Col xs={24} lg={8}><Button icon={<ReloadOutlined />} onClick={() => { void loadOverview(); void loadReport(); }}>Обновить</Button></Col>
        <Col xs={24} md={8}><Select allowClear placeholder="Вид документа" value={kind} onChange={setKind} options={filters.kinds || []} style={{ width: '100%' }} /></Col>
        <Col xs={24} md={8}><Select allowClear showSearch optionFilterProp="label" placeholder="Дело" value={nomenclature} onChange={setNomenclature} options={filters.nomenclature || []} style={{ width: '100%' }} /></Col>
        <Col xs={24} md={8}><Select allowClear showSearch optionFilterProp="label" placeholder="Пользователь" value={user} onChange={setUser} options={filters.users || []} style={{ width: '100%' }} /></Col>
      </Row>
      <Row gutter={[16, 16]}><Col xs={24} md={6}><ReportTotal title="Итого" value={report?.total || 0} icon={<BarChartOutlined />} color="#13c2c2" /></Col><Col xs={24} md={18}><Table size="small" rowKey="key" loading={reportLoading} columns={reportColumns} dataSource={report?.rows || []} pagination={false} locale={{ emptyText: 'Нет строк за выбранный период. Измените период, группировку или фильтры.' }} /></Col></Row>
    </Card>
  </div></Spin>;
};

export default DocumentStatisticsTab;
