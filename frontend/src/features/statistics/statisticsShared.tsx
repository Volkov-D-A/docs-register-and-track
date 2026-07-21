import React from 'react';
import { Card, Empty, Statistic, Table, Typography } from 'antd';
import {
  BarChartOutlined, CheckCircleOutlined, ClockCircleOutlined, FileDoneOutlined,
  FileTextOutlined, InboxOutlined, MessageOutlined, PlayCircleOutlined,
  RollbackOutlined, SendOutlined, StopOutlined, FileProtectOutlined,
} from '@ant-design/icons';

const { Text } = Typography;

export const ChartCard = ({ title, children, isEmpty = false, emptyDescription = 'Нет данных за выбранный период. Измените фильтры или обновите отчет.' }: any) => (
  <Card title={title} variant="borderless" style={{ borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    {isEmpty ? <Empty description={emptyDescription} image={Empty.PRESENTED_IMAGE_SIMPLE} /> : children}
  </Card>
);

export const ReportTotal = ({ title, value, icon, color = '#1677ff' }: any) => (
  <div style={{ border: '1px solid var(--app-border)', borderRadius: 8, padding: 16, height: '100%' }}>
    <Statistic title={title} value={value} prefix={<span style={{ color, marginRight: 8 }}>{icon}</span>} />
  </div>
);

const documentKindVisuals: Record<string, { icon: React.ReactNode; color: string }> = {
  incoming_letter: { icon: <InboxOutlined />, color: '#1677ff' },
  outgoing_letter: { icon: <SendOutlined />, color: '#52c41a' },
  citizen_appeal: { icon: <MessageOutlined />, color: '#fa8c16' },
  administrative_order: { icon: <FileProtectOutlined />, color: '#722ed1' },
};

export const DocumentYearCard = ({ year, total, items }: any) => (
  <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 20, flexWrap: 'wrap' }}>
      <Statistic title={`Документы за ${year} год`} value={total} prefix={<span style={{ color: '#1677ff', marginRight: 8 }}><FileTextOutlined /></span>} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
        {(items || []).map((item: any) => {
          const visual = documentKindVisuals[item.key] || { icon: <FileTextOutlined />, color: '#8c8c8c' };
          return <div key={item.key} title={item.name} style={{ width: 68, minHeight: 58, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4, border: '1px solid var(--app-border)', borderRadius: 8, background: 'var(--app-bg-container)' }}>
            <span style={{ color: visual.color, fontSize: 18, lineHeight: 1 }}>{visual.icon}</span>
            <Text strong style={{ fontSize: 16, lineHeight: '18px' }}>{item.count}</Text>
          </div>;
        })}
      </div>
    </div>
  </Card>
);

const assignmentStatusVisuals: Record<string, { icon: React.ReactNode; color: string }> = {
  new: { icon: <ClockCircleOutlined />, color: '#1677ff' }, in_progress: { icon: <PlayCircleOutlined />, color: '#faad14' },
  completed: { icon: <CheckCircleOutlined />, color: '#52c41a' }, finished: { icon: <FileDoneOutlined />, color: '#13c2c2' },
  returned: { icon: <RollbackOutlined />, color: '#fa541c' }, cancelled: { icon: <StopOutlined />, color: '#8c8c8c' },
};

export const AssignmentStatusCard = ({ items }: any) => {
  const total = (items || []).reduce((sum: number, item: any) => sum + (item.count || 0), 0);
  return <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 20, flexWrap: 'wrap' }}>
      <Statistic title="Поручения по статусам" value={total} prefix={<span style={{ color: '#52c41a', marginRight: 8 }}><CheckCircleOutlined /></span>} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
        {(items || []).map((item: any) => {
          const visual = assignmentStatusVisuals[item.key] || { icon: <BarChartOutlined />, color: '#8c8c8c' };
          return <div key={item.key} title={item.name} style={{ width: 68, minHeight: 58, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4, border: '1px solid var(--app-border)', borderRadius: 8, background: 'var(--app-bg-container)' }}>
            <span style={{ color: visual.color, fontSize: 18, lineHeight: 1 }}>{visual.icon}</span><Text strong style={{ fontSize: 16, lineHeight: '18px' }}>{item.count}</Text>
          </div>;
        })}
      </div>
    </div>
  </Card>;
};

const assignmentRatingColumns: any[] = [
  { title: 'Исп.', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: 'Нар.', dataIndex: 'count', key: 'count', width: 44, align: 'right' },
];

export const AssignmentRatingTable = ({ items }: any) => <Card title="Рейтинг" variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
  <Table size="small" rowKey="key" showHeader={false} columns={assignmentRatingColumns} dataSource={items || []} pagination={false} scroll={{ y: 250 }} locale={{ emptyText: 'Нет нарушений сроков за выбранный период.' }} />
</Card>;
