import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, App, Button, Card, Col, Descriptions, Row, Space, Spin, Statistic, Tag, Typography } from 'antd';
import { CloudOutlined, DatabaseOutlined, HddOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons';
import { models } from '../../../wailsjs/go/models';
import {
  GetStorageStatisticsStatus,
  GetSystemStatistics,
  RetryStorageStatisticsRefresh,
} from '../../../wailsjs/go/services/StatisticsService';
import { formatAppError } from '../../utils/appError';
import { pollStorageStatus } from '../../utils/storageStatusPolling';
import { useLatestRequest } from '../../hooks/useLatestRequest';
import type { LatestRequestHandlers } from '../../utils/latestRequest';

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
  const [storageStatus, setStorageStatus] = useState<models.StorageStatisticsStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [pollTimedOut, setPollTimedOut] = useState(false);
  const { run: runLatestSystem } = useLatestRequest();
  const { run: runLatestStorage } = useLatestRequest();
  const storageAbortRef = useRef<AbortController | null>(null);

  const startStorageRequest = useCallback(<T,>(
    request: (signal: AbortSignal) => Promise<T>,
    handlers: LatestRequestHandlers<T>,
  ) => {
    storageAbortRef.current?.abort();
    const controller = new AbortController();
    storageAbortRef.current = controller;
    const promise = runLatestStorage(() => request(controller.signal), {
      ...handlers,
      isRelevant: () => !controller.signal.aborted && (handlers.isRelevant?.() ?? true),
    });
    return { controller, promise };
  }, [runLatestStorage]);

  useEffect(() => () => storageAbortRef.current?.abort(), []);

  const loadSystem = useCallback(async () => {
    setLoading(true);
    await runLatestSystem(GetSystemStatistics, {
      onSuccess: setStats,
      onError: (err) => message.error(formatAppError(err)),
      onSettled: () => setLoading(false),
    });
  }, [message, runLatestSystem]);

  const loadStorageStatus = useCallback(async () => {
    setRetrying(true);
    setPollTimedOut(false);
    await startStorageRequest(() => GetStorageStatisticsStatus(), {
      onSuccess: setStorageStatus,
      onError: (err) => message.error(formatAppError(err)),
      onSettled: () => setRetrying(false),
    }).promise;
  }, [message, startStorageRequest]);

  const load = useCallback(() => {
    void loadSystem();
    void loadStorageStatus();
  }, [loadStorageStatus, loadSystem]);

  useEffect(() => { load(); }, [load]);

  const shouldPoll = isRefreshActive(storageStatus?.state) && !pollTimedOut;
  useEffect(() => {
    if (!shouldPoll) return undefined;
    const { controller, promise } = startStorageRequest((signal) => (
      pollStorageStatus(GetStorageStatisticsStatus, () => {}, {
        intervalMs: STORAGE_POLL_INTERVAL_MS,
        maxAttempts: STORAGE_POLL_MAX_ATTEMPTS,
        signal,
      })
    ), {
      onSuccess: (result) => {
        if (result.status) setStorageStatus(result.status);
        if (result.timedOut) {
          setPollTimedOut(true);
          message.warning('Сверка хранилища занимает больше ожидаемого. Можно повторить проверку состояния вручную.');
        }
      },
      onError: (err) => {
        setPollTimedOut(true);
        message.error(formatAppError(err));
      },
    });
    void promise;
    return () => controller.abort();
  }, [message, shouldPoll, startStorageRequest]);

  const retryRefresh = useCallback(async () => {
    setRetrying(true);
    setPollTimedOut(false);
    await startStorageRequest(() => RetryStorageStatisticsRefresh(), {
      onSuccess: setStorageStatus,
      onError: (err) => message.error(formatAppError(err)),
      onSettled: () => setRetrying(false),
    }).promise;
  }, [message, startStorageRequest]);

  const checkStorageStatus = useCallback(async () => {
    setRetrying(true);
    await startStorageRequest(() => GetStorageStatisticsStatus(), {
      onSuccess: (result) => {
        setStorageStatus(result);
        setPollTimedOut(false);
      },
      onError: (err) => message.error(formatAppError(err)),
      onSettled: () => setRetrying(false),
    }).promise;
  }, [message, startStorageRequest]);

  const storageObjects = storageStatus?.storageObjects ?? stats?.storageObjects ?? 0;
  const storageSize = storageStatus?.storageSize ?? stats?.storageSize ?? 'Нет данных';
  const refreshedAt = storageStatus?.refreshedAt ?? stats?.storageRefreshedAt;
  const refreshActive = isRefreshActive(storageStatus?.state);
  const state = serviceState(stats?.service?.state);
  const attachmentProblems = (stats?.attachments?.missingObjects ?? 0) + (stats?.attachments?.orphanObjects ?? 0);

  return <Spin spinning={loading && !stats}>
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}><StatCard title="Пользователи" value={stats?.userCount ?? 0} icon={<UserOutlined />} color="#1677ff" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Всего документов" value={stats?.totalDocuments ?? 0} icon={<DatabaseOutlined />} color="#52c41a" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="База данных" value={stats?.dbSize ?? 'Нет данных'} icon={<DatabaseOutlined />} color="#13c2c2" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Файлы в хранилище" value={storageObjects} icon={<CloudOutlined />} color="#722ed1" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Размер хранилища" value={storageSize} icon={<HddOutlined />} color="#fa8c16" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Активные пользователи, 15 мин." value={stats?.usage?.activeUsers15m ?? 0} icon={<UserOutlined />} color="#2f54eb" /></Col>
        <Col xs={24} sm={12} lg={6}><StatCard title="Активные сессии" value={stats?.usage?.activeSessions ?? 0} icon={<UserOutlined />} color="#597ef7" /></Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <Card title="Сервис" style={{ height: '100%' }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="Состояние"><Tag color={state.color}>{state.label}</Tag></Descriptions.Item>
              <Descriptions.Item label="Версия">{stats?.service?.version || 'Нет данных'}</Descriptions.Item>
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

      {attachmentProblems > 0 && <Alert
        type="warning"
        showIcon
        message="Последняя сверка вложений обнаружила расхождения"
        description={`Отсутствуют объектов: ${stats?.attachments?.missingObjects ?? 0}; лишних объектов: ${stats?.attachments?.orphanObjects ?? 0}.`}
      />}

      {refreshActive && !pollTimedOut && <Typography.Text type="secondary">Выполняется фоновая сверка MinIO. Показаны данные последней завершённой сверки.</Typography.Text>}
      {pollTimedOut && <Alert type="warning" showIcon message="Сверка ещё не завершена" description="Автоматическое ожидание остановлено. Проверьте состояние ещё раз вручную." />}
      {storageStatus?.state === 'failed' && <Alert type="error" showIcon message="Сверка MinIO завершилась с ошибкой" description={storageStatus.lastError || 'Повторите попытку.'} />}

      <Space>
        {refreshedAt && <Typography.Text type="secondary">Последняя полная сверка MinIO: {new Date(refreshedAt).toLocaleString('ru-RU')}</Typography.Text>}
        {stats?.generatedAt && <Typography.Text type="secondary">Сводка сформирована: {new Date(stats.generatedAt).toLocaleString('ru-RU')}</Typography.Text>}
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
