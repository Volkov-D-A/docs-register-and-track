import React, { useCallback, useEffect, useState } from 'react';
import { App, Button, DatePicker, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag } from 'antd';
import { DeleteOutlined, EditOutlined, FileAddOutlined, PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import locale from 'antd/es/date-picker/locale/ru_RU';
import { DOCUMENT_KIND_INCOMING_LETTER, getDocumentKindLabel, getDocumentKindMeta } from '../../constants/documentKinds';
import { useCurrentAccessSummary } from '../../hooks/useCurrentAccessSummary';
import { formatAppError } from '../../utils/appError';
import { confirmDiscardFormChanges } from '../../utils/dirtyForm';
import { services } from '../../../wailsjs/go/models';

const NomenclatureTab: React.FC = () => {
  const { message, modal } = App.useApp();
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editItem, setEditItem] = useState<any>(null);
  const [adminCreateModalOpen, setAdminCreateModalOpen] = useState(false);
  const [adminCreateItem, setAdminCreateItem] = useState<any>(null);
  const [adminCreateLoading, setAdminCreateLoading] = useState(false);
  const [form] = Form.useForm();
  const [adminCreateForm] = Form.useForm();
  const currentYear = new Date().getFullYear();
  const [filterYear, setFilterYear] = useState(currentYear);
  const { kinds: allDocumentKinds } = useCurrentAccessSummary();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { GetAll } = await import('../../../wailsjs/go/services/NomenclatureService');
      const items = await GetAll(filterYear, '');
      setData(items || []);
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  }, [filterYear, message]);

  useEffect(() => { void load(); }, [load]);

  const onSave = async (values: any) => {
    if (loading) {
      return;
    }
    setLoading(true);
    try {
      if (editItem) {
        const { Update } = await import('../../../wailsjs/go/services/NomenclatureService');
        await Update(editItem.id, values.name, values.index, values.year, values.kindCode, values.separator, values.numberingMode, values.isActive);
      } else {
        const { Create } = await import('../../../wailsjs/go/services/NomenclatureService');
        const startNumber = typeof values.nextNumber === 'number' ? values.nextNumber : 1;
        await Create(values.name, values.index, values.year, values.kindCode, values.separator, values.numberingMode, startNumber);
      }
      message.success(editItem ? 'Правило нумерации обновлено' : 'Правило нумерации создано');
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
      const { Delete } = await import('../../../wailsjs/go/services/NomenclatureService');
      await Delete(id);
      message.success('Правило нумерации удалено');
      await load();
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setLoading(false);
    }
  };

  const openAdminCreate = (record: any) => {
    setAdminCreateItem(record);
    adminCreateForm.resetFields();
    adminCreateForm.setFieldsValue({
      mode: 'insert_shift',
      number: record.nextNumber || 1,
      suffix: '',
      registrationDate: dayjs(),
    });
    setAdminCreateModalOpen(true);
  };

  const onAdminCreate = async (values: any) => {
    if (!adminCreateItem) {
      return;
    }
    setAdminCreateLoading(true);
    try {
      const { CreateAdminDraft } = await import('../../../wailsjs/go/services/DocumentRegistrationService');
      await CreateAdminDraft(adminCreateItem.kindCode, services.AdminDraftCreateRequest.createFrom({
        nomenclatureId: adminCreateItem.id,
        registrationDate: values.registrationDate?.format('YYYY-MM-DD') || '',
        adminNumberOverride: services.AdminNumberOverrideRequest.createFrom({
          mode: values.mode,
          number: values.number,
          suffix: values.mode === 'literal' ? values.suffix || '' : '',
        }),
      }));
      message.success('Черновик документа создан');
      setAdminCreateModalOpen(false);
      setAdminCreateItem(null);
      adminCreateForm.resetFields();
      await load();
    } catch (error: unknown) {
      message.error(formatAppError(error));
    } finally {
      setAdminCreateLoading(false);
    }
  };

  const columns = [
    { title: 'Индекс', dataIndex: 'index', key: 'index', width: 100 },
    { title: 'Наименование', dataIndex: 'name', key: 'name' },
    { title: 'Год', dataIndex: 'year', key: 'year', width: 80 },
    {
      title: 'Вид документа', dataIndex: 'kindCode', key: 'kindCode', width: 160,
      render: (value: string) => {
        const meta = getDocumentKindMeta(value);
        return <Tag color={meta?.color || 'blue'}>{getDocumentKindLabel(value)}</Tag>;
      },
    },
    { title: 'Разделитель', dataIndex: 'separator', key: 'separator', width: 110 },
    {
      title: 'Нумерация', dataIndex: 'numberingMode', key: 'numberingMode', width: 160,
      render: (value: string) => (
        <Tag>
          {value === 'manual_only' ? 'Вручную' : value === 'number_only' ? 'Только номер' : 'Индекс + номер'}
        </Tag>
      ),
    },
    { title: 'След. номер', dataIndex: 'nextNumber', key: 'nextNumber', width: 100 },
    {
      title: 'Активно', dataIndex: 'isActive', key: 'isActive', width: 80,
      render: (value: boolean) => value ? <Tag color="green">Да</Tag> : <Tag color="red">Нет</Tag>,
    },
    {
      title: 'Действия', key: 'actions', width: 100,
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" title="Редактировать правило нумерации" icon={<EditOutlined />} onClick={() => {
            setEditItem(record);
            form.setFieldsValue(record);
            setModalOpen(true);
          }} />
          <Button
            size="small"
            title={record.isActive ? 'Создать документ с особым номером' : 'Дело не активно'}
            icon={<FileAddOutlined />}
            disabled={!record.isActive}
            onClick={() => openAdminCreate(record)}
          />
          <Popconfirm
            title={`Удалить правило нумерации для ${getDocumentKindLabel(record.kindCode)} за ${record.year} год?`}
            description="Это действие нельзя отменить. Новые документы больше не будут использовать это правило."
            okText="Удалить"
            cancelText="Отмена"
            okButtonProps={{ danger: true, loading }}
            onConfirm={() => onDelete(record.id)}
          >
            <Button size="small" title="Удалить правило нумерации" icon={<DeleteOutlined />} danger loading={loading} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Select value={filterYear} onChange={setFilterYear} style={{ width: 100 }}>
          {[currentYear - 1, currentYear, currentYear + 1].map((year) => (
            <Select.Option key={year} value={year}>{year}</Select.Option>
          ))}
        </Select>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => {
          setEditItem(null);
          form.resetFields();
          form.setFieldsValue({ year: currentYear, kindCode: DOCUMENT_KIND_INCOMING_LETTER, separator: '/', numberingMode: 'index_and_number', nextNumber: 1 });
          setModalOpen(true);
        }}>Добавить</Button>
      </Space>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} size="small" pagination={false} />

      <Modal
        title={editItem ? 'Редактировать дело' : 'Новое дело'}
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
          {!editItem && (
            <Form.Item name="nextNumber" label="Начать нумерацию с номера" rules={[{ required: true }]}>
              <InputNumber min={1} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          )}
          {editItem && (
            <Form.Item name="isActive" label="Активно" valuePropName="checked">
              <Switch />
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Modal
        title={adminCreateItem ? `Создать документ: ${adminCreateItem.index}` : 'Создать документ'}
        open={adminCreateModalOpen}
        onCancel={() => {
          setAdminCreateModalOpen(false);
          setAdminCreateItem(null);
          adminCreateForm.resetFields();
        }}
        onOk={() => adminCreateForm.submit()}
        okText="Создать черновик"
        confirmLoading={adminCreateLoading}
      >
        <Form form={adminCreateForm} layout="vertical" onFinish={onAdminCreate}>
          <Form.Item name="mode" label="Сценарий" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="insert_shift">Встраивание со сдвигом</Select.Option>
              <Select.Option value="literal">Литерный номер без сдвига</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="number" label="Номер" rules={[{ required: true }]}>
            <InputNumber min={1} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="registrationDate" label="Дата регистрации" rules={[{ required: true }]}>
            <DatePicker style={{ width: '100%' }} format="DD.MM.YYYY" locale={locale} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, next) => prev.mode !== next.mode}>
            {({ getFieldValue }) => (
              getFieldValue('mode') === 'literal' ? (
                <Form.Item name="suffix" label="Литера" rules={[{ required: true, message: 'Укажите литеру' }]}>
                  <Input maxLength={10} placeholder="А" />
                </Form.Item>
              ) : null
            )}
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default NomenclatureTab;
