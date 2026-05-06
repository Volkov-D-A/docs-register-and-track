import React, { useEffect, useMemo, useState } from 'react';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Empty,
  Radio,
  Row,
  Select,
  Spin,
  Statistic,
  Table,
  Tabs,
  Typography,
} from 'antd';
import {
  BarChartOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloudOutlined,
  DatabaseOutlined,
  FileDoneOutlined,
  FileTextOutlined,
  FileProtectOutlined,
  HddOutlined,
  InboxOutlined,
  MessageOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  RollbackOutlined,
  SendOutlined,
  StopOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { Column, Line } from '@ant-design/plots';
import dayjs from 'dayjs';
import { useAuthStore } from '../store/useAuthStore';

const { Title, Text } = Typography;

const reportColumns = [
  { title: 'Показатель', dataIndex: 'name', key: 'name' },
  { title: 'Количество', dataIndex: 'count', key: 'count', width: 150 },
];

const assignmentRatingColumns: any[] = [
  { title: 'Исп.', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: 'Нар.', dataIndex: 'count', key: 'count', width: 44, align: 'right' },
];

const thisYearRange = (): [dayjs.Dayjs, dayjs.Dayjs] => [dayjs().startOf('year'), dayjs().endOf('year')];

const chartHeight = 320;

const StatCard = ({ title, value, icon, color = '#1677ff', suffix = '' }: any) => (
  <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    <Statistic
      title={title}
      value={value}
      prefix={<span style={{ color, marginRight: 8 }}>{icon}</span>}
      suffix={suffix}
    />
  </Card>
);

const ChartCard = ({ title, children, isEmpty = false }: any) => (
  <Card title={title} variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    {isEmpty ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} /> : children}
  </Card>
);

const ReportTotal = ({ title, value, icon, color = '#1677ff' }: any) => (
  <div style={{ border: '1px solid var(--app-border)', borderRadius: 8, padding: 16, height: '100%' }}>
    <Statistic
      title={title}
      value={value}
      prefix={<span style={{ color, marginRight: 8 }}>{icon}</span>}
    />
  </div>
);

const documentKindVisuals: Record<string, { icon: React.ReactNode; color: string }> = {
  incoming_letter: { icon: <InboxOutlined />, color: '#1677ff' },
  outgoing_letter: { icon: <SendOutlined />, color: '#52c41a' },
  citizen_appeal: { icon: <MessageOutlined />, color: '#fa8c16' },
  administrative_order: { icon: <FileProtectOutlined />, color: '#722ed1' },
};

const DocumentYearCard = ({ year, total, items }: any) => (
  <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 20, flexWrap: 'wrap' }}>
      <Statistic
        title={`Документы за ${year} год`}
        value={total}
        prefix={<span style={{ color: '#1677ff', marginRight: 8 }}><FileTextOutlined /></span>}
      />
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
        {(items || []).map((item: any) => {
          const visual = documentKindVisuals[item.key] || { icon: <FileTextOutlined />, color: '#8c8c8c' };

          return (
            <div
              key={item.key}
              title={item.name}
              style={{
                width: 68,
                minHeight: 58,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 4,
                border: '1px solid var(--app-border)',
                borderRadius: 8,
                background: 'var(--app-bg-container)',
              }}
            >
              <span style={{ color: visual.color, fontSize: 18, lineHeight: 1 }}>{visual.icon}</span>
              <Text strong style={{ fontSize: 16, lineHeight: '18px' }}>{item.count}</Text>
            </div>
          );
        })}
      </div>
    </div>
  </Card>
);

const assignmentStatusVisuals: Record<string, { icon: React.ReactNode; color: string }> = {
  new: { icon: <ClockCircleOutlined />, color: '#1677ff' },
  in_progress: { icon: <PlayCircleOutlined />, color: '#faad14' },
  completed: { icon: <CheckCircleOutlined />, color: '#52c41a' },
  finished: { icon: <FileDoneOutlined />, color: '#13c2c2' },
  returned: { icon: <RollbackOutlined />, color: '#fa541c' },
  cancelled: { icon: <StopOutlined />, color: '#8c8c8c' },
};

const AssignmentStatusCard = ({ items }: any) => {
  const total = (items || []).reduce((sum: number, item: any) => sum + (item.count || 0), 0);

  return (
    <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 20, flexWrap: 'wrap' }}>
        <Statistic
          title="Поручения по статусам"
          value={total}
          prefix={<span style={{ color: '#52c41a', marginRight: 8 }}><CheckCircleOutlined /></span>}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          {(items || []).map((item: any) => {
            const visual = assignmentStatusVisuals[item.key] || { icon: <BarChartOutlined />, color: '#8c8c8c' };

            return (
              <div
                key={item.key}
                title={item.name}
                style={{
                  width: 68,
                  minHeight: 58,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 4,
                  border: '1px solid var(--app-border)',
                  borderRadius: 8,
                  background: 'var(--app-bg-container)',
                }}
              >
                <span style={{ color: visual.color, fontSize: 18, lineHeight: 1 }}>{visual.icon}</span>
                <Text strong style={{ fontSize: 16, lineHeight: '18px' }}>{item.count}</Text>
              </div>
            );
          })}
        </div>
      </div>
    </Card>
  );
};

const AssignmentRatingTable = ({ items }: any) => (
  <Card title="Рейтинг" variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    <Table
      size="small"
      rowKey="key"
      showHeader={false}
      columns={assignmentRatingColumns}
      dataSource={items || []}
      pagination={false}
      scroll={{ y: 250 }}
    />
  </Card>
);

const StatisticsPage: React.FC = () => {
  const { message } = App.useApp();
  const { hasSystemPermission } = useAuthStore();

  const canViewDocuments = hasSystemPermission('stats_documents');
  const canViewAssignments = hasSystemPermission('stats_assignments');
  const canViewSystem = hasSystemPermission('stats_system');

  const tabItems = useMemo(() => [
    ...(canViewDocuments ? [{ key: 'documents', label: 'Документы' }] : []),
    ...(canViewAssignments ? [{ key: 'assignments', label: 'Поручения' }] : []),
    ...(canViewSystem ? [{ key: 'system', label: 'Системная' }] : []),
  ], [canViewDocuments, canViewAssignments, canViewSystem]);

  const [activeTab, setActiveTab] = useState(tabItems[0]?.key || '');

  const [documentStats, setDocumentStats] = useState<any>(null);
  const [documentFilters, setDocumentFilters] = useState<any>({ kinds: [], nomenclature: [], users: [] });
  const [documentReport, setDocumentReport] = useState<any>(null);
  const [documentLoading, setDocumentLoading] = useState(false);
  const [documentReportLoading, setDocumentReportLoading] = useState(false);
  const [documentRange, setDocumentRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(thisYearRange());
  const [documentGroupBy, setDocumentGroupBy] = useState('kind');
  const [documentKind, setDocumentKind] = useState<string | undefined>();
  const [documentNomenclature, setDocumentNomenclature] = useState<string | undefined>();
  const [documentUser, setDocumentUser] = useState<string | undefined>();

  const [assignmentStats, setAssignmentStats] = useState<any>(null);
  const [assignmentFilters, setAssignmentFilters] = useState<any>({ users: [] });
  const [assignmentReport, setAssignmentReport] = useState<any>(null);
  const [assignmentLoading, setAssignmentLoading] = useState(false);
  const [assignmentReportLoading, setAssignmentReportLoading] = useState(false);
  const [assignmentRange, setAssignmentRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(thisYearRange());
  const [assignmentOnlyOverdue, setAssignmentOnlyOverdue] = useState(false);
  const [assignmentUser, setAssignmentUser] = useState<string | undefined>();

  const [systemStats, setSystemStats] = useState<any>(null);
  const [systemLoading, setSystemLoading] = useState(false);

  useEffect(() => {
    if (!tabItems.some((item) => item.key === activeTab)) {
      setActiveTab(tabItems[0]?.key || '');
    }
  }, [activeTab, tabItems]);

  const loadDocumentOverview = async () => {
    setDocumentLoading(true);
    try {
      const { GetDocumentStatistics, GetDocumentFilterOptions } = await import('../../wailsjs/go/services/StatisticsService');
      const [stats, filters] = await Promise.all([GetDocumentStatistics(), GetDocumentFilterOptions()]);
      setDocumentStats(stats);
      setDocumentFilters(filters || { kinds: [], nomenclature: [], users: [] });
    } catch (err: any) {
      message.error(err?.message || String(err));
    } finally {
      setDocumentLoading(false);
    }
  };

  const loadDocumentReport = async () => {
    setDocumentReportLoading(true);
    try {
      const { GetDocumentReport } = await import('../../wailsjs/go/services/StatisticsService');
      const data = await GetDocumentReport(
        documentRange[0].format('YYYY-MM-DD'),
        documentRange[1].format('YYYY-MM-DD'),
        documentGroupBy,
        documentKind || '',
        documentNomenclature || '',
        documentUser || '',
      );
      setDocumentReport(data);
    } catch (err: any) {
      message.error(err?.message || String(err));
    } finally {
      setDocumentReportLoading(false);
    }
  };

  const loadAssignmentOverview = async () => {
    setAssignmentLoading(true);
    try {
      const { GetAssignmentStatistics, GetAssignmentFilterOptions } = await import('../../wailsjs/go/services/StatisticsService');
      const [stats, filters] = await Promise.all([GetAssignmentStatistics(), GetAssignmentFilterOptions()]);
      setAssignmentStats(stats);
      setAssignmentFilters(filters || { users: [] });
    } catch (err: any) {
      message.error(err?.message || String(err));
    } finally {
      setAssignmentLoading(false);
    }
  };

  const loadAssignmentReport = async () => {
    setAssignmentReportLoading(true);
    try {
      const { GetAssignmentReport } = await import('../../wailsjs/go/services/StatisticsService');
      const data = await GetAssignmentReport(
        assignmentRange[0].format('YYYY-MM-DD'),
        assignmentRange[1].format('YYYY-MM-DD'),
        assignmentOnlyOverdue,
        assignmentUser || '',
      );
      setAssignmentReport(data);
    } catch (err: any) {
      message.error(err?.message || String(err));
    } finally {
      setAssignmentReportLoading(false);
    }
  };

  const loadSystemStats = async () => {
    setSystemLoading(true);
    try {
      const { GetSystemStatistics } = await import('../../wailsjs/go/services/StatisticsService');
      setSystemStats(await GetSystemStatistics());
    } catch (err: any) {
      message.error(err?.message || String(err));
    } finally {
      setSystemLoading(false);
    }
  };

  useEffect(() => {
    if (activeTab === 'documents' && canViewDocuments && !documentStats) {
      void loadDocumentOverview();
    }
    if (activeTab === 'assignments' && canViewAssignments && !assignmentStats) {
      void loadAssignmentOverview();
    }
    if (activeTab === 'system' && canViewSystem && !systemStats) {
      void loadSystemStats();
    }
  }, [activeTab, canViewDocuments, canViewAssignments, canViewSystem]);

  useEffect(() => {
    if (canViewDocuments) {
      void loadDocumentReport();
    }
  }, [canViewDocuments, documentRange, documentGroupBy, documentKind, documentNomenclature, documentUser]);

  useEffect(() => {
    if (canViewAssignments) {
      void loadAssignmentReport();
    }
  }, [canViewAssignments, assignmentRange, assignmentOnlyOverdue, assignmentUser]);

  const documentKindChartConfig: any = {
    data: documentStats?.documentsByKindMonthly || [],
    xField: 'period',
    yField: 'value',
    colorField: 'categoryName',
    group: true,
    height: chartHeight,
    autoFit: true,
  };

  const documentRegistrarChartConfig: any = {
    data: documentStats?.documentsByRegistrarMonthly || [],
    xField: 'period',
    yField: 'value',
    colorField: 'categoryName',
    height: chartHeight,
    autoFit: true,
  };

  const documentKindTotals = useMemo(() => {
    const totals = new Map<string, { key: string; name: string; count: number }>();
    for (const item of documentStats?.documentsByKindMonthly || []) {
      const key = item.categoryKey || item.categoryName;
      const current = totals.get(key) || { key, name: item.categoryName || key, count: 0 };
      current.count += item.value || 0;
      totals.set(key, current);
    }
    return Array.from(totals.values()).sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
  }, [documentStats]);

  const assignmentMonthlyChartData = (assignmentStats?.monthlyTotals || []).flatMap((item: any) => [
    { period: item.period, metric: 'Всего', value: item.total },
    { period: item.period, metric: 'С нарушением сроков', value: item.overdue },
  ]);

  const assignmentMonthlyChartConfig: any = {
    data: assignmentMonthlyChartData,
    xField: 'period',
    yField: 'value',
    colorField: 'metric',
    group: true,
    height: chartHeight,
    autoFit: true,
  };

  const assignmentExecutorChartConfig: any = {
    data: assignmentStats?.monthlyByExecutor || [],
    xField: 'period',
    yField: 'value',
    colorField: 'categoryName',
    height: chartHeight,
    autoFit: true,
  };

  const renderDocumentsTab = () => (
    <Spin spinning={documentLoading && !documentStats}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12} xl={10}>
            <DocumentYearCard
              year={documentStats?.year || dayjs().year()}
              total={documentStats?.totalYear || 0}
              items={documentKindTotals}
            />
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={12}>
            <ChartCard title="Ежемесячно по видам документов" isEmpty={!documentStats?.documentsByKindMonthly?.length}>
              <Column {...documentKindChartConfig} />
            </ChartCard>
          </Col>
          <Col xs={24} xl={12}>
            <ChartCard title="Ежемесячно по зарегистрировавшему пользователю" isEmpty={!documentStats?.documentsByRegistrarMonthly?.length}>
              <Line {...documentRegistrarChartConfig} />
            </ChartCard>
          </Col>
        </Row>

        <Card title="Отчет за период" variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
          <Row gutter={[12, 12]} align="bottom" style={{ marginBottom: 16 }}>
            <Col xs={24} lg={8}>
              <Text type="secondary">Период</Text>
              <DatePicker.RangePicker
                value={documentRange}
                onChange={(dates) => {
                  if (dates?.[0] && dates?.[1]) {
                    setDocumentRange([dates[0], dates[1]]);
                  }
                }}
                allowClear={false}
                style={{ width: '100%', marginTop: 4 }}
              />
            </Col>
            <Col xs={24} lg={8}>
              <Text type="secondary">Группировка</Text>
              <Radio.Group
                value={documentGroupBy}
                onChange={(event) => setDocumentGroupBy(event.target.value)}
                optionType="button"
                buttonStyle="solid"
                style={{ display: 'flex', marginTop: 4 }}
                options={[
                  { label: 'Виды', value: 'kind' },
                  { label: 'Номенклатура', value: 'nomenclature' },
                  { label: 'Пользователи', value: 'user' },
                ]}
              />
            </Col>
            <Col xs={24} lg={8}>
              <Button icon={<ReloadOutlined />} onClick={() => { void loadDocumentOverview(); void loadDocumentReport(); }}>
                Обновить
              </Button>
            </Col>
            <Col xs={24} md={8}>
              <Select allowClear placeholder="Вид документа" value={documentKind} onChange={setDocumentKind} options={documentFilters.kinds || []} style={{ width: '100%' }} />
            </Col>
            <Col xs={24} md={8}>
              <Select allowClear showSearch optionFilterProp="label" placeholder="Номенклатура" value={documentNomenclature} onChange={setDocumentNomenclature} options={documentFilters.nomenclature || []} style={{ width: '100%' }} />
            </Col>
            <Col xs={24} md={8}>
              <Select allowClear showSearch optionFilterProp="label" placeholder="Пользователь" value={documentUser} onChange={setDocumentUser} options={documentFilters.users || []} style={{ width: '100%' }} />
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            <Col xs={24} md={6}>
              <ReportTotal title="Итого" value={documentReport?.total || 0} icon={<BarChartOutlined />} color="#13c2c2" />
            </Col>
            <Col xs={24} md={18}>
              <Table
                size="small"
                rowKey="key"
                loading={documentReportLoading}
                columns={reportColumns}
                dataSource={documentReport?.rows || []}
                pagination={false}
              />
            </Col>
          </Row>
        </Card>
      </div>
    </Spin>
  );

  const renderAssignmentsTab = () => (
    <Spin spinning={assignmentLoading && !assignmentStats}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12} xl={10}>
            <AssignmentStatusCard items={assignmentStats?.statusCounts || []} />
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={10}>
            <ChartCard title={`Поручения по месяцам, ${assignmentStats?.year || dayjs().year()} год`} isEmpty={!assignmentMonthlyChartData.length}>
              <Column {...assignmentMonthlyChartConfig} />
            </ChartCard>
          </Col>
          <Col xs={24} xl={10}>
            <ChartCard title="Ежемесячно по основным исполнителям" isEmpty={!assignmentStats?.monthlyByExecutor?.length}>
              <Line {...assignmentExecutorChartConfig} />
            </ChartCard>
          </Col>
          <Col xs={24} xl={4}>
            <AssignmentRatingTable items={assignmentStats?.overdueRating || []} />
          </Col>
        </Row>

        <Card title="Отчет за период" variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
          <Row gutter={[12, 12]} align="bottom" style={{ marginBottom: 16 }}>
            <Col xs={24} lg={8}>
              <Text type="secondary">Период</Text>
              <DatePicker.RangePicker
                value={assignmentRange}
                onChange={(dates) => {
                  if (dates?.[0] && dates?.[1]) {
                    setAssignmentRange([dates[0], dates[1]]);
                  }
                }}
                allowClear={false}
                style={{ width: '100%', marginTop: 4 }}
              />
            </Col>
            <Col xs={24} lg={8}>
              <Text type="secondary">Показатель</Text>
              <Radio.Group
                value={assignmentOnlyOverdue ? 'overdue' : 'all'}
                onChange={(event) => setAssignmentOnlyOverdue(event.target.value === 'overdue')}
                optionType="button"
                buttonStyle="solid"
                style={{ display: 'flex', marginTop: 4 }}
                options={[
                  { label: 'Всего', value: 'all' },
                  { label: 'С нарушением сроков', value: 'overdue' },
                ]}
              />
            </Col>
            <Col xs={24} lg={8}>
              <Button icon={<ReloadOutlined />} onClick={() => { void loadAssignmentOverview(); void loadAssignmentReport(); }}>
                Обновить
              </Button>
            </Col>
            <Col xs={24} md={8}>
              <Select allowClear showSearch optionFilterProp="label" placeholder="Пользователь" value={assignmentUser} onChange={setAssignmentUser} options={assignmentFilters.users || []} style={{ width: '100%' }} />
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            <Col xs={24} md={6}>
              <ReportTotal title="Итого" value={assignmentReport?.total || 0} icon={<CheckCircleOutlined />} color="#52c41a" />
            </Col>
            <Col xs={24} md={18}>
              <Table
                size="small"
                rowKey="key"
                loading={assignmentReportLoading}
                columns={reportColumns}
                dataSource={assignmentReport?.rows || []}
                pagination={false}
              />
            </Col>
          </Row>
        </Card>
      </div>
    </Spin>
  );

  const renderSystemTab = () => (
    <Spin spinning={systemLoading && !systemStats}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="Пользователи" value={systemStats?.userCount || 0} icon={<UserOutlined />} color="#1677ff" />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="Всего документов" value={systemStats?.totalDocuments || 0} icon={<DatabaseOutlined />} color="#52c41a" />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="База данных" value={systemStats?.dbSize || 'N/A'} icon={<DatabaseOutlined />} color="#13c2c2" />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="Файлы в хранилище" value={systemStats?.storageObjects || 0} icon={<CloudOutlined />} color="#722ed1" />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="Размер хранилища" value={systemStats?.storageSize || 'N/A'} icon={<HddOutlined />} color="#fa8c16" />
        </Col>
      </Row>
    </Spin>
  );

  if (!tabItems.length) {
    return (
      <div style={{ padding: 24 }}>
        <Title level={3}>Статистика</Title>
        <Empty description="Нет доступа к статистике" />
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <Title level={3} style={{ marginTop: 0 }}>Статистика</Title>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={tabItems.map((item) => ({
          ...item,
          children:
            item.key === 'documents' ? renderDocumentsTab()
              : item.key === 'assignments' ? renderAssignmentsTab()
                : renderSystemTab(),
        }))}
      />
    </div>
  );
};

export default StatisticsPage;
