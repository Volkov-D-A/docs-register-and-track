import { useEffect, useState } from 'react';
import { Alert, Button, Checkbox, Modal, Space, Table, Typography } from 'antd';
import { models } from '../../../wailsjs/go/models';
import { formatAppError } from '../../utils/appError';

const terminal = new Set(['completed', 'failed', 'cancelled', 'interrupted', 'rolled_back', 'rollback_failed', 'recovery_required']);
const phases: Record<string, string> = {
  finalizing: 'Завершение восстановления', queued: 'Ожидание запуска', downloading: 'Скачивание', verifying: 'Проверка архива и базы данных',
  safety_snapshot: 'Создание и проверка страховочной копии', replacing: 'Замена данных',
  clearing_database: 'Подготовка базы данных', clearing_objects: 'Подготовка файлового хранилища',
  restoring: 'Восстановление данных и проверка вложений', migrating: 'Обновление схемы',
  rolling_back: 'Возврат исходного состояния', rolled_back: 'Исходное состояние восстановлено',
  rollback_failed: 'Не удалось восстановить исходное состояние', recovery_required: 'Требуется сброс окружения',
  completed: 'Завершено', failed: 'Ошибка', cancelled: 'Отменено', interrupted: 'Прервано', deleting: 'Удаление',
};

export default function BackupCatalog({ copies, loading, onChanged }: { copies: models.BackupCopy[]; loading: boolean; onChanged: () => Promise<void> }) {
  const [selection, setSelection] = useState<{ copy: models.BackupCopy; kind: 'verify' | 'restore' | 'delete' }>();
  const [operation, setOperation] = useState<Pick<models.BackupOperationStarted, 'job' | 'statusToken' | 'expiresAt'>>();
  const [visible, setVisible] = useState(false);
  const [verification, setVerification] = useState('');
  const [acknowledged, setAcknowledged] = useState(false);
  const [pollingExpired, setPollingExpired] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const running = !!operation && !terminal.has(operation.job.state);
  useEffect(() => {
    if (!operation || terminal.has(operation.job.state) || pollingExpired) return;
    let disposed = false;
    const timer = window.setTimeout(() => { void (async () => {
      try {
        if (Date.now() >= new Date(operation.expiresAt).getTime()) { setPollingExpired(true); setError('Срок наблюдения истёк. Задание продолжает выполняться на сервере; после завершения войдите заново.'); return; }
        const api = await import('../../../wailsjs/go/services/SettingsService');
        const job = await api.GetBackupOperation(operation.job.id, operation.statusToken);
        if (disposed) return;
        setOperation({ ...operation, job });
        setError('');
        if (job.kind === 'verify' && job.state === 'completed') setVerification(job.id);
        if (terminal.has(job.state) && job.kind !== 'restore') await onChanged();
      } catch (e) { if (!disposed) { setError(formatAppError(e)); setOperation({ ...operation }); } }
    })(); }, 1500);
    return () => { disposed = true; window.clearTimeout(timer); };
  }, [operation, onChanged, pollingExpired]);
  const start = async (kind: string) => {
    if (!selection) return;
    setBusy(true); setPollingExpired(false); setError('');
    try {
      const api = await import('../../../wailsjs/go/services/SettingsService');
      setOperation(await api.StartBackupOperation(kind, models.BackupOperationRequest.createFrom({
        copyId: selection.copy.id, verificationId: verification,
        confirmation: kind === 'delete' ? selection.copy.deleteConfirmation : selection.copy.restoreConfirmation,
      })));
    } catch (e) { setError(formatAppError(e)); }
    finally { setBusy(false); }
  };
  const select = (copy: models.BackupCopy, kind: 'verify' | 'restore' | 'delete') => {
    setVisible(true); setSelection({ copy, kind }); setOperation(undefined); setVerification(''); setAcknowledged(false); setError('');
  };
  return <>
    <Table rowKey="id" dataSource={copies} loading={loading} pagination={{ pageSize: 10 }} columns={[
      { title: 'ID копии', dataIndex: 'id' },
      { title: 'Формат', dataIndex: 'format', render: value => `v${value}` },
      { title: 'Дата копии', dataIndex: 'createdAt', render: value => new Date(value).toLocaleString() },
      { title: 'Размер архива', dataIndex: 'size', render: value => `${(value / 1024 / 1024).toFixed(1)} МБ` },
      { title: 'Проверка', dataIndex: 'verification', render: value => ({ incomplete: 'Неполный комплект', deleting: 'Удаление не завершено', verified: 'Проверен' }[String(value)] ?? 'Архив не проверен') },
      { title: 'Примечание', dataIndex: 'issue' },
      { title: 'Действия', render: (_, copy: models.BackupCopy) => <Space wrap>
        <Button disabled={running || ['incomplete', 'deleting'].includes(copy.verification)} onClick={() => select(copy, 'verify')}>Проверить</Button>
        <Button disabled={running || ['incomplete', 'deleting'].includes(copy.verification)} onClick={() => select(copy, 'restore')}>Восстановить</Button>
        <Button danger disabled={running || !copy.canDelete} onClick={() => select(copy, 'delete')}>Удалить</Button>
      </Space> },
    ]} />
    {operation && <Button onClick={() => setVisible(true)}>Состояние операции</Button>}
    <Modal open={!!selection && visible} title={selection?.kind === 'restore' ? 'Восстановление копии' : selection?.kind === 'delete' ? 'Удаление копии' : 'Проверка копии'} footer={null} onCancel={() => setVisible(false)}>
      {selection && <Space orientation="vertical" style={{ width: '100%' }}>
        <Typography.Text>ID: {selection.copy.id}</Typography.Text>
        <Typography.Text>{new Date(selection.copy.createdAt).toLocaleString()} · {(selection.copy.size / 1024 / 1024).toFixed(1)} МБ</Typography.Text>
        {selection.kind === 'restore' && <Alert type="warning" showIcon title="Текущие данные будут заменены" description="Документы, файлы, пользователи и права будут заменены данными копии. Потребуется пароль пользователя из архива. После восстановления нужно войти заново, сохранить пароль SMB и явно включить расписание. Поддерживаются только выделенные БД и файловое хранилище Docflow." />}
        {selection.kind === 'delete' && <Alert type="warning" title="Удаляются архив и manifest на SMB" description="Локальный архив и история сохранятся. Сервер проверит минимум исправных копий; последнюю проверенную копию удалить нельзя." />}
        {operation && <Alert type={operation.job.error ? 'error' : operation.job.state === 'completed' ? 'success' : 'info'} title={phases[operation.job.state] ?? (operation.job.state.startsWith('rollback_') ? 'Возврат исходного состояния' : operation.job.state)} description={operation.job.error || 'Закрытие окна не прерывает задание на сервере.'} />}
        {error && <Alert type="error" title={error} />}
        {operation?.job.canCancel && running && <Button onClick={() => { void (async () => { try { await (await import('../../../wailsjs/go/services/SettingsService')).CancelBackupOperation(operation.job.id); } catch (e) { setError(formatAppError(e)); } })(); }}>Отменить задание</Button>}
        {!running && selection.kind !== 'delete' && !verification && <Button loading={busy} onClick={() => void start('verify')}>Проверить выбранную копию</Button>}
        {!running && (selection.kind === 'delete' || (selection.kind === 'restore' && verification && operation?.job.kind === 'verify')) && <>
          <Checkbox checked={acknowledged} onChange={e => setAcknowledged(e.target.checked)}>Подтверждаю {selection.kind === 'delete' ? 'удаление выбранной копии' : 'замену текущих данных выбранной копией'}</Checkbox>
          <Button danger type="primary" disabled={!acknowledged} loading={busy} onClick={() => void start(selection.kind)}>{selection.kind === 'delete' ? 'Удалить подтверждённую копию' : 'Заменить данные'}</Button>
        </>}
        {operation?.job.kind === 'restore' && operation.job.state === 'completed' && <Button type="primary" onClick={() => { void import('../../../wailsjs/go/services/AuthService').then(api => api.Logout()); }}>Выйти и войти заново</Button>}
        {operation && ['rollback_failed', 'recovery_required'].includes(operation.job.state) && <Alert type="error" title="Обычная работа заблокирована" description="Оператору нужно сохранить журнал и архивы staging, сбросить целевую БД и файлы, инициализировать приложение и восстановить копию через админпанель." />}
      </Space>}
    </Modal>
  </>;
}
