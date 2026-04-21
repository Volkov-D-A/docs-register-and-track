import React, { useEffect, useState } from 'react';
import { App, Card, Col, DatePicker, Empty, Row, Spin, Statistic, Typography } from 'antd';
import {
  InboxOutlined,
  SendOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  UserOutlined,
  DatabaseOutlined,
  CloudOutlined,
  HddOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { resolveUserProfile, useAuthStore } from '../store/useAuthStore';
import { useDocumentKinds } from '../hooks/useDocumentKinds';

const { Title } = Typography;
const emptyKinds: any[] = [];

const StatisticsPage: React.FC = () => {
  const { message } = App.useApp();
  const { hasSystemPermission, user } = useAuthStore();
  const { kinds: readableKinds } = useDocumentKinds({ mode: 'all', fallbackKinds: emptyKinds });
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs().startOf('month'), dayjs().endOf('month')]);

  const canViewIncoming = hasSystemPermission('stats_incoming');
  const canViewOutgoing = hasSystemPermission('stats_outgoing');
  const canViewAssignments = hasSystemPermission('stats_assignments');
  const canViewSystem = hasSystemPermission('stats_system');

  const loadStats = async () => {
    setLoading(true);
    try {
      const { GetStats } = await import('../../wailsjs/go/services/DashboardService');
      const data = await GetStats('', dateRange[0].format('YYYY-MM-DD'), dateRange[1].format('YYYY-MM-DD'));
      setStats(data);
    } catch (err: any) {
      message.error(err?.message || String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadStats();
  }, [dateRange]);

  const StatCard = ({ title, value, icon, color = '#1677ff', suffix = '' }: any) => (
    <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
      <Statistic
        title={title}
        value={value}
        prefix={<span style={{ color, marginRight: 8 }}>{icon}</span>}
        suffix={suffix}
      />
    </Card>
  );

  const profile = resolveUserProfile(user?.systemPermissions, readableKinds, user?.isDocumentParticipant);
  const hasAnySection = canViewIncoming || canViewOutgoing || canViewAssignments || canViewSystem;

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>Статистика</Title>
        <DatePicker.RangePicker
          value={dateRange}
          onChange={(dates) => {
            if (dates?.[0] && dates?.[1]) {
              setDateRange([dates[0], dates[1]]);
            }
          }}
          allowClear={false}
        />
      </div>

      {loading && !stats ? (
        <div style={{ textAlign: 'center', marginTop: 50 }}><Spin size="large" /></div>
      ) : !hasAnySection ? (
        <Empty description="Нет доступа к статистике" />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
          {(canViewIncoming || canViewOutgoing) && (
            <div>
              <Title level={5}>Документы</Title>
              <Row gutter={[16, 16]}>
                {canViewIncoming && (
                  <Col xs={24} sm={12} lg={8}>
                    <StatCard title="Входящие письма" value={stats?.incomingCount || 0} icon={<InboxOutlined />} color="#69c0ff" />
                  </Col>
                )}
                {canViewOutgoing && (
                  <Col xs={24} sm={12} lg={8}>
                    <StatCard title="Исходящие письма" value={stats?.outgoingCount || 0} icon={<SendOutlined />} color="#95de64" />
                  </Col>
                )}
              </Row>
            </div>
          )}

          {canViewAssignments && (
            <div>
              <Title level={5}>Поручения</Title>
              <Row gutter={[16, 16]}>
                {profile === 'clerk' ? (
                  <>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="Просрочено" value={stats?.allAssignmentsOverdue || 0} icon={<ClockCircleOutlined />} color="#ff7875" />
                    </Col>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="Завершено" value={stats?.allAssignmentsFinished || 0} icon={<CheckCircleOutlined />} color="#95de64" />
                    </Col>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="С опозданием" value={stats?.allAssignmentsFinishedLate || 0} icon={<ClockCircleOutlined />} color="#fa8c16" />
                    </Col>
                  </>
                ) : (
                  <>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="Новые" value={stats?.myAssignmentsNew || 0} icon={<CheckCircleOutlined />} color="#69c0ff" />
                    </Col>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="В работе" value={stats?.myAssignmentsInProgress || 0} icon={<ClockCircleOutlined />} color="#ffc069" />
                    </Col>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="Просрочено" value={stats?.myAssignmentsOverdue || 0} icon={<ClockCircleOutlined />} color="#ff7875" />
                    </Col>
                    <Col xs={24} sm={8} lg={6}>
                      <StatCard title="Завершено" value={stats?.myAssignmentsFinished || 0} icon={<CheckCircleOutlined />} color="#95de64" />
                    </Col>
                  </>
                )}
              </Row>
            </div>
          )}

          {canViewSystem && (
            <div>
              <Title level={5}>Система</Title>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} lg={6}>
                  <StatCard title="Пользователи" value={stats?.userCount || 0} icon={<UserOutlined />} color="#1677ff" />
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <StatCard title="Всего документов" value={stats?.totalDocuments || 0} icon={<DatabaseOutlined />} color="#52c41a" />
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <StatCard title="База данных" value={stats?.dbSize || 'N/A'} icon={<DatabaseOutlined />} color="#13c2c2" />
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <StatCard title="Файлы в хранилище" value={stats?.storageObjects || 0} icon={<CloudOutlined />} color="#722ed1" />
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <StatCard title="Размер хранилища" value={stats?.storageSize || 'N/A'} icon={<HddOutlined />} color="#fa8c16" />
                </Col>
              </Row>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default StatisticsPage;
