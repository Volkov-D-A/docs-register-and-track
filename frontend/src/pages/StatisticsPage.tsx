import React, { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { Empty, Spin, Tabs, Typography } from 'antd';
import { useAuthStore } from '../store/useAuthStore';

const DocumentStatisticsTab = lazy(() => import('../features/statistics/DocumentStatisticsTab'));
const AssignmentStatisticsTab = lazy(() => import('../features/statistics/AssignmentStatisticsTab'));
const SystemStatisticsTab = lazy(() => import('../features/statistics/SystemStatisticsTab'));
const { Title } = Typography;

const tabFallback = <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}><Spin size="large" /></div>;

const StatisticsPage: React.FC = () => {
  const { hasSystemPermission } = useAuthStore();
  const canViewDocuments = hasSystemPermission('stats_documents');
  const canViewAssignments = hasSystemPermission('stats_assignments');
  const canViewSystem = hasSystemPermission('stats_system');
  const tabs = useMemo(() => [
    ...(canViewDocuments ? [{ key: 'documents', label: 'Документы', component: DocumentStatisticsTab }] : []),
    ...(canViewAssignments ? [{ key: 'assignments', label: 'Поручения', component: AssignmentStatisticsTab }] : []),
    ...(canViewSystem ? [{ key: 'system', label: 'Системная', component: SystemStatisticsTab }] : []),
  ], [canViewAssignments, canViewDocuments, canViewSystem]);
  const [activeTab, setActiveTab] = useState(tabs[0]?.key || '');

  useEffect(() => {
    if (!tabs.some((tab) => tab.key === activeTab)) setActiveTab(tabs[0]?.key || '');
  }, [activeTab, tabs]);

  if (!tabs.length) return <div style={{ padding: 24 }}><Title level={3}>Статистика</Title><Empty description="Статистика недоступна для вашей роли. Обратитесь к администратору, если доступ нужен для работы." /></div>;

  return <div style={{ padding: 24 }}>
    <Title level={3} style={{ marginTop: 0 }}>Статистика</Title>
    <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabs.map((tab) => {
      const TabComponent = tab.component;
      return { key: tab.key, label: tab.label, children: <Suspense fallback={tabFallback}><TabComponent /></Suspense> };
    })} />
  </div>;
};

export default StatisticsPage;
