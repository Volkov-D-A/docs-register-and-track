import React, { useState } from 'react';
import { Alert, App, Button, DatePicker, Divider, Form, List, Typography } from 'antd';
import { DeleteOutlined, SearchOutlined, WarningOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';

const StorageTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [reconciling, setReconciling] = useState(false);
  const [selectedDate, setSelectedDate] = useState<any>(null);
  const [reconciliation, setReconciliation] = useState<{ missingObjects: string[]; orphanObjects: string[] } | null>(null);

  const onBulkDelete = () => {
    if (loading) {
      return;
    }
    if (!selectedDate) {
      message.warning('Пожалуйста, выберите дату');
      return;
    }

    modal.confirm({
      title: 'Массовое удаление файлов',
      content: `Это действие безвозвратно удалит все файлы, загруженные до ${selectedDate.format('DD.MM.YYYY')}. Убедитесь, что нужные вложения сохранены в резервной копии.`,
      okText: 'Удалить файлы',
      cancelText: 'Отмена',
      okType: 'danger',
      onOk: async () => {
        if (loading) {
          return;
        }
        setLoading(true);
        try {
          const { BulkDeleteOlderThan } = await import('../../../wailsjs/go/services/AttachmentService');
          const count = await BulkDeleteOlderThan(selectedDate.toISOString());
          message.success(`Удалено файлов: ${count}`);
        } catch (error: unknown) {
          message.error(formatAppError(error));
        } finally {
          setLoading(false);
        }
      },
    });
  };

  const onReconcile = async () => {
    if (reconciling) return;
    setReconciling(true);
    try {
      const { ReconcileStorage } = await import('../../../wailsjs/go/services/AttachmentService');
      const result = await ReconcileStorage();
      setReconciliation(result);
      if (!result.missingObjects.length && !result.orphanObjects.length) message.success('Расхождений между базой данных и MinIO не найдено');
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setReconciling(false);
    }
  };

  return (
    <div style={{ maxWidth: 600 }}>
      <div style={{
        padding: 24,
        background: 'var(--app-warning-bg)',
        borderRadius: 8,
        border: '1px solid var(--app-warning-border)',
        marginBottom: 16,
      }}>
        <Typography.Paragraph type="warning" strong style={{ marginBottom: 0 }}>
          <WarningOutlined style={{ marginRight: 8 }} />
          Осторожно: удалённые файлы невозможно восстановить средствами системы.
        </Typography.Paragraph>
      </div>
      <Form layout="vertical">
        <Form.Item label="Удалить файлы, загруженные до даты:">
          <DatePicker
            style={{ width: '100%' }}
            onChange={(date) => setSelectedDate(date)}
            disabledDate={(current) => current && current.valueOf() > Date.now()}
          />
        </Form.Item>
        <Form.Item>
          <Button
            type="primary"
            danger
            icon={<DeleteOutlined />}
            onClick={onBulkDelete}
            loading={loading}
            disabled={!selectedDate}
          >
            Удалить файлы
          </Button>
        </Form.Item>
      </Form>
      <Divider />
      <Typography.Title level={4}>Сверка вложений с MinIO</Typography.Title>
      <Typography.Paragraph type="secondary">
        Проверка сопоставляет ссылки на вложения в базе данных с объектами в MinIO и ничего не изменяет автоматически.
      </Typography.Paragraph>
      <Button icon={<SearchOutlined />} onClick={() => void onReconcile()} loading={reconciling}>
        Сверить с MinIO
      </Button>
      {reconciliation && <div style={{ marginTop: 16 }}>
        {!reconciliation.missingObjects.length && !reconciliation.orphanObjects.length && <Alert type="success" showIcon message="Расхождения не обнаружены" />}
        {!!reconciliation.missingObjects.length && <Alert
          type="error"
          showIcon
          message={`Ссылки без файлов: ${reconciliation.missingObjects.length}`}
          description={<><Typography.Paragraph style={{ marginTop: 8 }}>Рекомендуется восстановить эти объекты из согласованной резервной копии. Если восстановление невозможно, удалите соответствующие вложения из документов штатными средствами.</Typography.Paragraph><List size="small" bordered dataSource={reconciliation.missingObjects} renderItem={(path) => <List.Item><Typography.Text code>{path}</Typography.Text></List.Item>} /></>}
          style={{ marginTop: 12 }}
        />}
        {!!reconciliation.orphanObjects.length && <Alert
          type="warning"
          showIcon
          message={`Файлы без ссылок: ${reconciliation.orphanObjects.length}`}
          description={<><Typography.Paragraph style={{ marginTop: 8 }}>Проверьте список и резервную копию; после подтверждения ненужные объекты можно удалить из MinIO. Автоматическое удаление не выполняется.</Typography.Paragraph><List size="small" bordered dataSource={reconciliation.orphanObjects} renderItem={(path) => <List.Item><Typography.Text code>{path}</Typography.Text></List.Item>} /></>}
          style={{ marginTop: 12 }}
        />}
      </div>}
    </div>
  );
};

export default StorageTab;
