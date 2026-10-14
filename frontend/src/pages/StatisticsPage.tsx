import React, { lazy, Suspense } from 'react';
import { Spin, Tabs, Typography } from 'antd';

const DocumentStatisticsTab = lazy(() => import('../features/statistics/DocumentStatisticsTab'));
const DeadlineStatisticsTab = lazy(() => import('../features/statistics/DeadlineStatisticsTab'));
const tabFallback = <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}><Spin size="large" /></div>;

const StatisticsPage: React.FC = () => <div style={{ padding: 24 }}>
  <Typography.Title level={3} style={{ marginTop: 0 }}>Статистика</Typography.Title>
  <Tabs destroyOnHidden items={[
    { key: 'documents', label: 'Документы', children: <Suspense fallback={tabFallback}><DocumentStatisticsTab /></Suspense> },
    { key: 'controls', label: 'Контроль сроков', children: <Suspense fallback={tabFallback}><DeadlineStatisticsTab /></Suspense> },
  ]} />
</div>;

export default StatisticsPage;
