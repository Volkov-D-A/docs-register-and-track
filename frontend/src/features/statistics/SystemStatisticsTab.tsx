import React, { useCallback, useEffect, useState } from 'react';
import { App, Card, Col, Row, Spin, Statistic } from 'antd';
import { CloudOutlined, DatabaseOutlined, HddOutlined, UserOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';

const StatCard = ({ title, value, icon, color }: any) => <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}><Statistic title={title} value={value} prefix={<span style={{ color, marginRight: 8 }}>{icon}</span>} /></Card>;

const SystemStatisticsTab: React.FC = () => {
  const { message } = App.useApp(); const [stats, setStats] = useState<any>(null); const [loading, setLoading] = useState(false);
  const load = useCallback(async () => { setLoading(true); try { const { GetSystemStatistics } = await import('../../../wailsjs/go/services/StatisticsService'); const result = await GetSystemStatistics(); setStats(result); if (result.storageRefreshInProgress) message.info('Выполняется фоновая сверка размера файлов с MinIO. Показаны данные последней завершённой сверки.'); } catch (err: unknown) { message.error(formatAppError(err)); } finally { setLoading(false); } }, [message]);
  useEffect(() => { void load(); }, [load]);
  return <Spin spinning={loading && !stats}><Row gutter={[16, 16]}>
    <Col xs={24} sm={12} lg={6}><StatCard title="Пользователи" value={stats?.userCount || 0} icon={<UserOutlined />} color="#1677ff" /></Col><Col xs={24} sm={12} lg={6}><StatCard title="Всего документов" value={stats?.totalDocuments || 0} icon={<DatabaseOutlined />} color="#52c41a" /></Col><Col xs={24} sm={12} lg={6}><StatCard title="База данных" value={stats?.dbSize || 'Нет данных'} icon={<DatabaseOutlined />} color="#13c2c2" /></Col><Col xs={24} sm={12} lg={6}><StatCard title="Файлы в хранилище" value={stats?.storageObjects || 0} icon={<CloudOutlined />} color="#722ed1" /></Col><Col xs={24} sm={12} lg={6}><StatCard title="Размер хранилища" value={stats?.storageSize || 'Нет данных'} icon={<HddOutlined />} color="#fa8c16" /></Col>
  </Row></Spin>;
};
export default SystemStatisticsTab;
