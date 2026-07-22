import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Checkbox, Form, Input, Modal, Space, Tag, Typography } from 'antd';
import { CheckCircleOutlined, DatabaseOutlined, DeleteOutlined, WarningOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';

const ROLLBACK_MIGRATION_CONFIRMATION_PHRASE = 'ОТКАТ МИГРАЦИИ';

const MigrationsTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [rollbackForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);
  const [rollbackModalOpen, setRollbackModalOpen] = useState(false);
  const [status, setStatus] = useState<any>(null);

  const loadStatus = useCallback(async () => {
    setLoading(true);
    try {
      const { GetMigrationStatus } = await import('../../../wailsjs/go/services/SettingsService');
      const result = await GetMigrationStatus();
      setStatus(result);
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => { void loadStatus(); }, [loadStatus]);

  const schemaNeedsUpdate = Boolean(
    status && !status.upToDate && !status.schemaTooNew && !status.dirty,
  );

  const onRunMigrations = () => {
    if (running) {
      return;
    }
    modal.confirm({
      title: 'Запуск миграций',
      content: 'Вы уверены, что хотите применить миграции базы данных? Убедитесь, что все пользователи завершили работу в системе.',
      okText: 'Запустить',
      cancelText: 'Отмена',
      okType: 'primary',
      onOk: async () => {
        if (running) {
          return;
        }
        setRunning(true);
        try {
          const { RunMigrations } = await import('../../../wailsjs/go/services/SettingsService');
          await RunMigrations();
          message.success('Миграции успешно применены');
          await loadStatus();
        } catch (error: unknown) {
          message.error(formatAppError(error));
        } finally {
          setRunning(false);
        }
      },
    });
  };

  const onRollback = () => {
    rollbackForm.resetFields();
    setRollbackModalOpen(true);
  };

  const onConfirmRollback = async () => {
    if (rollingBack) {
      return;
    }
    const values = await rollbackForm.validateFields();
    setRollingBack(true);
    try {
      const { RollbackMigration } = await import('../../../wailsjs/go/services/SettingsService');
      await RollbackMigration({
        backupCompleted: values.backupCompleted,
        backupReference: values.backupReference,
        acknowledgedDataLoss: values.acknowledgedDataLoss,
        confirmation: values.confirmation,
      });
      message.warning('Откат выполнен. Обычная работа заблокирована до повторного применения миграций.');
      setRollbackModalOpen(false);
      await loadStatus();
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setRollingBack(false);
    }
  };

  return (
    <div style={{ maxWidth: 600 }}>
      <div style={{
        padding: 24,
        background: 'var(--app-subtle-surface)',
        borderRadius: 8,
        border: '1px solid var(--app-border)',
        marginBottom: 16,
      }}>
        <Typography.Title level={5} style={{ marginTop: 0 }}>
          <DatabaseOutlined style={{ marginRight: 8 }} />
          Статус миграций базы данных
        </Typography.Title>

        {loading ? (
          <Typography.Text type="secondary">Загрузка...</Typography.Text>
        ) : status ? (
          <Space orientation="vertical" size="small" style={{ width: '100%' }}>
            <div>
              <Typography.Text strong>Текущая версия: </Typography.Text>
              <Typography.Text>{status.currentVersion || 0}</Typography.Text>
            </div>
            <div>
              <Typography.Text strong>Доступно миграций: </Typography.Text>
              <Typography.Text>{status.availableCount}</Typography.Text>
            </div>
            <div>
              <Typography.Text strong>Последняя доступная версия: </Typography.Text>
              <Typography.Text>{status.latestAvailableVersion || 0}</Typography.Text>
            </div>
            <div>
              <Typography.Text strong>Статус: </Typography.Text>
              {status.upToDate ? (
                <Tag icon={<CheckCircleOutlined />} color="success">Актуальна</Tag>
              ) : status.schemaTooNew ? (
                <Tag icon={<WarningOutlined />} color="error">Схема новее приложения</Tag>
              ) : status.dirty ? (
                <Tag icon={<WarningOutlined />} color="error">Миграция завершилась с ошибкой</Tag>
              ) : (
                <Tag icon={<WarningOutlined />} color="warning">Требуется обновление</Tag>
              )}
            </div>
            {schemaNeedsUpdate ? (
              <Typography.Text type="warning">
                Прикладные операции заблокированы до применения доступных миграций.
              </Typography.Text>
            ) : null}
          </Space>
        ) : (
          <Typography.Text type="secondary">Не удалось получить статус</Typography.Text>
        )}
      </div>

      <Space size="small" style={{ marginBottom: 8 }}>
        <Button
          type="primary"
          icon={<DatabaseOutlined />}
          onClick={onRunMigrations}
          loading={running}
          disabled={status?.upToDate || status?.schemaTooNew || status?.dirty}
        >
          Применить миграции
        </Button>
        <Button
          danger
          icon={<DeleteOutlined />}
          onClick={onRollback}
          loading={rollingBack}
          disabled={!status?.currentVersion || !status?.upToDate}
        >
          Откатить последнюю миграцию
        </Button>
      </Space>
      <div>
        <Typography.Text type="secondary">
          Перед запуском миграций убедитесь, что все пользователи завершили работу. Перед откатом требуется свежая резервная копия PostgreSQL и MinIO.
        </Typography.Text>
      </div>

      <Modal
        title="Откатить последнюю миграцию?"
        open={rollbackModalOpen}
        onCancel={() => setRollbackModalOpen(false)}
        onOk={onConfirmRollback}
        okText="Откатить миграцию"
        cancelText="Отмена"
        okButtonProps={{ danger: true, loading: rollingBack }}
        confirmLoading={rollingBack}
        destroyOnClose
      >
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Text type="danger">
            Откат последней миграции может удалить таблицы, столбцы и данные, созданные этой миграцией.
          </Typography.Text>
          <Form form={rollbackForm} layout="vertical" preserve={false}>
            <Form.Item
              name="backupCompleted"
              valuePropName="checked"
              rules={[{
                validator: (_, checked) => checked ? Promise.resolve() : Promise.reject(new Error('Подтвердите наличие свежей резервной копии')),
              }]}
            >
              <Checkbox>Свежая резервная копия PostgreSQL и MinIO создана и проверена</Checkbox>
            </Form.Item>
            <Form.Item
              name="backupReference"
              label="Идентификатор или путь к резервной копии"
              rules={[{ required: true, whitespace: true, message: 'Укажите резервную копию' }]}
            >
              <Input placeholder="Например: smb://backup/docflow/2026-05-28_120000.tar" />
            </Form.Item>
            <Form.Item
              name="acknowledgedDataLoss"
              valuePropName="checked"
              rules={[{
                validator: (_, checked) => checked ? Promise.resolve() : Promise.reject(new Error('Подтвердите риск потери данных')),
              }]}
            >
              <Checkbox>Я понимаю, что откат может удалить production-данные</Checkbox>
            </Form.Item>
            <Form.Item
              name="confirmation"
              label={`Введите: ${ROLLBACK_MIGRATION_CONFIRMATION_PHRASE}`}
              rules={[{
                validator: (_, value) => String(value || '').trim() === ROLLBACK_MIGRATION_CONFIRMATION_PHRASE
                  ? Promise.resolve()
                  : Promise.reject(new Error('Контрольная фраза не совпадает')),
              }]}
            >
              <Input />
            </Form.Item>
          </Form>
        </Space>
      </Modal>
    </div>
  );
};

export default MigrationsTab;
