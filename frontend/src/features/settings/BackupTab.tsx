import { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Form, Input, InputNumber, Select, Space, Switch, Table, Typography } from 'antd';
import { models } from '../../../wailsjs/go/models';
import { formatAppError } from '../../utils/appError';

const labels: Record<string, string> = {
  queued: 'В очереди', snapshotting: 'Создание снимка', staged: 'Ожидает отправки',
  transferring: 'Передача на SMB', verifying: 'Проверка', completed: 'Готово',
  failed: 'Ошибка', cancelled: 'Отменено', interrupted: 'Прервано перезапуском',
};

export default function BackupTab() {
  const { message } = App.useApp();
  const [form] = Form.useForm<models.BackupSettingsUpdate>();
  const [jobs, setJobs] = useState<models.BackupJob[]>([]);
  const [issue, setIssue] = useState('');
  const [nextRun, setNextRun] = useState('');
  const [busy, setBusy] = useState(false);
  const [passwordSet, setPasswordSet] = useState(false);
  const reloadJobs = useCallback(async () => {
    const api = await import('../../../wailsjs/go/services/SettingsService');
    setJobs(await api.ListBackups() ?? []);
  }, []);
  useEffect(() => {
    let disposed = false;
    void (async () => {
      try {
        const api = await import('../../../wailsjs/go/services/SettingsService');
        const response = await api.GetBackupSettings();
        if (disposed) return;
        form.setFieldsValue({ settings: response.settings, password: '', clearPassword: false });
        setNextRun(response.nextRun); setIssue(response.issue); setPasswordSet(response.settings.passwordSet);
        await reloadJobs();
      } catch (error) { if (!disposed) message.error(formatAppError(error)); }
    })();
    const timer = window.setInterval(() => { void reloadJobs().catch(() => undefined); }, 5000);
    return () => { disposed = true; window.clearInterval(timer); };
  }, [form, message, reloadJobs]);
  const action = async (work: () => Promise<unknown>, success: string) => {
    setBusy(true);
    try { await work(); message.success(success); await reloadJobs(); }
    catch (error) { message.error(formatAppError(error)); }
    finally { setBusy(false); }
  };
  return <Space orientation="vertical" style={{ width: '100%' }} size="large">
    {issue && <Alert type="warning" showIcon title={issue} />}
    <Typography.Paragraph>Резервная копия содержит базу данных и вложения. На время создания локального снимка работа с документами приостанавливается. Передача на SMB выполняется после возобновления работы.</Typography.Paragraph>
    <Form form={form} layout="vertical" onFinish={values => void action(async () => {
      const api = await import('../../../wailsjs/go/services/SettingsService');
      await api.SaveBackupSettings(models.BackupSettingsUpdate.createFrom(values));
      const response = await api.GetBackupSettings();
      setPasswordSet(response.settings.passwordSet); setNextRun(response.nextRun);
      form.setFieldsValue({ password: '', clearPassword: false });
    }, 'Настройки сохранены')}>
      <Space align="start" wrap>
        <Form.Item name={['settings', 'smb', 'host']} label="Сервер SMB" rules={[{ required: true }]}><Input placeholder="nas.example.local" /></Form.Item>
        <Form.Item name={['settings', 'smb', 'share']} label="Общая папка" rules={[{ required: true }]}><Input placeholder="backups" /></Form.Item>
        <Form.Item name={['settings', 'smb', 'directory']} label="Подкаталог"><Input placeholder="docflow" /></Form.Item>
      </Space>
      <Space align="start" wrap>
        <Form.Item name={['settings', 'smb', 'user']} label="Пользователь" rules={[{ required: true }]}><Input autoComplete="off" /></Form.Item>
        <Form.Item name={['settings', 'smb', 'domain']} label="Домен (если нужен)"><Input /></Form.Item>
        <Form.Item name="password" label={passwordSet ? 'Новый пароль (текущий сохранён)' : 'Пароль'}><Input.Password autoComplete="new-password" /></Form.Item>
        <Form.Item name="clearPassword" label="Удалить сохранённый пароль" valuePropName="checked"><Switch /></Form.Item>
      </Space>
      <Space align="start" wrap>
        <Form.Item name={['settings', 'enabled']} label="По расписанию" valuePropName="checked"><Switch /></Form.Item>
        <Form.Item name={['settings', 'time']} label="Время" rules={[{ required: true }]}><Input type="time" /></Form.Item>
        <Form.Item name={['settings', 'timezone']} label="Часовой пояс" rules={[{ required: true }]}><Input placeholder="Asia/Yekaterinburg" /></Form.Item>
        <Form.Item name={['settings', 'weekdays']} label="Дни недели" rules={[{ required: true }]}><Select mode="multiple" style={{ minWidth: 270 }} options={['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'].map((label, value) => ({ label, value }))} /></Form.Item>
        <Form.Item name={['settings', 'retentionDays']} label="Хранить, дней"><InputNumber min={1} max={3650} /></Form.Item>
        <Form.Item name={['settings', 'keepCopies']} label="Сохранять минимум копий"><InputNumber min={1} max={1000} /></Form.Item>
      </Space>
      <Space wrap>
        <Button type="primary" htmlType="submit" loading={busy}>Сохранить</Button>
        <Button disabled={busy || !!issue} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).CheckBackupConnection(), 'Подключение и файловые операции проверены')}>Проверить сохранённое подключение</Button>
        <Button disabled={busy || !!issue} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).StartBackup(), 'Задание создано')}>Создать копию</Button>
      </Space>
    </Form>
    {nextRun && <Typography.Text>Следующий запуск: {new Date(nextRun).toLocaleString()}</Typography.Text>}
    <Table rowKey="id" dataSource={jobs} pagination={{ pageSize: 10 }} columns={[
      { title: 'Создано', dataIndex: 'createdAt', render: value => new Date(value).toLocaleString() },
      { title: 'Состояние', dataIndex: 'state', render: value => labels[value] ?? value },
      { title: 'Размер', dataIndex: 'archiveSize', render: value => value ? `${(value / 1024 / 1024).toFixed(1)} МБ` : '—' },
      { title: 'Ошибка', dataIndex: 'error' },
      { title: 'Действия', render: (_, job) => ['snapshotting', 'transferring', 'verifying'].includes(job.state) ? <Button onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).CancelBackup(job.id), 'Отмена запрошена')}>Отменить</Button> : ['staged', 'cancelled'].includes(job.state) && job.archiveSize > 0 ? <Button onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).RetryBackup(job.id), 'Повторная отправка запланирована')}>Повторить отправку</Button> : null },
    ]} />
    <Alert type="info" showIcon title="Аварийное восстановление" description="Восстановление выполняется в отдельном режиме сервера при остановленном приложении, в пустую базу и пустое файловое хранилище. Панель восстановления доступна даже при потере основной базы; для входа нужен аварийный ключ." />
  </Space>;
}
