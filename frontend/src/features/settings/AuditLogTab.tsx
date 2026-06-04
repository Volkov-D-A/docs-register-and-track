import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Space, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';

const actionLabels: Record<string, string> = {
  SETTINGS_UPDATE: 'Изменение настроек',
  USER_CREATE: 'Создание пользователя',
  USER_UPDATE: 'Обновление пользователя',
  USER_PASSWORD_RESET: 'Сброс пароля',
  NOMENCLATURE_CREATE: 'Создание номенклатуры',
  NOMENCLATURE_UPDATE: 'Обновление номенклатуры',
  NOMENCLATURE_DELETE: 'Удаление номенклатуры',
  DOCTYPE_CREATE: 'Создание типа документа',
  DOCTYPE_UPDATE: 'Обновление типа документа',
  DOCTYPE_DELETE: 'Удаление типа документа',
  ORG_UPDATE: 'Обновление организации',
  ORG_DELETE: 'Удаление организации',
  DEPT_CREATE: 'Создание подразделения',
  DEPT_UPDATE: 'Обновление подразделения',
  DEPT_DELETE: 'Удаление подразделения',
  MIGRATION_RUN: 'Применение миграций',
  MIGRATION_ROLLBACK: 'Откат миграции',
  FILES_BULK_DELETE: 'Массовое удаление файлов',
};

const AuditLogTab: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const load = useCallback(async (nextPage: number) => {
    setLoading(true);
    try {
      const { GetAll } = await import('../../../wailsjs/go/services/AdminAuditLogService');
      const result = await GetAll(nextPage, pageSize);
      setData(result?.items || []);
      setTotal(result?.total || 0);
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  }, [message, pageSize]);

  useEffect(() => { void load(page); }, [load, page]);

  const columns = [
    {
      title: 'Дата и время',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (value: string) => value ? new Date(value).toLocaleString('ru-RU') : '-',
    },
    { title: 'Пользователь', dataIndex: 'userName', key: 'userName', width: 200 },
    {
      title: 'Действие',
      dataIndex: 'action',
      key: 'action',
      width: 220,
      render: (value: string) => {
        const label = actionLabels[value] || value;
        let color = 'default';
        if (value.includes('CREATE')) color = 'green';
        else if (value.includes('DELETE') || value.includes('ROLLBACK')) color = 'red';
        else if (value.includes('UPDATE') || value.includes('RESET')) color = 'blue';
        else if (value.includes('MIGRATION_RUN')) color = 'orange';
        return <Tag color={color}>{label}</Tag>;
      },
    },
    { title: 'Подробности', dataIndex: 'details', key: 'details' },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button title="Обновить журнал администрирования" icon={<ReloadOutlined />} onClick={() => void load(page)}>Обновить</Button>
      </Space>
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        size="small"
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: false,
          onChange: (nextPage) => setPage(nextPage),
          showTotal: (count) => `Всего: ${count}`,
        }}
      />
    </div>
  );
};

export default AuditLogTab;
