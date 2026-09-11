import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, App, Button, Form, Input, InputNumber, Select, Space, Switch, Table, Typography } from 'antd';
import { onServerEvent } from '../../events/serverEvents';
import BackupCatalog from './BackupCatalog';
import { models } from '../../../wailsjs/go/models';
import { formatAppError } from '../../utils/appError';

const labels: Record<string, string> = {
  queued: 'В очереди', snapshotting: 'Создание снимка', staged: 'Ожидает отправки',
  transferring: 'Передача на SMB', verifying: 'Проверка', completed: 'Готово',
  deleting: 'Удаление', deleted: 'Удалено с SMB', downloading: 'Скачивание', safety_snapshot: 'Страховочная копия', replacing: 'Замена данных', finalizing: 'Завершение', restoring: 'Восстановление', migrating: 'Миграции', clearing_database: 'Подготовка БД', clearing_objects: 'Подготовка файлов', rolled_back: 'Исходное состояние возвращено', rollback_failed: 'Откат не выполнен', recovery_required: 'Требуется сброс окружения',
  failed: 'Ошибка', cancelled: 'Отменено', interrupted: 'Прервано перезапуском',
};

export default function BackupTab() {
  const { message } = App.useApp();
  const [form] = Form.useForm<models.BackupSettingsUpdate>();
  const [jobs, setJobs] = useState<models.BackupJob[]>([]);
  const [copies, setCopies] = useState<models.BackupCopy[]>([]);
  const [catalogIssue, setCatalogIssue] = useState('');
  const [catalogBusy, setCatalogBusy] = useState(false);
  const [issue, setIssue] = useState('');
  const [nextRun, setNextRun] = useState('');
  const [busy, setBusy] = useState(false);
  const completedJobs = useRef('');
  const [passwordSet, setPasswordSet] = useState(false);
  const [connectionConfigured, setConnectionConfigured] = useState(false);
  const savedConnectionConfigured = useRef(false);
  const updateConnection = useCallback((settings: models.BackupSettings) => {
    const configured = !!(settings.smb.host.trim() && settings.smb.share.trim() && settings.smb.user.trim() && settings.passwordSet);
    savedConnectionConfigured.current = configured;
    setConnectionConfigured(configured);
    setPasswordSet(settings.passwordSet);
  }, []);
  const reloadJobs = useCallback(async () => {
    const api = await import('../../../wailsjs/go/services/SettingsService');
    setJobs(await api.ListBackups() ?? []);
  }, []);
  const reloadCatalog = useCallback(async () => {
    if (!savedConnectionConfigured.current) {
      setCopies([]);
      setCatalogIssue('');
      return;
    }
    setCatalogBusy(true);
    try {
      const api = await import('../../../wailsjs/go/services/SettingsService');
      setCopies(await api.ListBackupCopies() ?? []);
      setCatalogIssue('');
    } catch (error) { setCopies([]); setCatalogIssue(formatAppError(error)); }
    finally { setCatalogBusy(false); }
  }, []);
  useEffect(() => {
    let disposed = false;
    void (async () => {
      try {
        const api = await import('../../../wailsjs/go/services/SettingsService');
        const response = await api.GetBackupSettings();
        if (disposed) return;
        form.setFieldsValue({ settings: response.settings, password: '', clearPassword: false });
        setNextRun(response.nextRun); setIssue(response.issue); updateConnection(response.settings);
        await reloadJobs();
        await reloadCatalog();
      } catch (error) { if (!disposed) message.error(formatAppError(error)); }
    })();
    return () => { disposed = true; };
  }, [form, message, reloadJobs, reloadCatalog, updateConnection]);
  useEffect(() => onServerEvent((event) => {
    if (event.topic === 'backups' || event.topic === 'resync') {
      void reloadJobs().catch(() => undefined);
      if (event.topic === 'resync') void reloadCatalog();
    }
  }), [reloadJobs, reloadCatalog]);
  useEffect(() => {
    const finished = jobs.filter(job => !job.kind && ['completed', 'deleted'].includes(job.state)).map(job => job.id + job.state).join(',');
    if (completedJobs.current && completedJobs.current !== finished) void reloadCatalog();
    completedJobs.current = finished || 'none';
  }, [jobs, reloadCatalog]);
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
      updateConnection(response.settings); setNextRun(response.nextRun); setIssue(response.issue);
      form.setFieldsValue({ password: '', clearPassword: false });
      await reloadCatalog();
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
        <Button disabled={busy || !!issue || !connectionConfigured} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).CheckBackupConnection(), 'Подключение и файловые операции проверены')}>Проверить сохранённое подключение</Button>
        <Button disabled={busy || !!issue || !connectionConfigured} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).StartBackup(), 'Задание создано')}>Создать копию</Button>
      </Space>
    </Form>
    {nextRun && <Typography.Text>Следующий запуск: {new Date(nextRun).toLocaleString()}</Typography.Text>}
    <Typography.Title level={4}>Каталог копий на SMB</Typography.Title>
    <Typography.Text>Каталог доступен независимо от локальной истории. Наличие manifest и совпадение размера не означают полную проверку архива.</Typography.Text>
    <Button disabled={!connectionConfigured} loading={catalogBusy} onClick={() => void reloadCatalog()}>Обновить каталог</Button>
    {!connectionConfigured && <Alert type="info" showIcon title="Сначала заполните и сохраните настройки SMB-подключения и пароль." />}
    {catalogIssue && <Alert type="error" showIcon title="Каталог SMB недоступен" description={catalogIssue} />}
    <BackupCatalog copies={copies} loading={catalogBusy} onChanged={reloadCatalog} />
    <Typography.Title level={4}>История заданий</Typography.Title>
    <Table rowKey="id" dataSource={jobs} pagination={{ pageSize: 10 }} columns={[
      { title: 'Операция', dataIndex: 'kind', render: value => ({ verify: 'Проверка', restore: 'Восстановление', delete: 'Удаление' }[String(value)] ?? 'Создание копии') },
      { title: 'ID копии', render: (_, job) => job.copyId || job.id },
      { title: 'Создано', dataIndex: 'createdAt', render: value => new Date(value).toLocaleString() },
      { title: 'Состояние', dataIndex: 'state', render: value => labels[value] ?? value },
      { title: 'Размер', dataIndex: 'archiveSize', render: value => value ? `${(value / 1024 / 1024).toFixed(1)} МБ` : '—' },
      { title: 'Ошибка', dataIndex: 'error' },
      { title: 'Действия', render: (_, job) => job.kind ? null : ['snapshotting', 'transferring', 'verifying'].includes(job.state) ? <Button onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).CancelBackup(job.id), 'Отмена запрошена')}>Отменить</Button> : ['staged', 'cancelled'].includes(job.state) && job.archiveSize > 0 ? <Button onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).RetryBackup(job.id), 'Повторная отправка запланирована')}>Повторить отправку</Button> : null },
    ]} />

  </Space>;
}
