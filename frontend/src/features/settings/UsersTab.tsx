import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, Checkbox, Collapse, DatePicker, Form, Input, Modal, Row, Col, Select, Space, Switch, Table, Tag, Typography } from 'antd';
import { EditOutlined, KeyOutlined, PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { models } from '../../../wailsjs/go/models';
import { useCurrentAccessSummary } from '../../hooks/useCurrentAccessSummary';
import { formatAppError } from '../../utils/appError';
import { confirmDiscardFormChanges } from '../../utils/dirtyForm';

const UsersTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [departments, setDepartments] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [documentAccessCollapseKeys, setDocumentAccessCollapseKeys] = useState<string[]>([]);
  const [passwordForm] = Form.useForm();
  const [form] = Form.useForm();
  const { kinds: allDocumentKinds } = useCurrentAccessSummary();

  const documentActionOptions = [
    { value: 'create', label: 'Регистрация' },
    { value: 'read', label: 'Просмотр всех' },
    { value: 'update', label: 'Редактирование' },
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
    form.setFieldsValue({ documentAccess: buildEmptyDocumentAccess(), systemPermissions: [], statisticsPermissions: [], isActive: true, isDocumentParticipant: false, substitutionActive: true });
    setModalOpen(true);
  };

  const openEditModal = async (record: any) => {
    setEditItem(record);
    setDocumentAccessCollapseKeys([]);
    form.resetFields();
    form.setFieldsValue({ ...record, departmentId: record.department?.id, documentAccess: buildEmptyDocumentAccess(), systemPermissions: [], statisticsPermissions: [], isDocumentParticipant: record.isDocumentParticipant });
    setModalOpen(true);

    try {
      const { GetUserAccessProfile } = await import('../../../wailsjs/go/services/DocumentAccessAdminService');
      const { GetUserSubstitution } = await import('../../../wailsjs/go/services/UserSubstitutionService');
      const [profile, substitution] = await Promise.all([
        GetUserAccessProfile(record.id),
        GetUserSubstitution(record.id),
      ]);
      form.setFieldsValue({
        ...record,
        departmentId: record.department?.id,
        systemPermissions: buildSystemPermissionsFormValue(profile),
        statisticsPermissions: buildStatisticsPermissionsFormValue(profile),
        documentAccess: buildDocumentAccessFormValue(profile),
        substituteUserId: substitution?.substituteUserId || undefined,
        substitutionActive: substitution?.isActive ?? true,
        substitutionPeriod: substitution?.startsAt || substitution?.endsAt
          ? [substitution?.startsAt ? dayjs(substitution.startsAt) : null, substitution?.endsAt ? dayjs(substitution.endsAt) : null]
          : null,
      });
    } catch (error: unknown) {
      message.error(formatAppError(error));
    }
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetAllUsers } = await import('../../../wailsjs/go/services/UserService');
      const { GetAllDepartments } = await import('../../../wailsjs/go/services/DepartmentService');

      const [users, deps] = await Promise.all([GetAllUsers(), GetAllDepartments()]);
      setData(users || []);
      setDepartments(deps || []);
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
      const { UpdateUserAccessProfile } = await import('../../../wailsjs/go/services/DocumentAccessAdminService');
      const { UpdateUserSubstitution } = await import('../../../wailsjs/go/services/UserSubstitutionService');
      let generatedTemporaryPassword = '';

      if (editItem) {
        const { UpdateUser } = await import('../../../wailsjs/go/services/UserService');
        await UpdateUser({
          id: editItem.id,
          login: values.login,
          fullName: values.fullName,
          isActive: values.isActive,
          departmentId: values.departmentId,
          isDocumentParticipant: !!values.isDocumentParticipant,
        });
        await UpdateUserAccessProfile(buildAccessRequest(editItem.id, values));
        await UpdateUserSubstitution({
          principalUserId: editItem.id,
          substituteUserId: values.isDocumentParticipant ? (values.substituteUserId || '') : '',
          startsAt: values.isDocumentParticipant && values.substitutionPeriod?.[0] ? values.substitutionPeriod[0].format('YYYY-MM-DD') : '',
          endsAt: values.isDocumentParticipant && values.substitutionPeriod?.[1] ? values.substitutionPeriod[1].format('YYYY-MM-DD') : '',
          isActive: values.substitutionActive ?? true,
        });
      } else {
        const { CreateUser } = await import('../../../wailsjs/go/services/UserService');
        const createdUser = await CreateUser({
          login: values.login,
          password: values.password || '',
          fullName: values.fullName,
          departmentId: values.departmentId,
          isDocumentParticipant: !!values.isDocumentParticipant,
        });
        await UpdateUserAccessProfile(buildAccessRequest(createdUser.id, values));
        await UpdateUserSubstitution({
          principalUserId: createdUser.id,
          substituteUserId: values.isDocumentParticipant ? (values.substituteUserId || '') : '',
          startsAt: values.isDocumentParticipant && values.substitutionPeriod?.[0] ? values.substitutionPeriod[0].format('YYYY-MM-DD') : '',
          endsAt: values.isDocumentParticipant && values.substitutionPeriod?.[1] ? values.substitutionPeriod[1].format('YYYY-MM-DD') : '',
          isActive: values.substitutionActive ?? true,
        });
        generatedTemporaryPassword = createdUser.temporaryPassword || '';
      }
      message.success(editItem ? 'Пользователь обновлён' : 'Пользователь создан');
      if (!editItem && generatedTemporaryPassword) {
        modal.info({
          title: 'Временный пароль',
          content: (
            <Space orientation="vertical" size="small">
              <Typography.Text>
                Передайте пользователю временный пароль. При первом входе система потребует его сменить.
              </Typography.Text>
              <Typography.Text copyable code>
                {generatedTemporaryPassword}
              </Typography.Text>
            </Space>
          ),
          okText: 'Готово',
        });
      }
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

  const onPasswordChange = async (values: any) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      const { ResetPassword } = await import('../../../wailsjs/go/services/UserService');
      await ResetPassword(editItem.id, values.newPassword);
      message.success('Пароль успешно изменён');
      setPasswordModalOpen(false);
      passwordForm.resetFields();
      setEditItem(null);
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  };

  const systemPermissionLabels: Record<string, string> = {
    admin: 'Администратор',
    references: 'Справочники',
    stats_documents: 'Статистика: документы',
    stats_assignments: 'Статистика: поручения',
    stats_system: 'Статистика: системная',
  };

  const isBruteforceLocked = (user: any) => !user?.isActive && (user?.failedLoginAttempts || 0) >= 5;

  const compactUserFormItemStyle = { marginBottom: 12 };

  const columns = [
    { title: 'Логин', dataIndex: 'login', key: 'login', width: 150 },
    { title: 'ФИО', dataIndex: 'fullName', key: 'fullName' },
    {
      title: 'Подразделение', dataIndex: 'department', key: 'department',
      render: (dep: any) => dep?.name || '-',
    },
    {
      title: 'Системные права', dataIndex: 'systemPermissions', key: 'systemPermissions',
      render: (permissions: string[]) => (permissions || []).map((permission) => (
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
          <Button size="small" title="Редактировать пользователя" icon={<EditOutlined />} onClick={() => { void openEditModal(record); }} />
          <Button size="small" title="Сменить пароль пользователя" icon={<KeyOutlined />} onClick={() => {
            setEditItem(record);
            setPasswordModalOpen(true);
          }} />
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
        onCancel={() => confirmDiscardFormChanges(modal, form, () => {
          setModalOpen(false);
          setEditItem(null);
          setDocumentAccessCollapseKeys([]);
          form.resetFields();
        })}
        onOk={() => form.submit()}
        width={1100}
        confirmLoading={loading}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto', overflowX: 'hidden' } }}
      >
        <Form form={form} layout="vertical" onFinish={onSave} style={{ overflowX: 'hidden' }}>
          <Row gutter={24} align="top" wrap style={{ marginInline: 0 }}>
            <Col xs={24} lg={10}>
              <Typography.Title level={5} style={{ marginBottom: 12 }}>Сведения о пользователе</Typography.Title>
              <Form.Item name="login" label="Логин" rules={[{ required: true }]} style={compactUserFormItemStyle}>
                <Input />
              </Form.Item>
              {!editItem && (
                <Form.Item
                  name="password"
                  label="Временный пароль"
                  rules={[{ min: 8, message: 'Минимум 8 символов' }]}
                  style={compactUserFormItemStyle}
                >
                  <Input.Password placeholder="Сгенерировать автоматически" />
                </Form.Item>
              )}
              <Form.Item name="fullName" label="ФИО" rules={[{ required: true }]} style={compactUserFormItemStyle}>
                <Input />
              </Form.Item>
              <Form.Item name="departmentId" label="Подразделение" rules={[{ required: true }]} style={compactUserFormItemStyle}>
                <Select
                  showSearch
                  optionFilterProp="children"
                  onChange={() => form.setFieldsValue({ substituteUserId: undefined, substitutionPeriod: null })}
                >
                  {departments.map((department) => (
                    <Select.Option key={department.id} value={department.id}>{department.name}</Select.Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="isDocumentParticipant" label="Участник документооборота" valuePropName="checked" style={compactUserFormItemStyle}>
                <Switch />
              </Form.Item>
              <Form.Item shouldUpdate={(prev, next) => prev.isDocumentParticipant !== next.isDocumentParticipant} noStyle>
                {({ getFieldValue }) => {
                  const participant = !!getFieldValue('isDocumentParticipant');
                  const departmentID = getFieldValue('departmentId');
                  return (
                    <>
                      <Form.Item name="substituteUserId" label="Замещающий" style={compactUserFormItemStyle}>
                        <Select
                          allowClear
                          showSearch
                          disabled={!participant}
                          optionFilterProp="label"
                          placeholder="Выберите сотрудника"
                          options={data
                            .filter((user) => user.id !== editItem?.id && user.isActive && user.department?.id === departmentID)
                            .map((user) => ({ value: user.id, label: user.fullName }))}
                        />
                      </Form.Item>
                      <Form.Item name="substitutionPeriod" label="Период замещения" style={compactUserFormItemStyle}>
                        <DatePicker.RangePicker
                          style={{ width: '100%' }}
                          format="DD.MM.YYYY"
                          allowEmpty={[true, true]}
                          disabled={!participant}
                        />
                      </Form.Item>
                      <Form.Item name="substitutionActive" label="Замещение активно" valuePropName="checked" style={compactUserFormItemStyle}>
                        <Switch disabled={!participant} />
                      </Form.Item>
                    </>
                  );
                }}
              </Form.Item>
              {editItem && (
                <>
                  {isBruteforceLocked(editItem) && (
                    <Typography.Text type="warning" style={{ display: 'block', marginBottom: 12 }}>
                      Пользователь автоматически заблокирован после 5 неверных попыток входа. Включение флага «Активен» разблокирует его и сбросит счетчик ошибок.
                    </Typography.Text>
                  )}
                  <Form.Item name="isActive" label="Активен" valuePropName="checked" style={compactUserFormItemStyle}>
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
                  { label: 'Документы', value: 'stats_documents' },
                  { label: 'Поручения', value: 'stats_assignments' },
                  { label: 'Системная', value: 'stats_system' },
                ]} />
              </Form.Item>
              <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 6 }}>Права на документы</Typography.Title>
              <Collapse
                ghost
                size="small"
                activeKey={documentAccessCollapseKeys}
                onChange={(keys) => setDocumentAccessCollapseKeys(Array.isArray(keys) ? keys.map(String) : [String(keys)])}
                style={{ marginTop: 0 }}
                items={allDocumentKinds.map((kind) => ({
                  key: kind.code,
                  label: kind.label,
                  forceRender: true,
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
        onCancel={() => confirmDiscardFormChanges(modal, passwordForm, () => {
          setPasswordModalOpen(false);
          setEditItem(null);
          passwordForm.resetFields();
        })}
        onOk={() => passwordForm.submit()}
        width={400}
        confirmLoading={loading}
      >
        <Form form={passwordForm} layout="vertical" onFinish={onPasswordChange}>
          <Form.Item
            name="newPassword"
            label="Новый пароль"
            rules={[{ required: true, min: 8, message: 'Минимум 8 символов' }]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default UsersTab;
