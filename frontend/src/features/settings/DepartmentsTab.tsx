import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Form, Input, Modal, Popconfirm, Select, Space, Table } from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { formatAppError } from '../../utils/appError';
import { confirmDiscardFormChanges } from '../../utils/dirtyForm';

const DepartmentsTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();
  const [nomenclatureList, setNomenclatureList] = useState<any[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetAllDepartments } = await import('../../../wailsjs/go/services/DepartmentService');
      const { GetAll } = await import('../../../wailsjs/go/services/NomenclatureService');

      const [items, nomenclature] = await Promise.all([
        GetAllDepartments(),
        GetAll(0, ''),
      ]);

      setData(items || []);
      setNomenclatureList(nomenclature || []);
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => { void load(); }, [load]);

  const onSave = async (values: any) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      if (editItem) {
        const { UpdateDepartment } = await import('../../../wailsjs/go/services/DepartmentService');
        await UpdateDepartment(editItem.id, values.name, values.nomenclatureIds);
      } else {
        const { CreateDepartment } = await import('../../../wailsjs/go/services/DepartmentService');
        await CreateDepartment(values.name, values.nomenclatureIds);
      }
      message.success(editItem ? 'Подразделение обновлено' : 'Подразделение создано');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      await load();
    } catch (error: unknown) {
      message.error(formatAppError(error));
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
      const { DeleteDepartment } = await import('../../../wailsjs/go/services/DepartmentService');
      await DeleteDepartment(id);
      message.success('Подразделение удалено');
      await load();
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" title="Редактировать подразделение" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Popconfirm
            title={`Удалить подразделение "${record.name}"?`}
            description="Это действие нельзя отменить. Подразделение исчезнет из справочника."
            okText="Удалить"
            cancelText="Отмена"
            okButtonProps={{ danger: true, loading }}
            onConfirm={() => onDelete(record.id)}
          >
            <Button size="small" title="Удалить подразделение" icon={<DeleteOutlined />} danger loading={loading} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Button type="primary" icon={<PlusOutlined />} onClick={() => {
        setEditItem(null);
        form.resetFields();
        setModalOpen(true);
      }} style={{ marginBottom: 16 }}>Добавить подразделение</Button>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />

      <Modal
        title={editItem ? 'Редактировать подразделение' : 'Новое подразделение'}
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
          <Form.Item name="nomenclatureIds" label="Видимые дела">
            <Select mode="multiple" optionFilterProp="children" showSearch>
              {nomenclatureList.map((nomenclature) => (
                <Select.Option key={nomenclature.id} value={nomenclature.id}>
                  {nomenclature.index} {nomenclature.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DepartmentsTab;
