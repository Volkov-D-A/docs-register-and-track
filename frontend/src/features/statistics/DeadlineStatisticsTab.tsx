import React, { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { App, Button, Card, Col, DatePicker, Radio, Row, Select, Spin, Table, Typography } from 'antd';
import { CheckCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { formatAppError } from '../../utils/appError';
import { useLatestRequest } from '../../hooks/useLatestRequest';
import { onServerEvent } from '../../events/serverEvents';
import { DeadlineStatusCard, ReportTotal } from './statisticsShared';
import { chartHeight, reportColumns } from './statisticsConstants';

const DeadlineCharts = lazy(() => import('./DeadlineCharts'));
const { Text } = Typography;
const thisYearRange = (): [dayjs.Dayjs, dayjs.Dayjs] => [dayjs().startOf('year'), dayjs().endOf('year')];

const DeadlineStatisticsTab: React.FC = () => {
  const { message } = App.useApp();
  const [stats, setStats] = useState<any>(null);
  const [filters, setFilters] = useState<any>({ users: [] });
  const [report, setReport] = useState<any>(null);
  const [statsLoading, setStatsLoading] = useState(false);
  const [filtersLoading, setFiltersLoading] = useState(false);
  const [reportLoading, setReportLoading] = useState(false);
  const [range, setRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(thisYearRange());
  const [onlyOverdue, setOnlyOverdue] = useState(false);
  const [user, setUser] = useState<string | undefined>();
  const { run: runLatestStats, invalidate: invalidateStats } = useLatestRequest();
  const { run: runLatestFilters, invalidate: invalidateFilters } = useLatestRequest();
  const { run: runLatestReport, invalidate: invalidateReport } = useLatestRequest();
  const loadOverview = useCallback(async () => {
    setStatsLoading(true);
    setFiltersLoading(true);
    await Promise.all([
      runLatestStats(async () => {
        const { GetDeadlineStatistics } = await import('../../../wailsjs/go/services/StatisticsService');
        return GetDeadlineStatistics();
      }, {
        onSuccess: setStats,
        onError: (err) => message.error(formatAppError(err)),
        onSettled: () => setStatsLoading(false),
      }),
      runLatestFilters(async () => {
        const { GetDeadlineFilterOptions } = await import('../../../wailsjs/go/services/StatisticsService');
        return GetDeadlineFilterOptions();
      }, {
        onSuccess: (nextFilters) => setFilters(nextFilters || { users: [] }),
        onError: (err) => message.error(formatAppError(err)),
        onSettled: () => setFiltersLoading(false),
      }),
    ]);
  }, [message, runLatestFilters, runLatestStats]);
  const loadReport = useCallback(async () => {
    setReportLoading(true);
    await runLatestReport(async () => {
      const { GetDeadlineReport } = await import('../../../wailsjs/go/services/StatisticsService');
      return GetDeadlineReport(range[0].format('YYYY-MM-DD'), range[1].format('YYYY-MM-DD'), onlyOverdue, user || '');
    }, {
      onSuccess: setReport,
      onError: (err) => message.error(formatAppError(err)),
      onSettled: () => setReportLoading(false),
    });
  }, [message, onlyOverdue, range, runLatestReport, user]);
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const stop = onServerEvent((event) => {
      if (!['resync', 'access-changed', 'documents', 'document-changed'].includes(event.topic)) return;
      invalidateStats(); invalidateFilters(); invalidateReport();
      setStats(null); setReport(null); setFilters({ users: [], kinds: [], nomenclature: [] });
      clearTimeout(timer);
      timer = setTimeout(() => { void loadOverview(); void loadReport(); }, 100);
    });
    return () => { stop(); clearTimeout(timer); };
  }, [invalidateStats, invalidateFilters, invalidateReport, loadOverview, loadReport]);

  useEffect(() => { void loadOverview(); }, [loadOverview]);
  useEffect(() => { void loadReport(); }, [loadReport]);
  const monthlyData = (stats?.monthlyTotals || []).flatMap((item: any) => [{ period: item.period, metric: 'Всего', value: item.total }, { period: item.period, metric: 'С нарушением сроков', value: item.overdue }]);
  const monthlyConfig: any = { data: monthlyData, xField: 'period', yField: 'value', colorField: 'metric', group: true, height: chartHeight, autoFit: true };
  const executorConfig: any = { data: stats?.monthlyByExecutor || [], xField: 'period', yField: 'value', colorField: 'categoryName', height: chartHeight, autoFit: true };
  return <Spin spinning={(statsLoading || filtersLoading) && !stats}><div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
    <Row gutter={[16, 16]}><Col xs={24} lg={12} xl={10}><DeadlineStatusCard items={stats?.statusCounts || []} /></Col></Row>
    <Row gutter={[16, 16]}><Suspense fallback={<Col span={24}><Spin /></Col>}><DeadlineCharts monthlyConfig={monthlyConfig} executorConfig={executorConfig} hasMonthlyData={!!monthlyData.length} hasExecutorData={!!stats?.monthlyByExecutor?.length} ratingItems={stats?.overdueRating || []} year={stats?.year || dayjs().year()} /></Suspense></Row>
    <Card title="Отчет за период" variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
      <Row gutter={[12, 12]} align="bottom" style={{ marginBottom: 16 }}>
        <Col xs={24} lg={8}><Text type="secondary">Период</Text><DatePicker.RangePicker value={range} onChange={(dates) => { if (dates?.[0] && dates?.[1]) setRange([dates[0], dates[1]]); }} allowClear={false} style={{ width: '100%', marginTop: 4 }} /></Col>
        <Col xs={24} lg={8}><Text type="secondary">Показатель</Text><Radio.Group value={onlyOverdue ? 'overdue' : 'all'} onChange={(event) => setOnlyOverdue(event.target.value === 'overdue')} optionType="button" buttonStyle="solid" style={{ display: 'flex', marginTop: 4 }} options={[{ label: 'Всего', value: 'all' }, { label: 'С нарушением сроков', value: 'overdue' }]} /></Col>
        <Col xs={24} lg={8}><Button icon={<ReloadOutlined />} onClick={() => { void loadOverview(); void loadReport(); }}>Обновить</Button></Col>
        <Col xs={24} md={8}><Select allowClear showSearch optionFilterProp="label" placeholder="Пользователь" value={user} onChange={setUser} options={filters.users || []} style={{ width: '100%' }} /></Col>
      </Row>
      <Row gutter={[16, 16]}><Col xs={24} md={6}><ReportTotal title="Итого" value={report?.total || 0} icon={<CheckCircleOutlined />} color="#52c41a" /></Col><Col xs={24} md={18}><Table size="small" rowKey="key" loading={reportLoading} columns={reportColumns} dataSource={report?.rows || []} pagination={false} locale={{ emptyText: 'Нет строк за выбранный период. Измените период, показатель или пользователя.' }} /></Col></Row>
    </Card>
  </div></Spin>;
};

export default DeadlineStatisticsTab;
