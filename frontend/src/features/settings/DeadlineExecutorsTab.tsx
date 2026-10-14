import { useCallback, useEffect, useRef, useState } from 'react';
import { App, Button, Form, Input, Modal, Space, Switch, Table, Tag, Typography } from 'antd';
import { models } from '../../../wailsjs/go/models';
import { formatAppError } from '../../utils/appError';

type ExecutorValues = { name: string; institution: string; position: string; isActive: boolean };

export default function DeadlineExecutorsTab() {
  const { message, modal } = App.useApp();
  const [items, setItems] = useState<models.DeadlineExecutor[]>([]);
  const [loading, setLoading] = useState(false);
  const [editor, setEditor] = useState<models.DeadlineExecutor | null | undefined>(undefined);
  const [form] = Form.useForm<ExecutorValues>();
  const generation = useRef(0);
  const load = useCallback(async () => {
    const current = ++generation.current;
    setLoading(true);
    try {
      const service = await import('../../../wailsjs/go/services/DeadlineControlService');
      const result = await service.ListExecutors(true);
      if (generation.current === current) setItems(result);
    } catch (error) { if (generation.current === current) message.error(formatAppError(error)); }
    finally { if (generation.current === current) setLoading(false); }
  }, [message]);
  useEffect(() => { void load(); return () => { generation.current += 1; }; }, [load]);

  const edit = (item: models.DeadlineExecutor | null) => {
    setEditor(item);
    form.setFieldsValue({ name: item?.name || '', institution: item?.institution || '', position: item?.position || '', isActive: item?.isActive ?? true });
  };
  const closeEditor = () => {
    const close = () => { setEditor(undefined); form.resetFields(); };
    if (!form.isFieldsTouched()) { close(); return; }
    modal.confirm({ title: 'Закрыть без сохранения?', onOk: close });
  };
  const save = async (values: ExecutorValues) => {
    setLoading(true);
    try {
      const service = await import('../../../wailsjs/go/services/DeadlineControlService');
      await service.SaveExternalExecutor(editor?.id || '', models.ExternalDeadlineExecutorRequest.createFrom({ ...values, version: editor?.version || 0 }));
      setEditor(undefined); form.resetFields(); message.success('Исполнитель сохранён'); await load();
    } catch (error) { message.error(formatAppError(error)); await load(); }
    finally { setLoading(false); }
  };
  return <>
    <Space style={{ marginBottom: 16 }}>
      <Button type="primary" onClick={() => edit(null)}>Добавить внешнего исполнителя</Button>
      <Button onClick={() => void load()} loading={loading}>Обновить</Button>
    </Space>
    <Typography.Paragraph type="secondary">Пользователи доступны для назначения по отдельному признаку в карточке пользователя. Внешним исполнителям учётная запись не нужна. Неактивные записи сохраняются для истории.</Typography.Paragraph>
    <Table<models.DeadlineExecutor> rowKey="id" dataSource={items} loading={loading} pagination={{ pageSize: 20 }} columns={[
      { title: 'ФИО', dataIndex: 'name' },
      { title: 'Учреждение', dataIndex: 'institution' },
      { title: 'Должность', dataIndex: 'position' },
      { title: 'Тип', render: (_value, item) => item.userId ? 'Пользователь программы' : 'Внешний исполнитель' },
      { title: 'Назначение', render: (_value, item) => <Tag color={item.isActive ? 'green' : 'default'}>{item.isActive ? 'Разрешено' : 'Недоступен'}</Tag> },
      { title: 'Действия', render: (_value, item) => item.userId ? <Typography.Text type="secondary">Изменяется в карточке пользователя</Typography.Text> : <Button onClick={() => edit(item)}>Изменить</Button> },
    ]} />
    <Modal title={editor ? 'Изменить внешнего исполнителя' : 'Добавить внешнего исполнителя'} open={editor !== undefined} onCancel={closeEditor} onOk={() => form.submit()} confirmLoading={loading} destroyOnHidden>
      <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
        <Form.Item name="name" label="ФИО" rules={[{ required: true, whitespace: true }]}><Input maxLength={500} /></Form.Item>
        <Form.Item name="institution" label="Учреждение"><Input maxLength={500} /></Form.Item>
        <Form.Item name="position" label="Должность"><Input maxLength={500} /></Form.Item>
        <Form.Item name="isActive" label="Можно назначать исполнителем и соисполнителем" valuePropName="checked"><Switch /></Form.Item>
      </Form>
    </Modal>
  </>;
}
