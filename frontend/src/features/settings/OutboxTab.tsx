import React, { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Card, Col, Popconfirm, Row, Space, Statistic, Table, Tag, Tooltip, Typography } from 'antd';
import { ReloadOutlined, RedoOutlined } from '@ant-design/icons';
import type { models } from '../../../wailsjs/go/models';
import { formatAppError } from '../../utils/appError';

type FailedOutboxEvent = models.FailedOutboxEvent;
type OutboxStats = models.OutboxStats;

const eventTypeLabels: Record<string, string> = {
  user_event: 'Уведомление пользователя',
  journal_entry: 'Запись журнала',
  admin_audit: 'Административный аудит',
  attachment_delete: 'Удаление вложения',
};

const formatUUID = (value: unknown): string => {
  if (typeof value === 'string') return value;
  if (!Array.isArray(value) || value.length !== 16 || !value.every((part) => Number.isInteger(part) && part >= 0 && part <= 255)) {
    return '';
  }
  const hex = value.map((part) => part.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
};

const formatDate = (value: unknown): string => {
  if (!value) return '-';
  const date = new Date(value as string);
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString('ru-RU');
};

const OutboxTab: React.FC = () => {
  const { message } = App.useApp();
  const [stats, setStats] = useState<OutboxStats | null>(null);
  const [failedEvents, setFailedEvents] = useState<FailedOutboxEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [requeueingID, setRequeueingID] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetFailed, GetStats } = await import('../../../wailsjs/go/services/OutboxAdminService');
      const [nextStats, nextFailedEvents] = await Promise.all([GetStats(), GetFailed(100)]);
      setStats(nextStats);
      setFailedEvents(nextFailedEvents || []);
    } catch (error: unknown) {
      message.error(formatAppError(error, 'Не удалось загрузить состояние очереди'));
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => { void load(); }, [load]);

  const requeue = async (event: FailedOutboxEvent) => {
    const eventID = formatUUID(event.id);
    if (!eventID) {
      message.error('Не удалось определить идентификатор задачи');
      return;
    }
    setRequeueingID(eventID);
    try {
      const { Requeue } = await import('../../../wailsjs/go/services/OutboxAdminService');
      await Requeue(eventID);
      message.success('Задача возвращена в очередь и будет повторно обработана');
      await load();
    } catch (error: unknown) {
      message.error(formatAppError(error, 'Не удалось вернуть задачу в очередь'));
    } finally {
      setRequeueingID(null);
    }
  };

  const columns = [
    {
      title: 'Тип события',
      dataIndex: 'eventType',
      key: 'eventType',
      width: 190,
      render: (value: string) => <Tag color="red">{eventTypeLabels[value] || value}</Tag>,
    },
    {
      title: 'Ошибка',
      dataIndex: 'lastError',
      key: 'lastError',
      render: (value: string) => (
        <Tooltip title={value}>
          <Typography.Text ellipsis style={{ maxWidth: 360, display: 'inline-block' }}>{value || '-'}</Typography.Text>
        </Tooltip>
      ),
    },
    { title: 'Попыток', dataIndex: 'attempts', key: 'attempts', width: 90 },
    {
      title: 'Не обработана с',
      dataIndex: 'failedAt',
      key: 'failedAt',
      width: 175,
      render: formatDate,
    },
    {
      title: 'Действие',
      key: 'action',
      width: 130,
      render: (_: unknown, event: FailedOutboxEvent) => {
        const eventID = formatUUID(event.id);
        return (
          <Popconfirm
            title="Повторить обработку?"
            description="Ошибка и счётчик попыток будут сброшены. Задача вернётся в очередь немедленно."
            okText="Повторить"
            cancelText="Отмена"
            onConfirm={() => void requeue(event)}
            disabled={!eventID}
          >
            <Button icon={<RedoOutlined />} loading={requeueingID === eventID} disabled={!eventID}>Повторить</Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>Обновить</Button>
      </Space>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} lg={6}><Card size="small"><Statistic title="Ожидают" value={stats?.Pending ?? 0} /></Card></Col>
        <Col xs={24} sm={12} lg={6}><Card size="small"><Statistic title="В обработке" value={stats?.Processing ?? 0} /></Card></Col>
        <Col xs={24} sm={12} lg={6}><Card size="small"><Statistic title="Не обработаны" value={stats?.Failed ?? 0} styles={{ content: { color: stats?.Failed ? '#cf1322' : undefined } }} /></Card></Col>
        <Col xs={24} sm={12} lg={6}><Card size="small"><Statistic title="Обработано" value={stats?.Processed ?? 0} /></Card></Col>
      </Row>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        title="Повторная обработка"
        description="В списке показаны только задачи, исчерпавшие автоматические попытки. Перед повтором устраните первопричину ошибки."
      />
      <Table
        columns={columns}
        dataSource={failedEvents}
        rowKey={(event) => formatUUID(event.id)}
        loading={loading}
        size="small"
        pagination={{ pageSize: 20, showSizeChanger: false, showTotal: (count) => `Всего: ${count}` }}
        locale={{ emptyText: 'Терминально не обработанных задач нет' }}
      />
    </div>
  );
};

export default OutboxTab;
