import React, { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Card, Col, Row, Space, Spin, Statistic, Typography } from 'antd';
import { CloudOutlined, DatabaseOutlined, HddOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons';
import { models } from '../../../wailsjs/go/models';
import {
  GetStorageStatisticsStatus,
  GetSystemStatistics,
  RetryStorageStatisticsRefresh,
} from '../../../wailsjs/go/services/StatisticsService';
import { formatAppError } from '../../utils/appError';
import { pollStorageStatus } from '../../utils/storageStatusPolling';

const STORAGE_POLL_INTERVAL_MS = 1500;
const STORAGE_POLL_MAX_ATTEMPTS = 40;

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

const isRefreshActive = (state?: string) => state === 'pending' || state === 'running';

const SystemStatisticsTab: React.FC = () => {
  const { message } = App.useApp();
  const [stats, setStats] = useState<models.SystemStatistics | null>(null);
  const [storageStatus, setStorageStatus] = useState<models.StorageStatisticsStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [pollTimedOut, setPollTimedOut] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setPollTimedOut(false);
    try {
      const systemResult = await GetSystemStatistics();
      const storageResult = await GetStorageStatisticsStatus();
      setStats(systemResult);
      setStorageStatus(storageResult);
    } catch (err: unknown) {
      message.error(formatAppError(err));
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => { void load(); }, [load]);

  const shouldPoll = isRefreshActive(storageStatus?.state) && !pollTimedOut;
  useEffect(() => {
    if (!shouldPoll) return undefined;
    const controller = new AbortController();
    void pollStorageStatus(GetStorageStatisticsStatus, setStorageStatus, {
      intervalMs: STORAGE_POLL_INTERVAL_MS,
      maxAttempts: STORAGE_POLL_MAX_ATTEMPTS,
      signal: controller.signal,
    }).then((result) => {
      if (result.timedOut && !controller.signal.aborted) {
        setPollTimedOut(true);
        message.warning('Сверка хранилища занимает больше ожидаемого. Можно повторить проверку состояния вручную.');
      }
    }).catch((err: unknown) => {
      if (!controller.signal.aborted) {
        setPollTimedOut(true);
        message.error(formatAppError(err));
      }
    });
    return () => controller.abort();
  }, [message, shouldPoll]);

  const retryRefresh = useCallback(async () => {
    setRetrying(true);
    setPollTimedOut(false);
    try {
      setStorageStatus(await RetryStorageStatisticsRefresh());
    } catch (err: unknown) {
      message.error(formatAppError(err));
    } finally {
      setRetrying(false);
    }
  }, [message]);

  const checkStorageStatus = useCallback(async () => {
    setRetrying(true);
    try {
      const result = await GetStorageStatisticsStatus();
      setStorageStatus(result);
      setPollTimedOut(false);
    } catch (err: unknown) {
      message.error(formatAppError(err));
    } finally {
      setRetrying(false);
    }
  }, [message]);

  const storageObjects = storageStatus?.storageObjects ?? stats?.storageObjects ?? 0;
  const storageSize = storageStatus?.storageSize ?? stats?.storageSize ?? 'Нет данных';
  const refreshedAt = storageStatus?.refreshedAt ?? stats?.storageRefreshedAt;
  const refreshActive = isRefreshActive(storageStatus?.state);

  return <Spin spinning={loading && !stats}>
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}><StatCard title="Пользователи" value={stats?.userCount ?? 0} icon={<UserOutlined />} color="#1677ff" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Всего документов" value={stats?.totalDocuments ?? 0} icon={<DatabaseOutlined />} color="#52c41a" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="База данных" value={stats?.dbSize ?? 'Нет данных'} icon={<DatabaseOutlined />} color="#13c2c2" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Файлы в хранилище" value={storageObjects} icon={<CloudOutlined />} color="#722ed1" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Размер хранилища" value={storageSize} icon={<HddOutlined />} color="#fa8c16" /></Col>
      </Row>

      {refreshActive && !pollTimedOut && <Typography.Text type="secondary">Выполняется фоновая сверка MinIO. Показаны данные последней завершённой сверки.</Typography.Text>}
      {pollTimedOut && <Alert type="warning" showIcon message="Сверка ещё не завершена" description="Автоматическое ожидание остановлено. Проверьте состояние ещё раз вручную." />}
      {storageStatus?.state === 'failed' && <Alert type="error" showIcon message="Сверка MinIO завершилась с ошибкой" description={storageStatus.lastError || 'Повторите попытку.'} />}

      <Space>
        {refreshedAt && <Typography.Text type="secondary">Последняя полная сверка MinIO: {new Date(refreshedAt).toLocaleString('ru-RU')}</Typography.Text>}
        <Button
          icon={<ReloadOutlined />}
          loading={loading || retrying}
          disabled={refreshActive && !pollTimedOut}
          onClick={() => {
            if (storageStatus?.state === 'failed') void retryRefresh();
            else if (pollTimedOut) void checkStorageStatus();
            else void load();
          }}
        >
          {storageStatus?.state === 'failed' ? 'Повторить сверку' : 'Обновить данные'}
        </Button>
      </Space>
    </Space>
  </Spin>;
};

export default SystemStatisticsTab;
