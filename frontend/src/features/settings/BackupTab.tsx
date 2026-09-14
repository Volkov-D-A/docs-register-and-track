import { useCallback, useEffect, useRef, useState } from 'react';
import { ReloadOutlined } from '@ant-design/icons';
import { Alert, App, Button, Collapse, Form, Input, InputNumber, Modal, Select, Space, Switch, Typography } from 'antd';
import { onServerEvent } from '../../events/serverEvents';
import BackupProgress from './BackupProgress';
import BackupCatalog from './BackupCatalog';
import { models } from '../../../wailsjs/go/models';
import { formatAppError, normalizeAppError } from '../../utils/appError';

export default function BackupTab() {
  const { message } = App.useApp();
  const [form] = Form.useForm<models.BackupSettingsUpdate>();
  const [panels, setPanels] = useState<string[]>(['connection', 'schedule']);
  const [jobs, setJobs] = useState<models.BackupJob[]>([]);
  const [creationVisible, setCreationVisible] = useState(false);
  const [catalogActionsContainer, setCatalogActionsContainer] = useState<HTMLSpanElement | null>(null);
  const [creationJobID, setCreationJobID] = useState<string>();
  const creationJob = jobs.find(job => !job.kind && job.id === creationJobID);
  const creationRunning = !!creationJob && ['queued', 'snapshotting', 'transferring', 'verifying'].includes(creationJob.state);
  const [copies, setCopies] = useState<models.BackupCopy[]>([]);
  const [catalogIssue, setCatalogIssue] = useState('');
  const [catalogDeferred, setCatalogDeferred] = useState(false);
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
    setPanels([...(configured ? [] : ['connection']), ...(settings.time && settings.timezone && settings.weekdays?.length ? [] : ['schedule'])]);
  }, []);
  const reloadJobs = useCallback(async () => {
    const api = await import('../../../wailsjs/go/services/SettingsService');
    setJobs(await api.ListBackups() ?? []);
  }, []);
  const reloadCatalog = useCallback(async () => {
    if (!savedConnectionConfigured.current) {
      setCopies([]);
      setCatalogIssue('');
      setCatalogDeferred(false);
      return;
    }
    setCatalogBusy(true);
    try {
      const api = await import('../../../wailsjs/go/services/SettingsService');
      setCopies(await api.ListBackupCopies() ?? []);
      setCatalogIssue('');
      setCatalogDeferred(false);
    } catch (error) {
      if (normalizeAppError(error).code === 'MAINTENANCE') {
        setCatalogIssue('');
        setCatalogDeferred(true);
      } else {
        setCatalogDeferred(false);
        setCopies([]);
        setCatalogIssue(formatAppError(error));
      }
    }
    finally { setCatalogBusy(false); }
  }, []);
  useEffect(() => {
    if (!catalogDeferred) return;
    let disposed = false;
    let timer: number;
    const retry = async () => {
      await reloadCatalog();
      if (!disposed) timer = window.setTimeout(() => { void retry(); }, 3000);
    };
    timer = window.setTimeout(() => { void retry(); }, 3000);
    return () => { disposed = true; window.clearTimeout(timer); };
  }, [catalogDeferred, reloadCatalog]);
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
    const finished = jobs.filter(job => !job.kind && ['completed', 'deleted'].includes(job.state)).map(job => job.id + job.state).join(',') || 'none';
    if (completedJobs.current && completedJobs.current !== finished) void reloadCatalog();
    completedJobs.current = finished;
  }, [jobs, reloadCatalog]);
  const action = async (work: () => Promise<unknown>, success: string) => {
    setBusy(true);
    try { await work(); message.success(success); await reloadJobs(); }
    catch (error) { message.error(formatAppError(error)); }
    finally { setBusy(false); }
  };
  return <Space orientation="vertical" style={{ width: '100%' }} size="large">
    {issue && <Alert type="warning" showIcon title={issue} />}
    <Form form={form} layout="vertical" onFinish={values => void action(async () => {
      const api = await import('../../../wailsjs/go/services/SettingsService');
      await api.SaveBackupSettings(models.BackupSettingsUpdate.createFrom({ ...form.getFieldsValue(true), ...values }));
      const response = await api.GetBackupSettings();
      updateConnection(response.settings); setNextRun(response.nextRun); setIssue(response.issue);
      form.setFieldsValue({ password: '', clearPassword: false });
      await reloadCatalog();
    }, 'Настройки сохранены')}>
      <Collapse activeKey={panels} onChange={keys => setPanels(Array.isArray(keys) ? keys : [keys])} items={[{ key: 'connection', label: 'Настройки подключения', forceRender: true, children: <>
      <Space align="start" wrap>
        <Form.Item name={['settings', 'smb', 'host']} label="Сервер SMB" rules={[{ required: true }]}><Input placeholder="nas.example.local" /></Form.Item>
        <Form.Item name={['settings', 'smb', 'share']} label="Общая папка" rules={[{ required: true }]}><Input placeholder="backups" /></Form.Item>
        <Form.Item name={['settings', 'smb', 'directory']} label="Подкаталог"><Input placeholder="docflow" /></Form.Item>
        <Form.Item name={['settings', 'smb', 'user']} label="Пользователь" rules={[{ required: true }]}><Input autoComplete="off" /></Form.Item>
        <Form.Item name="password" label={passwordSet ? 'Новый пароль (текущий сохранён)' : 'Пароль'}><Input.Password autoComplete="new-password" /></Form.Item>
        <Form.Item name="clearPassword" label="Удалить сохранённый пароль" valuePropName="checked"><Switch /></Form.Item>
      </Space>
      </> }, { key: 'schedule', label: 'Настройки расписания', forceRender: true, children: <Space align="start" wrap>
        <Form.Item name={['settings', 'enabled']} label="По расписанию" valuePropName="checked"><Switch /></Form.Item>
        <Form.Item name={['settings', 'time']} label="Время" rules={[{ required: true }]}><Input type="time" /></Form.Item>
        <Form.Item name={['settings', 'timezone']} label="Часовой пояс" rules={[{ required: true }]}><Input placeholder="Asia/Yekaterinburg" /></Form.Item>
        <Form.Item name={['settings', 'weekdays']} label="Дни недели" rules={[{ required: true }]}><Select mode="multiple" style={{ minWidth: 270 }} options={['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'].map((label, value) => ({ label, value }))} /></Form.Item>
        <Form.Item name={['settings', 'retentionDays']} label="Хранить, дней"><InputNumber min={1} max={3650} /></Form.Item>
        <Form.Item name={['settings', 'keepCopies']} label="Сохранять минимум копий"><InputNumber min={1} max={1000} /></Form.Item>
      </Space> }]} />
      <Space wrap style={{ marginTop: 16 }}>
        <Button type="primary" htmlType="submit" loading={busy}>Сохранить</Button>
        <Button disabled={busy || !!issue || !connectionConfigured} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).CheckBackupConnection(), 'Подключение и файловые операции проверены')}>Проверить сохранённое подключение</Button>
        <Button disabled={busy || !!issue || !connectionConfigured} onClick={() => {
          const latest = jobs.find(job => !job.kind);
          const resumable = latest && (['queued', 'snapshotting', 'transferring', 'verifying', 'staged'].includes(latest.state) || (latest.state === 'cancelled' && latest.archiveSize > 0));
          setCreationJobID(resumable ? latest.id : undefined);
          setCreationVisible(true);
        }}>Создать копию</Button>
        <span ref={setCatalogActionsContainer} />
      </Space>
    </Form>
    {nextRun && <Typography.Text>Следующий запуск: {new Date(nextRun).toLocaleString()}</Typography.Text>}
    <Space align="center">
      <Typography.Title level={4} style={{ margin: 0 }}>Каталог копий на SMB</Typography.Title>
      <Button type="text" icon={<ReloadOutlined />} aria-label="Обновить каталог" title="Обновить каталог" disabled={!connectionConfigured} loading={catalogBusy} onClick={() => void reloadCatalog()} />
    </Space>
    {!connectionConfigured && <Alert type="info" showIcon title="Сначала заполните и сохраните настройки SMB-подключения и пароль." />}
    {catalogDeferred && <Alert type="info" showIcon title="Обновление каталога отложено до завершения обслуживания сервера." />}
    {catalogIssue && <Alert type="error" showIcon title="Каталог SMB недоступен" description={catalogIssue} />}
    <BackupCatalog copies={copies} loading={catalogBusy} onChanged={reloadCatalog} actionsContainer={catalogActionsContainer} />
    <Modal destroyOnHidden open={creationVisible} title="Создание резервной копии" footer={null} onCancel={() => setCreationVisible(false)}>
      <Space orientation="vertical" style={{ width: '100%' }}>
        {creationJob && <>
          <Typography.Text>ID: {creationJob.id}</Typography.Text>
          <BackupProgress stages={creationJob.stages} state={creationJob.state} error={creationJob.error} />
          {['snapshotting', 'transferring', 'verifying'].includes(creationJob.state) && <Button disabled={busy} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).CancelBackup(creationJob.id), 'Отмена запрошена')}>Отменить</Button>}
          {['staged', 'cancelled'].includes(creationJob.state) && creationJob.archiveSize > 0 && <Button disabled={busy} onClick={() => void action(async () => (await import('../../../wailsjs/go/services/SettingsService')).RetryBackup(creationJob.id), 'Повторная отправка запланирована')}>Повторить отправку</Button>}
        </>}
        <Button type="primary" loading={busy} disabled={creationRunning || !!issue || !connectionConfigured} onClick={() => void action(async () => {
          const job = await (await import('../../../wailsjs/go/services/SettingsService')).StartBackup();
          setCreationJobID(job.id);
          setJobs(current => [job, ...current.filter(existing => existing.id !== job.id)]);
        }, 'Задание создано')}>Начать создание копии</Button>
      </Space>
    </Modal>
  </Space>;
}
