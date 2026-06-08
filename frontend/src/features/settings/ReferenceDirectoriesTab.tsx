import React, { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Form, Input, Modal, Popconfirm, Select, Space, Table, Tabs, Typography } from 'antd';
import { BankOutlined, DeleteOutlined, EditOutlined, SolutionOutlined, SwapOutlined } from '@ant-design/icons';

import { formatAppError } from '../../utils/appError';
import { confirmDiscardFormChanges } from '../../utils/dirtyForm';

const OrganizationsTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [mergeModalOpen, setMergeModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [mergeItem, setMergeItem] = useState<any>(null);
  const [form] = Form.useForm();
  const [mergeForm] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetOrganizations } = await import('../../../wailsjs/go/services/ReferenceService');
      const items = await GetOrganizations();
      setData(items || []);
    } catch (err: any) {
      message.error(formatAppError(err));
    }
    setLoading(false);
  }, [message]);

  useEffect(() => { load(); }, [load]);

  const onSave = async (values: any) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      if (editItem) {
        const { UpdateOrganization } = await import('../../../wailsjs/go/services/ReferenceService');
        await UpdateOrganization(editItem.id, values.name);
      }
      message.success('Организация обновлена');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      await load();
    } catch (err: any) {
      message.error(formatAppError(err));
    } finally {
      setLoading(false);
    }
  };

  const onDelete = async (id: string) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      const { DeleteOrganization } = await import('../../../wailsjs/go/services/ReferenceService');
      await DeleteOrganization(id);
      message.success('Организация удалена');
      await load();
    } catch (err: any) {
      message.error(formatAppError(err));
    } finally {
      setLoading(false);
    }
  };

  const onMerge = async (values: any) => {
    if (loading || !mergeItem) {
      return;
    }
    setLoading(true);
    try {
      const { MergeOrganizations } = await import('../../../wailsjs/go/services/ReferenceService');
      await MergeOrganizations(mergeItem.id, values.targetId);
      message.success('Организации объединены');
      setMergeModalOpen(false);
      mergeForm.resetFields();
      setMergeItem(null);
      await load();
    } catch (err: any) {
      message.error(formatAppError(err));
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    {
      title: 'Действия', key: 'actions', width: 132,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" title="Редактировать организацию" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Button size="small" title="Объединить с другой организацией" icon={<SwapOutlined />} onClick={() => {
            setMergeItem(record);
            mergeForm.resetFields();
            setMergeModalOpen(true);
          }} />
          <Popconfirm
            title={`Удалить организацию "${record.name}"?`}
            description="Это действие нельзя отменить. Организация исчезнет из справочника."
            okText="Удалить"
            cancelText="Отмена"
            okButtonProps={{ danger: true, loading }}
            onConfirm={() => onDelete(record.id)}
          >
            <Button size="small" title="Удалить организацию" icon={<DeleteOutlined />} danger loading={loading} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />
      <Typography.Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
        Организации добавляются автоматически при регистрации документов
      </Typography.Text>

      <Modal
        title="Редактировать организацию"
        open={modalOpen}
        onCancel={() => confirmDiscardFormChanges(modal, form, () => {
          setModalOpen(false);
          setEditItem(null);
          form.resetFields();
        })}
        onOk={() => form.submit()}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="name" label="Наименование" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Объединить организацию"
        open={mergeModalOpen}
        onCancel={() => confirmDiscardFormChanges(modal, mergeForm, () => {
          setMergeModalOpen(false);
          setMergeItem(null);
          mergeForm.resetFields();
        })}
        onOk={() => mergeForm.submit()}
        okText="Объединить"
        okButtonProps={{ danger: true }}
        confirmLoading={loading}
      >
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Alert
            type="warning"
            showIcon
            message="Документы будут перенесены на выбранную организацию, исходная запись будет удалена."
          />
          <Typography.Text>
            Исходная организация: <Typography.Text strong>{mergeItem?.name}</Typography.Text>
          </Typography.Text>
          <Form form={mergeForm} layout="vertical" onFinish={onMerge}>
            <Form.Item
              name="targetId"
              label="Объединить с"
              rules={[{ required: true, message: 'Выберите организацию' }]}
            >
              <Select
                showSearch
                placeholder="Выберите основную организацию"
                optionFilterProp="label"
                options={data
                  .filter(item => item.id !== mergeItem?.id)
                  .map(item => ({ value: item.id, label: item.name }))}
              />
            </Form.Item>
          </Form>
        </Space>
      </Modal>
    </div>
  );
};

const ResolutionExecutorsTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetResolutionExecutors } = await import('../../../wailsjs/go/services/ReferenceService');
      const items = await GetResolutionExecutors();
      setData(items || []);
    } catch (err: any) {
      message.error(formatAppError(err));
    }
    setLoading(false);
  }, [message]);

  useEffect(() => { load(); }, [load]);

  const onSave = async (values: any) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      if (editItem) {
        const { UpdateResolutionExecutor } = await import('../../../wailsjs/go/services/ReferenceService');
        await UpdateResolutionExecutor(editItem.id, values.name);
      }
      message.success('Исполнитель резолюции обновлён');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      await load();
    } catch (err: any) {
      message.error(formatAppError(err));
    } finally {
      setLoading(false);
    }
  };

  const onDelete = async (id: string) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      const { DeleteResolutionExecutor } = await import('../../../wailsjs/go/services/ReferenceService');
      await DeleteResolutionExecutor(id);
      message.success('Исполнитель резолюции удалён');
      await load();
    } catch (err: any) {
      message.error(formatAppError(err));
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    { title: 'ФИО', dataIndex: 'name', key: 'name' },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" title="Редактировать исполнителя резолюции" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Popconfirm
            title={`Удалить исполнителя резолюции "${record.name}"?`}
            description="Это действие нельзя отменить. Исполнитель резолюции исчезнет из справочника."
            okText="Удалить"
            cancelText="Отмена"
            okButtonProps={{ danger: true, loading }}
            onConfirm={() => onDelete(record.id)}
          >
            <Button size="small" title="Удалить исполнителя резолюции" icon={<DeleteOutlined />} danger loading={loading} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />
      <Typography.Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
        Исполнители добавляются автоматически при регистрации документов
      </Typography.Text>

      <Modal
        title="Редактировать исполнителя резолюции"
        open={modalOpen}
        onCancel={() => confirmDiscardFormChanges(modal, form, () => {
          setModalOpen(false);
          setEditItem(null);
          form.resetFields();
        })}
        onOk={() => form.submit()}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="name" label="ФИО" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export const ReferenceDirectoriesTab: React.FC = () => (
  <Tabs
    defaultActiveKey="organizations"
    destroyOnHidden
    items={[
      { key: 'organizations', label: 'Организации', icon: <BankOutlined />, children: <OrganizationsTab /> },
      { key: 'resolutionExecutors', label: 'Исполнители', icon: <SolutionOutlined />, children: <ResolutionExecutorsTab /> },
    ]}
  />
);
