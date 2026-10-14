import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Card, Col, Descriptions, Row, Space, Spin, Statistic, Tag, Typography } from 'antd';
import { DatabaseOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons';
import { models } from '../../../wailsjs/go/models';
import {
  GetSystemStatistics,
} from '../../../wailsjs/go/services/StatisticsService';
import { formatAppError } from '../../utils/appError';
import { useLatestRequest } from '../../hooks/useLatestRequest';


type StatCardProps = {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  color: string;
};

const StatCard = ({ title, value, icon, color }: StatCardProps) => (
  <Card variant="borderless" style={{ height: '100%', borderRadius: 8, boxShadow: '0 2px 8px var(--app-panel-shadow)' }}>
    <Statistic title={title} value={value} prefix={<span style={{ color, marginRight: 8 }}>{icon}</span>} />
  </Card>
);


const formatUptime = (seconds?: number) => {
  if (!seconds || seconds < 60) return 'меньше минуты';
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return [days && `${days} дн.`, hours && `${hours} ч.`, minutes && `${minutes} мин.`].filter(Boolean).join(' ');
};

const serviceState = (state?: string) => {
  if (state === 'ready') return { color: 'success', label: 'Готов' };
  if (state === 'maintenance') return { color: 'warning', label: 'Обслуживание' };
  return { color: 'error', label: 'Не готов' };
};

const SystemStatisticsTab: React.FC = () => {
  const { message } = App.useApp();
  const [stats, setStats] = useState<models.SystemStatistics | null>(null);
  const [loading,setLoading]=useState(false);
  const {run:runLatestSystem}=useLatestRequest();
  const load=useCallback(async()=>{
   setLoading(true);await runLatestSystem(()=>GetSystemStatistics(),{onSuccess:setStats,onError:(error)=>message.error(formatAppError(error)),onSettled:()=>setLoading(false)});
  },[message,runLatestSystem]);
  useEffect(()=>{void load();},[load]);
  const state=serviceState(stats?.service?.state);

  return <Spin spinning={loading && !stats}>
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}><StatCard title="Пользователи" value={stats?.userCount ?? 0} icon={<UserOutlined />} color="#1677ff" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Всего документов" value={stats?.totalDocuments ?? 0} icon={<DatabaseOutlined />} color="#52c41a" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="База данных" value={stats?.dbSize ?? 'Нет данных'} icon={<DatabaseOutlined />} color="#13c2c2" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Активные пользователи, 15 мин." value={stats?.usage?.activeUsers15m ?? 0} icon={<UserOutlined />} color="#2f54eb" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Активные сессии" value={stats?.usage?.activeSessions ?? 0} icon={<UserOutlined />} color="#597ef7" /></Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <Card title="Сервис" style={{ height: '100%' }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Состояние"><Tag color={state.color}>{state.label}</Tag></Descriptions.Item>
              <Descriptions.Item label="Версия">{stats?.service?.version || 'Нет данных'}</Descriptions.Item>
              <Descriptions.Item label="Сборка клиента">{stats?.clientBuildVersion || 'Не определена'}{stats?.clientDirty ? ' (локальная)' : ''} / {stats?.clientRevision}</Descriptions.Item>
              <Descriptions.Item label="Сборка сервера">{stats?.service?.buildVersion || 'Не определена'}{stats?.service?.sourceDirty ? ' (локальная)' : ''} / {stats?.service?.sourceRevision}</Descriptions.Item>
              <Descriptions.Item label="Запущен">
                {stats?.service?.startedAt ? new Date(stats.service.startedAt).toLocaleString('ru-RU') : 'Нет данных'}
              </Descriptions.Item>
              <Descriptions.Item label="Время работы">{formatUptime(stats?.service?.uptimeSeconds)}</Descriptions.Item>
              <Descriptions.Item label="Схема БД">
                {stats?.service ? `${stats.service.schemaCurrentVersion} / ${stats.service.schemaRequiredVersion}` : 'Нет данных'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card title="API" style={{ height: '100%' }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Запросы с момента запуска">{stats?.api?.requestsSinceStart ?? 0}</Descriptions.Item>
              <Descriptions.Item label="Ошибки 4xx / 5xx">{stats?.api?.clientErrorsSinceStart ?? 0} / {stats?.api?.serverErrorsSinceStart ?? 0}</Descriptions.Item>
              <Descriptions.Item label="Превышения времени ожидания">{stats?.api?.deadlineExceededSinceStart ?? 0}</Descriptions.Item>
              <Descriptions.Item label={`p95, окно до ${stats?.api?.sampleWindow ?? 0} запросов`}>{stats?.api?.p95Milliseconds ?? 0} мс</Descriptions.Item>
              <Descriptions.Item label="Выполняются сейчас">{stats?.api?.inFlight ?? 0}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card title="PostgreSQL" style={{ height: '100%' }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Пул: занято / свободно / открыто / максимум">
                {stats?.database?.poolInUse ?? 0} / {stats?.database?.poolIdle ?? 0} / {stats?.database?.poolOpen ?? 0} / {stats?.database?.poolMax ?? 0}
              </Descriptions.Item>
              <Descriptions.Item label="Ожидания соединения">
                {stats?.database?.waitCountSinceStart ?? 0} ({stats?.database?.waitMillisecondsSinceStart ?? 0} мс)
              </Descriptions.Item>
              <Descriptions.Item label="Операции / ошибки">{stats?.database?.operationsSinceStart ?? 0} / {stats?.database?.operationErrorsSinceStart ?? 0}</Descriptions.Item>
              <Descriptions.Item label="p95 операций">{stats?.database?.operationP95Milliseconds ?? 0} мс</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card title="Фоновая очередь" style={{ height: '100%' }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Ожидают / выполняются">{stats?.outbox?.pending ?? 0} / {stats?.outbox?.processing ?? 0}</Descriptions.Item>
              <Descriptions.Item label="Необработанные ошибки">{stats?.outbox?.failed ?? 0}</Descriptions.Item>
              <Descriptions.Item label="Обработано с момента запуска">{stats?.outbox?.processedSinceStart ?? 0}</Descriptions.Item>
              <Descriptions.Item label="Повторные попытки">{stats?.outbox?.retriesSinceStart ?? 0}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
      </Row>

      <Space>
        {stats?.generatedAt && <Typography.Text type="secondary">Сводка сформирована: {new Date(stats.generatedAt).toLocaleString('ru-RU')}</Typography.Text>}
        <Button icon={<ReloadOutlined />} loading={loading} onClick={()=>void load()}>Обновить данные</Button>
      </Space>
    </Space>
  </Spin>;
};

export default SystemStatisticsTab;
