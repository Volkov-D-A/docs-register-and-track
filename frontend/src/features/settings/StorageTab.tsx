import React, { useState } from 'react';
import { App, Button, DatePicker, Form, Typography } from 'antd';
import { DeleteOutlined, WarningOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';

const StorageTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [selectedDate, setSelectedDate] = useState<any>(null);

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
    </div>
  );
};

export default StorageTab;
