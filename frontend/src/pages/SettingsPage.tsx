import React, { useState, useEffect } from 'react';
import {
  Tabs, Table, Button, Modal, Form, Input, InputNumber, Select, Space,
  Typography, Popconfirm, Switch, Tag, App, DatePicker, Checkbox, Row, Col, Collapse
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, KeyOutlined, DatabaseOutlined, CheckCircleOutlined, WarningOutlined, FileSearchOutlined, ReloadOutlined, BookOutlined, FileTextOutlined, BankOutlined, ApartmentOutlined, TeamOutlined, SettingOutlined, CloudServerOutlined, SolutionOutlined } from '@ant-design/icons';
import { DOCUMENT_KIND_INCOMING_LETTER, getDocumentKindLabel, getDocumentKindMeta } from '../constants/documentKinds';
import { useDocumentKinds } from '../hooks/useDocumentKinds';
import { useAuthStore } from '../store/useAuthStore';
import { models } from '../../wailsjs/go/models';

const { Title } = Typography;

// === Номенклатура ===
/**
 * Вкладка управления справочником номенклатуры дел.
 */
const NomenclatureTab: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();
  const currentYear = new Date().getFullYear();
  const [filterYear, setFilterYear] = useState(currentYear);
  const { kinds: allDocumentKinds } = useDocumentKinds();

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
        await Update(editItem.id, values.name, values.index, values.year, values.kindCode, values.separator, values.numberingMode, values.isActive);
      } else {
        const { Create } = await import('../../wailsjs/go/services/NomenclatureService');
        await Create(values.name, values.index, values.year, values.kindCode, values.separator, values.numberingMode);
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
      title: 'Вид документа', dataIndex: 'kindCode', key: 'kindCode', width: 160,
      render: (v: string) => {
        const meta = getDocumentKindMeta(v);
        return <Tag color={meta?.color || 'blue'}>{getDocumentKindLabel(v)}</Tag>;
      },
    },
    { title: 'Разделитель', dataIndex: 'separator', key: 'separator', width: 110 },
    {
      title: 'Нумерация', dataIndex: 'numberingMode', key: 'numberingMode', width: 160,
      render: (v: string) => (
        <Tag>
          {v === 'manual_only' ? 'Вручную' : v === 'number_only' ? 'Только номер' : 'Индекс + номер'}
        </Tag>
      ),
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
          form.setFieldsValue({ year: currentYear, kindCode: DOCUMENT_KIND_INCOMING_LETTER, separator: '/', numberingMode: 'index_and_number' });
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
          <Form.Item name="kindCode" label="Вид документа" rules={[{ required: true }]}>
            <Select>
              {allDocumentKinds.map((kind) => (
                <Select.Option key={kind.code} value={kind.code}>{kind.label}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="separator" label="Разделитель" rules={[{ required: true }]}>
            <Input maxLength={10} placeholder="/" />
          </Form.Item>
          <Form.Item name="numberingMode" label="Режим нумерации" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="index_and_number">Индекс + номер</Select.Option>
              <Select.Option value="number_only">Только номер</Select.Option>
              <Select.Option value="manual_only">Номер вводится вручную</Select.Option>
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
/**
 * Вкладка управления справочником типов документов.
 */
const DocumentTypesTab: React.FC = () => {
  const { message } = App.useApp();
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
/**
 * Вкладка управления справочником организаций.
 */
const OrganizationsTab: React.FC = () => {
  const { message } = App.useApp();
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

// === Исполнители резолюции ===
/**
 * Вкладка управления справочником исполнителей резолюции.
 */
const ResolutionExecutorsTab: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const { GetResolutionExecutors } = await import('../../wailsjs/go/services/ReferenceService');
      const items = await GetResolutionExecutors();
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
        const { UpdateResolutionExecutor } = await import('../../wailsjs/go/services/ReferenceService');
        await UpdateResolutionExecutor(editItem.id, values.name);
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
      const { DeleteResolutionExecutor } = await import('../../wailsjs/go/services/ReferenceService');
      await DeleteResolutionExecutor(id);
      message.success('Удалено');
      load();
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const columns = [
    { title: 'ФИО', dataIndex: 'name', key: 'name' },
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
        Исполнители добавляются автоматически при регистрации документов
      </Typography.Text>

      <Modal
        title="Редактировать исполнителя"
        open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditItem(null); }}
        onOk={() => form.submit()}
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

// === Подразделения ===
/**
 * Вкладка управления справочником подразделений организации.
 */
const DepartmentsTab: React.FC = () => {
  const { message } = App.useApp();
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
        GetAll(0, ""), // Загрузка всех номенклатур
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
/**
 * Вкладка управления пользователями системы (создание, роли, отделы, пароли).
 */
const UsersTab: React.FC = () => {
  const { message } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [departments, setDepartments] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [documentAccessCollapseKeys, setDocumentAccessCollapseKeys] = useState<string[]>([]);
  const [passwordForm] = Form.useForm();
  const [form] = Form.useForm();
  const { kinds: allDocumentKinds } = useDocumentKinds();

  const documentActionOptions = [
    { value: 'create', label: 'Регистрация' },
    { value: 'read', label: 'Просмотр всех' },
    { value: 'update', label: 'Редактирование' },
    { value: 'delete', label: 'Удаление' },
    { value: 'assign', label: 'Поручения' },
    { value: 'acknowledge', label: 'Ознакомления' },
    { value: 'upload', label: 'Управление файлами' },
    { value: 'link', label: 'Связи' },
    { value: 'view_journal', label: 'Журнал' },
  ];

  const buildEmptyDocumentAccess = () => (
    allDocumentKinds.reduce((acc: Record<string, { actions: string[] }>, kind) => {
      acc[kind.code] = { actions: [] };
      return acc;
    }, {})
  );

  const buildDocumentAccessFormValue = (profile?: any) => {
    const result = buildEmptyDocumentAccess();

    for (const permission of profile?.permissions || []) {
      if (!permission?.isAllowed || !result[permission.kindCode]) {
        continue;
      }
      result[permission.kindCode].actions.push(permission.action);
    }
    return result;
  };

  const buildSystemPermissionsFormValue = (profile?: any) => (
    (profile?.systemPermissions || [])
      .filter((permission: any) => permission?.isAllowed)
      .filter((permission: any) => !String(permission.permission || '').startsWith('stats_'))
      .map((permission: any) => permission.permission)
  );

  const buildStatisticsPermissionsFormValue = (profile?: any) => (
    (profile?.systemPermissions || [])
      .filter((permission: any) => permission?.isAllowed)
      .filter((permission: any) => String(permission.permission || '').startsWith('stats_'))
      .map((permission: any) => permission.permission)
  );

  const buildAccessRequest = (userId: string, values: any) => {
    const documentAccess = values.documentAccess || {};
    const systemPermissions = [...(values.systemPermissions || []), ...(values.statisticsPermissions || [])]
      .map((permission: string) => ({ permission, isAllowed: true }));
    const permissions: any[] = [];

    Object.entries(documentAccess).forEach(([kindCode, config]: [string, any]) => {
      for (const action of config?.actions || []) {
        permissions.push({ kindCode, action, isAllowed: true });
      }
    });

    return models.UpdateUserDocumentAccessRequest.createFrom({
      userId,
      systemPermissions,
      permissions,
    });
  };

  const openCreateModal = () => {
    setEditItem(null);
    setDocumentAccessCollapseKeys([]);
    form.resetFields();
    form.setFieldsValue({ documentAccess: buildEmptyDocumentAccess(), systemPermissions: [], statisticsPermissions: [], isActive: true, isDocumentParticipant: false });
    setModalOpen(true);
  };

  const openEditModal = async (record: any) => {
    setEditItem(record);
    setDocumentAccessCollapseKeys([]);
    form.resetFields();
    form.setFieldsValue({ ...record, departmentId: record.department?.id, documentAccess: buildEmptyDocumentAccess(), systemPermissions: [], statisticsPermissions: [], isDocumentParticipant: record.isDocumentParticipant });
    setModalOpen(true);

    try {
      const { GetUserAccessProfile } = await import('../../wailsjs/go/services/DocumentAccessAdminService');
      const profile = await GetUserAccessProfile(record.id);
      form.setFieldsValue({
        ...record,
        departmentId: record.department?.id,
        systemPermissions: buildSystemPermissionsFormValue(profile),
        statisticsPermissions: buildStatisticsPermissionsFormValue(profile),
        documentAccess: buildDocumentAccessFormValue(profile),
      });
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

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
      const { UpdateUserAccessProfile } = await import('../../wailsjs/go/services/DocumentAccessAdminService');

      if (editItem) {
        const { UpdateUser } = await import('../../wailsjs/go/services/UserService');
        await UpdateUser({
          id: editItem.id,
          login: values.login,
          fullName: values.fullName,
          isActive: values.isActive,
          departmentId: values.departmentId,
          isDocumentParticipant: !!values.isDocumentParticipant,
        });
        await UpdateUserAccessProfile(buildAccessRequest(editItem.id, values));
      } else {
        const { CreateUser } = await import('../../wailsjs/go/services/UserService');
        const createdUser = await CreateUser({
          login: values.login,
          password: values.password,
          fullName: values.fullName,
          departmentId: values.departmentId,
          isDocumentParticipant: !!values.isDocumentParticipant,
        });
        await UpdateUserAccessProfile(buildAccessRequest(createdUser.id, values));
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

  const onPasswordChange = async (values: any) => {
    try {
      const { ResetPassword } = await import('../../wailsjs/go/services/UserService');
      await ResetPassword(editItem.id, values.newPassword);
      message.success('Пароль успешно изменен');
      setPasswordModalOpen(false);
      passwordForm.resetFields();
      setEditItem(null);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
  };

  const systemPermissionLabels: Record<string, string> = {
    admin: 'Администратор',
    references: 'Справочники',
    stats_incoming: 'Статистика: входящие письма',
    stats_outgoing: 'Статистика: исходящие письма',
    stats_assignments: 'Статистика: поручения',
    stats_system: 'Статистика: системная',
  };

  const isBruteforceLocked = (user: any) => !user?.isActive && (user?.failedLoginAttempts || 0) >= 5;

  const columns = [
    { title: 'Логин', dataIndex: 'login', key: 'login', width: 150 },
    { title: 'ФИО', dataIndex: 'fullName', key: 'fullName' },
    {
      title: 'Подразделение', dataIndex: 'department', key: 'department',
      render: (dep: any) => dep?.name || '-',
    },
    {
      title: 'Системные права', dataIndex: 'systemPermissions', key: 'systemPermissions',
      render: (permissions: string[]) => (permissions || []).map(permission => (
        <Tag key={permission} color={permission === 'admin' ? 'red' : 'blue'}>
          {systemPermissionLabels[permission] || permission}
        </Tag>
      )),
    },
    {
      title: 'Статус', key: 'status', width: 220,
      render: (_: any, record: any) => {
        if (isBruteforceLocked(record)) {
          return (
            <Space orientation="vertical" size={4}>
              <Tag color="volcano">Заблокирован</Tag>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                5 ошибок входа подряд
              </Typography.Text>
            </Space>
          );
        }

        return record.isActive
          ? <Tag color="green">Активен</Tag>
          : <Tag color="red">Отключен вручную</Tag>;
      },
    },
    {
      title: 'Действия', key: 'actions', width: 120,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => { void openEditModal(record); }} />
          <Button size="small" icon={<KeyOutlined />} onClick={() => {
            setEditItem(record);
            setPasswordModalOpen(true);
          }} title="Сменить пароль" />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal} style={{ marginBottom: 16 }}>Новый пользователь</Button>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />

      <Typography.Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
        Пользователь со статусом «Заблокирован» был деактивирован автоматически после 5 неверных попыток входа.
        Для восстановления откройте его карточку и снова включите флаг «Активен».
      </Typography.Text>

      <Modal
        title={editItem ? 'Редактировать пользователя' : 'Новый пользователь'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setEditItem(null);
          setDocumentAccessCollapseKeys([]);
        }}
        onOk={() => form.submit()}
        width={1100}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto', overflowX: 'hidden' } }}
      >
        <Form form={form} layout="vertical" onFinish={onSave} style={{ overflowX: 'hidden' }}>
          <Row gutter={24} align="top" wrap style={{ marginInline: 0 }}>
            <Col xs={24} lg={10}>
              <Typography.Title level={5}>Сведения о пользователе</Typography.Title>
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
              <Form.Item name="isDocumentParticipant" label="Участник документооборота" valuePropName="checked">
                <Switch />
              </Form.Item>
              {editItem && (
                <>
                  {isBruteforceLocked(editItem) && (
                    <Typography.Text type="warning" style={{ display: 'block', marginBottom: 12 }}>
                      Пользователь автоматически заблокирован после 5 неверных попыток входа. Включение флага «Активен» разблокирует его и сбросит счетчик ошибок.
                    </Typography.Text>
                  )}
                  <Form.Item name="isActive" label="Активен" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </>
              )}
            </Col>
            <Col xs={24} lg={14}>
              <Typography.Title level={5} style={{ marginBottom: 8 }}>Права доступа</Typography.Title>
              <Form.Item name="systemPermissions" label="Системные права" style={{ marginBottom: 8 }}>
                <Checkbox.Group options={[
                  { label: 'Администратор', value: 'admin' },
                  { label: 'Справочники', value: 'references' },
                ]} />
              </Form.Item>
              <Form.Item name="statisticsPermissions" label="Статистика" style={{ marginBottom: 8 }}>
                <Checkbox.Group options={[
                  { label: 'Входящие письма', value: 'stats_incoming' },
                  { label: 'Исходящие письма', value: 'stats_outgoing' },
                  { label: 'Поручения', value: 'stats_assignments' },
                  { label: 'Системная', value: 'stats_system' },
                ]} />
              </Form.Item>
              <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 6 }}>Права на документы</Typography.Title>
              <Collapse
                ghost
                size="small"
                destroyOnHidden
                activeKey={documentAccessCollapseKeys}
                onChange={(keys) => setDocumentAccessCollapseKeys(Array.isArray(keys) ? keys.map(String) : [String(keys)])}
                style={{ marginTop: 0 }}
                items={allDocumentKinds.map((kind) => ({
                  key: kind.code,
                  label: kind.label,
                  children: (
                    <Form.Item
                      name={['documentAccess', kind.code, 'actions']}
                      style={{ marginBottom: 0 }}
                    >
                      <Checkbox.Group options={documentActionOptions} />
                    </Form.Item>
                  ),
                }))}
              />
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        title={`Смена пароля для пользователя ${editItem?.login}`}
        open={passwordModalOpen}
        onCancel={() => { setPasswordModalOpen(false); setEditItem(null); passwordForm.resetFields(); }}
        onOk={() => passwordForm.submit()}
        width={400}
      >
        <Form form={passwordForm} layout="vertical" onFinish={onPasswordChange}>
          <Form.Item
            name="newPassword"
            label="Новый пароль"
            rules={[{ required: true, min: 6, message: 'Минимум 6 символов' }]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// === Системные настройки ===
/**
 * Вкладка основных (глобальных) системных настроек.
 */
const SystemSettingsTab: React.FC = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const { GetAll } = await import('../../wailsjs/go/services/SettingsService');
      const settings = await GetAll();
      if (settings) {
        const values: any = {};
        settings.forEach((s: any) => {
          if (s.key === 'assignment_completion_attachments_enabled') {
            values[s.key] = String(s.value).toLowerCase() === 'true';
            return;
          }
          values[s.key] = s.value;
        });
        if (values.assignment_completion_attachments_enabled === undefined) {
          values.assignment_completion_attachments_enabled = false;
        }
        form.setFieldsValue(values);
      }
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const onSave = async (values: any) => {
    setLoading(true);
    try {
      const { Update } = await import('../../wailsjs/go/services/SettingsService');
      // Сохранение каждой настройки
      for (const key in values) {
        const value = typeof values[key] === 'boolean' ? String(values[key]) : String(values[key]);
        await Update(key, value);

        // Автоматически добавляем в справочник организаций
        if (key === 'organization_name' && values[key] && String(values[key]).trim() !== '') {
          try {
            const { FindOrCreateOrganization } = await import('../../wailsjs/go/services/ReferenceService');
            await FindOrCreateOrganization(String(values[key]).trim());
          } catch (e) {
            console.error('Failed to add organization to directory:', e);
          }
        }
      }
      message.success('Настройки сохранены');
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
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
        <Form.Item name="max_file_size_mb" label="Максимальный размер файла (МБ)" rules={[{ required: true }]}>
          <InputNumber min={1} max={1000} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="allowed_file_types" label="Разрешенные типы файлов (через запятую)" rules={[{ required: true }]}>
          <Input placeholder=".pdf, .doc, .docx" />
        </Form.Item>
        <Form.Item
          name="assignment_completion_attachments_enabled"
          label="Файлы при завершении поручения"
          valuePropName="checked"
        >
          <Switch checkedChildren="Включено" unCheckedChildren="Выключено" />
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

// === Миграции БД ===
/**
 * Вкладка управления миграциями базы данных.
 */
const MigrationsTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);
  const [status, setStatus] = useState<any>(null);

  const loadStatus = async () => {
    setLoading(true);
    try {
      const { GetMigrationStatus } = await import('../../wailsjs/go/services/SettingsService');
      const result = await GetMigrationStatus();
      setStatus(result);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { loadStatus(); }, []);

  const onRunMigrations = () => {
    modal.confirm({
      title: 'Запуск миграций',
      content: 'Вы уверены, что хотите применить миграции базы данных? Убедитесь, что все пользователи завершили работу в системе.',
      okText: 'Запустить',
      cancelText: 'Отмена',
      okType: 'primary',
      onOk: async () => {
        setRunning(true);
        try {
          const { RunMigrations } = await import('../../wailsjs/go/services/SettingsService');
          await RunMigrations();
          message.success('Миграции успешно применены');
          await loadStatus();
        } catch (err: any) {
          message.error(err?.message || String(err));
        }
        setRunning(false);
      },
    });
  };

  const onRollback = () => {
    modal.confirm({
      title: 'Откат миграции',
      content: 'Вы уверены, что хотите откатить последнюю миграцию? Это может привести к потере данных, добавленных в новых таблицах/столбцах.',
      okText: 'Откатить',
      cancelText: 'Отмена',
      okType: 'primary',
      okButtonProps: { danger: true },
      onOk: async () => {
        setRollingBack(true);
        try {
          const { RollbackMigration } = await import('../../wailsjs/go/services/SettingsService');
          await RollbackMigration();
          message.success('Миграция успешно откачена');
          await loadStatus();
        } catch (err: any) {
          message.error(err?.message || String(err));
        }
        setRollingBack(false);
      },
    });
  };

  return (
    <div style={{ maxWidth: 600 }}>
      <div style={{
        padding: 24,
        background: '#fafafa',
        borderRadius: 8,
        border: '1px solid #f0f0f0',
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
              <Typography.Text>{status.totalAvailable}</Typography.Text>
            </div>
            <div>
              <Typography.Text strong>Статус: </Typography.Text>
              {status.upToDate ? (
                <Tag icon={<CheckCircleOutlined />} color="success">Актуальна</Tag>
              ) : status.dirty ? (
                <Tag icon={<WarningOutlined />} color="error">Ошибка миграции (dirty)</Tag>
              ) : (
                <Tag icon={<WarningOutlined />} color="warning">Требуется обновление</Tag>
              )}
            </div>
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
          disabled={status?.upToDate}
        >
          Применить миграции
        </Button>
        <Button
          danger
          icon={<DeleteOutlined />}
          onClick={onRollback}
          loading={rollingBack}
          disabled={!status?.currentVersion}
        >
          Откатить последнюю
        </Button>
      </Space>
      <div>
        <Typography.Text type="secondary">
          Перед запуском или откатом миграций убедитесь, что все пользователи завершили работу
        </Typography.Text>
      </div>
    </div>
  );
};

// === Управление хранилищем ===
/**
 * Вкладка для массового управления файлами: удаление файлов старше указанной даты.
 */
const StorageTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [selectedDate, setSelectedDate] = useState<any>(null);

  const onBulkDelete = () => {
    if (!selectedDate) {
      message.warning('Пожалуйста, выберите дату');
      return;
    }

    modal.confirm({
      title: 'Массовое удаление файлов',
      content: `Внимание! Это действие безвозвратно удалит все файлы, загруженные до ${selectedDate.format('DD.MM.YYYY')}. Продолжить?`,
      okText: 'Да, удалить',
      cancelText: 'Отмена',
      okType: 'danger',
      onOk: async () => {
        setLoading(true);
        try {
          const { BulkDeleteOlderThan } = await import('../../wailsjs/go/services/AttachmentService');
          const count = await BulkDeleteOlderThan(selectedDate.toISOString());
          message.success(`Успешно удалено файлов: ${count}`);
        } catch (err: any) {
          message.error(err?.message || String(err));
        }
        setLoading(false);
      },
    });
  };

  return (
    <div style={{ maxWidth: 600 }}>
      <div style={{
        padding: 24,
        background: '#fffbe6',
        borderRadius: 8,
        border: '1px solid #ffe58f',
        marginBottom: 16,
      }}>
        <Typography.Paragraph type="warning" strong style={{ marginBottom: 0 }}>
          <WarningOutlined style={{ marginRight: 8 }} />
          Осторожно: удаленные файлы невозможно восстановить средствами системы.
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

// === Журнал действий администраторов ===
/**
 * Вкладка отображения журнала действий администраторов в панели настроек.
 */
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

  const load = async (p: number) => {
    setLoading(true);
    try {
      const { GetAll } = await import('../../wailsjs/go/services/AdminAuditLogService');
      const result = await GetAll(p, pageSize);
      setData(result?.items || []);
      setTotal(result?.total || 0);
    } catch (err: any) {
      message.error(err?.message || String(err));
    }
    setLoading(false);
  };

  useEffect(() => { load(page); }, [page]);

  const columns = [
    {
      title: 'Дата и время', dataIndex: 'createdAt', key: 'createdAt', width: 170,
      render: (v: string) => v ? new Date(v).toLocaleString('ru-RU') : '-',
    },
    { title: 'Пользователь', dataIndex: 'userName', key: 'userName', width: 200 },
    {
      title: 'Действие', dataIndex: 'action', key: 'action', width: 220,
      render: (v: string) => {
        const label = actionLabels[v] || v;
        let color = 'default';
        if (v.includes('CREATE')) color = 'green';
        else if (v.includes('DELETE') || v.includes('ROLLBACK')) color = 'red';
        else if (v.includes('UPDATE') || v.includes('RESET')) color = 'blue';
        else if (v.includes('MIGRATION_RUN')) color = 'orange';
        return <Tag color={color}>{label}</Tag>;
      },
    },
    { title: 'Подробности', dataIndex: 'details', key: 'details' },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={() => load(page)}>Обновить</Button>
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
          onChange: (p) => setPage(p),
          showTotal: (t) => `Всего: ${t}`,
        }}
      />
    </div>
  );
};

export const ReferenceDirectoriesTab: React.FC = () => (
  <Tabs
    defaultActiveKey="documentTypes"
    destroyOnHidden
    items={[
      { key: 'documentTypes', label: 'Типы документов', icon: <FileTextOutlined />, children: <DocumentTypesTab /> },
      { key: 'organizations', label: 'Организации', icon: <BankOutlined />, children: <OrganizationsTab /> },
      { key: 'resolutionExecutors', label: 'Исполнители', icon: <SolutionOutlined />, children: <ResolutionExecutorsTab /> },
    ]}
  />
);

// === Основная страница ===
/**
 * Страница настроек системы. 
 * Объединяет все административные справочники и системные опции во вкладках.
 */
const SettingsPage: React.FC = () => {
  const { hasSystemPermission } = useAuthStore();
  const canAccessAdminSettings = hasSystemPermission('admin');
  const items = [
    ...(canAccessAdminSettings ? [
      { key: 'nomenclature', label: 'Номенклатура', icon: <BookOutlined />, children: <NomenclatureTab /> },
    ] : []),
    ...(canAccessAdminSettings ? [
      { key: 'departments', label: 'Отделы', icon: <ApartmentOutlined />, children: <DepartmentsTab /> },
      { key: 'users', label: 'Пользователи', icon: <TeamOutlined />, children: <UsersTab /> },
      { key: 'system', label: 'Настройки', icon: <SettingOutlined />, children: <SystemSettingsTab /> },
      { key: 'storage', label: 'Хранилище', icon: <CloudServerOutlined />, children: <StorageTab /> },
      { key: 'migrations', label: 'Миграции', icon: <DatabaseOutlined />, children: <MigrationsTab /> },
      { key: 'auditLog', label: 'Журнал', icon: <FileSearchOutlined />, children: <AuditLogTab /> },
    ] : []),
  ];

  return (
    <div>
      <Title level={4}>Настройки</Title>
      <Tabs
        defaultActiveKey={items[0]?.key}
        destroyOnHidden
        items={items}
      />
    </div>
  );
};

export default SettingsPage;
