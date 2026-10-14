import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Form, Input, InputNumber } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';

const SystemSettingsTab: React.FC = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetAll } = await import('../../../wailsjs/go/services/SettingsService');
      const settings = await GetAll();
      if (settings) {
        const values: any = {};
        settings.forEach((setting: any) => {
          values[setting.key] = setting.value;
        });
        form.setFieldsValue(values);
      }
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  }, [form, message]);

  useEffect(() => { void load(); }, [load]);

  const onSave = async (values: any) => {
    setLoading(true);
    try {
      const { Update } = await import('../../../wailsjs/go/services/SettingsService');
      for (const key in values) {
        const value = typeof values[key] === 'boolean' ? String(values[key]) : String(values[key]);
        await Update(key, value);

        if (key === 'organization_name' && values[key] && String(values[key]).trim() !== '') {
          try {
            const { FindOrCreateOrganization } = await import('../../../wailsjs/go/services/ReferenceService');
            await FindOrCreateOrganization(String(values[key]).trim());
          } catch (error) {
            console.error('Failed to add organization to directory:', error);
          }
        }
      }
      message.success('Настройки сохранены');
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: 600 }}>
      <Form form={form} layout="vertical" onFinish={onSave}>
        <Form.Item name="organization_name" label="Название организации" rules={[{ required: true }]}>
          <Input placeholder="Название вашей организации" />
        </Form.Item>
        <Form.Item name="organization_short_name" label="Краткое название организации" rules={[{ required: true }]}>
          <Input placeholder="Краткое название организации" />
        </Form.Item>
        <Form.Item
          name="password_lifetime_days"
          label="Срок жизни пароля, дней"
          rules={[{ required: true, message: 'Укажите срок жизни пароля' }]}
        >
          <InputNumber min={0} max={3650} placeholder="0 - без ограничения" style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} icon={<EditOutlined />}>
            Сохранить
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
};

export default SystemSettingsTab;
