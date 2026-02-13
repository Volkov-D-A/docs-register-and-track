import React, { useState, useEffect } from 'react';
import {
  Tabs, Table, Button, Modal, Form, Input, InputNumber, Select, Space,
  Typography, Popconfirm, message, Switch, Tag,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';

const { Title } = Typography;

// === Номенклатура ===
const NomenclatureTab: React.FC = () => {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();
  const currentYear = new Date().getFullYear();
  const [filterYear, setFilterYear] = useState(currentYear);

  const load = async () => {
    setLoading(true);
    try {
      const { GetAll } = await import('../../wailsjs/go/services/NomenclatureService');
      const items = await GetAll(filterYear, '');
      setData(items || []);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, [filterYear]);

  const onSave = async (values: any) => {
    try {
      if (editItem) {
        const { Update } = await import('../../wailsjs/go/services/NomenclatureService');
        await Update(editItem.id, values.name, values.index, values.year, values.direction, values.isActive);
      } else {
        const { Create } = await import('../../wailsjs/go/services/NomenclatureService');
        await Create(values.name, values.index, values.year, values.direction);
      }
      message.success(editItem ? 'Обновлено' : 'Создано');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const onDelete = async (id: string) => {
    try {
      const { Delete } = await import('../../wailsjs/go/services/NomenclatureService');
      await Delete(id);
      message.success('Удалено');
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const columns = [
    { title: 'Индекс', dataIndex: 'index', key: 'index', width: 100 },
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    { title: 'Год', dataIndex: 'year', key: 'year', width: 80 },
    {
      title: 'Направление', dataIndex: 'direction', key: 'direction', width: 120,
      render: (v: string) => v === 'incoming' ? <Tag color="blue">Входящие</Tag> : <Tag color="green">Исходящие</Tag>,
    },
    { title: 'След. номер', dataIndex: 'nextNumber', key: 'nextNumber', width: 100 },
    {
      title: 'Активно', dataIndex: 'isActive', key: 'isActive', width: 80,
      render: (v: boolean) => v ? <Tag color="green">Да</Tag> : <Tag color="red">Нет</Tag>,
    },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Popconfirm title="Удалить?" onConfirm={() => onDelete(record.id)}>
            <Button size="small" icon={<DeleteOutlined />} danger />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Select value={filterYear} onChange={setFilterYear} style={{ width: 100 }}>
          {[currentYear - 1, currentYear, currentYear + 1].map(y => (
            <Select.Option key={y} value={y}>{y}</Select.Option>
          ))}
        </Select>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
          setEditItem(null);
          form.resetFields();
          form.setFieldsValue({ year: currentYear });
          setModalOpen(true);
        }}>Добавить</Button>
      </Space>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />

      <Modal
        title={editItem ? 'Редактировать дело' : 'Новое дело'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditItem(null); }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="index" label="Индекс" rules={[{ required: true }]}>
            <Input placeholder="01-01" />
          </Form.Item>
          <Form.Item name="name" label="Наименование" rules={[{ required: true }]}>
            <Input placeholder="Входящая корреспонденция" />
          </Form.Item>
          <Form.Item name="year" label="Год" rules={[{ required: true }]}>
            <InputNumber min={2020} max={2030} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="direction" label="Направление" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="incoming">Входящие</Select.Option>
              <Select.Option value="outgoing">Исходящие</Select.Option>
            </Select>
          </Form.Item>
          {editItem && (
            <Form.Item name="isActive" label="Активно" valuePropName="checked">
              <Switch />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
};

// === Типы документов ===
const DocumentTypesTab: React.FC = () => {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const { GetDocumentTypes } = await import('../../wailsjs/go/services/ReferenceService');
      const items = await GetDocumentTypes();
      setData(items || []);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const onSave = async (values: any) => {
    try {
      if (editItem) {
        const { UpdateDocumentType } = await import('../../wailsjs/go/services/ReferenceService');
        await UpdateDocumentType(editItem.id, values.name);
      } else {
        const { CreateDocumentType } = await import('../../wailsjs/go/services/ReferenceService');
        await CreateDocumentType(values.name);
      }
      message.success(editItem ? 'Обновлено' : 'Создано');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const onDelete = async (id: string) => {
    try {
      const { DeleteDocumentType } = await import('../../wailsjs/go/services/ReferenceService');
      await DeleteDocumentType(id);
      message.success('Удалено');
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const columns = [
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Popconfirm title="Удалить?" onConfirm={() => onDelete(record.id)}>
            <Button size="small" icon={<DeleteOutlined />} danger />
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
      }} style={{ marginBottom: 16 }}>Добавить тип</Button>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />

      <Modal
        title={editItem ? 'Редактировать тип' : 'Новый тип документа'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditItem(null); }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="name" label="Наименование" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// === Организации ===
const OrganizationsTab: React.FC = () => {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const { GetOrganizations } = await import('../../wailsjs/go/services/ReferenceService');
      const items = await GetOrganizations();
      setData(items || []);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const onSave = async (values: any) => {
    try {
      if (editItem) {
        const { UpdateOrganization } = await import('../../wailsjs/go/services/ReferenceService');
        await UpdateOrganization(editItem.id, values.name);
      }
      message.success('Обновлено');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const onDelete = async (id: string) => {
    try {
      const { DeleteOrganization } = await import('../../wailsjs/go/services/ReferenceService');
      await DeleteOrganization(id);
      message.success('Удалено');
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const columns = [
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Popconfirm title="Удалить?" onConfirm={() => onDelete(record.id)}>
            <Button size="small" icon={<DeleteOutlined />} danger />
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
        onCancel={() => { setModalOpen(false); setEditItem(null); }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="name" label="Наименование" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// === Подразделения ===
const DepartmentsTab: React.FC = () => {
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();

  const [nomenclatureList, setNomenclatureList] = useState<any[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const { GetAllDepartments } = await import('../../wailsjs/go/services/DepartmentService');
      const { GetAll } = await import('../../wailsjs/go/services/NomenclatureService');

      const [items, nomenclature] = await Promise.all([
        GetAllDepartments(),
        GetAll(0, ""), // Fetch all nomenclature
      ]);

      setData(items || []);
      setNomenclatureList(nomenclature || []);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const onSave = async (values: any) => {
    try {
      if (editItem) {
        const { UpdateDepartment } = await import('../../wailsjs/go/services/DepartmentService');
        await UpdateDepartment(editItem.id, values.name, values.nomenclatureIds);
      } else {
        const { CreateDepartment } = await import('../../wailsjs/go/services/DepartmentService');
        await CreateDepartment(values.name, values.nomenclatureIds);
      }
      message.success(editItem ? 'Обновлено' : 'Создано');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const onDelete = async (id: string) => {
    try {
      const { DeleteDepartment } = await import('../../wailsjs/go/services/DepartmentService');
      await DeleteDepartment(id);
      message.success('Удалено');
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const columns = [
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Popconfirm title="Удалить?" onConfirm={() => onDelete(record.id)}>
            <Button size="small" icon={<DeleteOutlined />} danger />
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
        onCancel={() => { setModalOpen(false); setEditItem(null); }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="name" label="Наименование" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="nomenclatureIds" label="Видимые дела">
            <Select mode="multiple" optionFilterProp="children" showSearch>
              {nomenclatureList.map(n => (
                <Select.Option key={n.id} value={n.id}>
                  {n.index} {n.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// === Пользователи ===
const UsersTab: React.FC = () => {
  const [data, setData] = useState<any[]>([]);
  const [departments, setDepartments] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const { GetAllUsers } = await import('../../wailsjs/go/services/UserService');
      const { GetAllDepartments } = await import('../../wailsjs/go/services/DepartmentService');

      const [users, deps] = await Promise.all([GetAllUsers(), GetAllDepartments()]);
      setData(users || []);
      setDepartments(deps || []);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const onSave = async (values: any) => {
    try {
      if (editItem) {
        const { UpdateUser } = await import('../../wailsjs/go/services/UserService');
        await UpdateUser({ id: editItem.id, ...values });
      } else {
        const { CreateUser } = await import('../../wailsjs/go/services/UserService');
        await CreateUser(values);
      }
      message.success(editItem ? 'Обновлено' : 'Создано');
      setModalOpen(false);
      form.resetFields();
      setEditItem(null);
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const roleLabels: Record<string, string> = {
    admin: 'Администратор',
    clerk: 'Делопроизводитель',
    executor: 'Исполнитель',
  };

  const columns = [
    { title: 'Логин', dataIndex: 'login', key: 'login', width: 150 },
    { title: 'ФИО', dataIndex: 'fullName', key: 'fullName' },
    {
      title: 'Подразделение', dataIndex: 'department', key: 'department',
      render: (dep: any) => dep?.name || '-',
    },
    {
      title: 'Роли', dataIndex: 'roles', key: 'roles',
      render: (roles: string[]) => (roles || []).map(r => (
        <Tag key={r} color={r === 'admin' ? 'red' : r === 'clerk' ? 'blue' : 'green'}>
          {roleLabels[r] || r}
        </Tag>
      )),
    },
    {
      title: 'Активен', dataIndex: 'isActive', key: 'isActive', width: 80,
      render: (v: boolean) => v ? <Tag color="green">Да</Tag> : <Tag color="red">Нет</Tag>,
    },
    {
      title: 'Действия', key: 'actions', width: 80,
      render: (_: any, record: any) => (
        <Button size="small" icon={<EditOutlined />} onClick={() => {
          setEditItem(record);
          form.setFieldsValue({ ...record });
          setModalOpen(true);
        }} />
      ),
    },
  ];

  return (
    <div>
      <Button type="primary" icon={<PlusOutlined />} onClick={() => {
        setEditItem(null);
        form.resetFields();
        setModalOpen(true);
      }} style={{ marginBottom: 16 }}>Новый пользователь</Button>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />

      <Modal
        title={editItem ? 'Редактировать пользователя' : 'Новый пользователь'}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditItem(null); }}
        onOk={() => form.submit()}
        width={500}
      >
        <Form form={form} layout="vertical" onFinish={onSave}>
          <Form.Item name="login" label="Логин" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          {!editItem && (
            <Form.Item name="password" label="Пароль" rules={[{ required: true, min: 6 }]}>
              <Input.Password />
            </Form.Item>
          )}
          <Form.Item name="fullName" label="ФИО" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="departmentId" label="Подразделение" rules={[{ required: true }]}>
            <Select showSearch optionFilterProp="children">
              {departments.map(d => (
                <Select.Option key={d.id} value={d.id}>{d.name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="roles" label="Роли" rules={[{ required: true }]}>
            <Select mode="multiple">
              <Select.Option value="admin">Администратор</Select.Option>
              <Select.Option value="clerk">Делопроизводитель</Select.Option>
              <Select.Option value="executor">Исполнитель</Select.Option>
            </Select>
          </Form.Item>
          {editItem && (
            <Form.Item name="isActive" label="Активен" valuePropName="checked">
              <Switch />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
};

// === Основная страница ===
const SettingsPage: React.FC = () => {
  return (
    <div>
      <Title level={4}>Настройки</Title>
      <Tabs
        defaultActiveKey="nomenclature"
        items={[
          { key: 'nomenclature', label: 'Номенклатура дел', children: <NomenclatureTab /> },
          { key: 'documentTypes', label: 'Типы документов', children: <DocumentTypesTab /> },
          { key: 'organizations', label: 'Организации', children: <OrganizationsTab /> },
          { key: 'departments', label: 'Подразделения', children: <DepartmentsTab /> },
          { key: 'users', label: 'Пользователи', children: <UsersTab /> },
        ]}
      />
    </div>
  );
};

export default SettingsPage;
