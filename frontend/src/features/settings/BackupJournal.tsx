import { useEffect, useState } from 'react';
import { HistoryOutlined } from '@ant-design/icons';
import { Alert, Button, Modal, Space, Table, Typography } from 'antd';
import { models } from '../../../wailsjs/go/models';
import { onServerEvent } from '../../events/serverEvents';
import { formatAppError } from '../../utils/appError';
import BackupProgress from './BackupProgress';
import { backupKindLabel, backupStateLabel } from './backupLabels';

export default function BackupJournal({ jobs, onRefresh }: { jobs: models.BackupJob[]; onRefresh: () => Promise<void> }) {
  const [visible, setVisible] = useState(false);
  const [selectedID, setSelectedID] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [liveOperations, setLiveOperations] = useState<Record<string, models.BackupOperation>>({});
  useEffect(() => onServerEvent(event => {
    if (event.topic !== 'operation' || !event.operation) return;
    const job = event.operation;
    setLiveOperations(current => {
      if (current[job.id]?.updatedAt && new Date(current[job.id].updatedAt) > new Date(job.updatedAt)) return current;
      return { ...current, [job.id]: job };
    });
  }), []);
  const history = jobs.map(job => {
    const live = liveOperations[job.id];
    return live && new Date(live.updatedAt) >= new Date(job.updatedAt) ? models.BackupJob.createFrom({ ...job, ...live }) : job;
  });
  for (const live of Object.values(liveOperations)) {
    if (!history.some(job => job.id === live.id)) history.push(models.BackupJob.createFrom(live));
  }
  history.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  const selected = history.find(job => job.id === selectedID);
  const open = async () => {
    setVisible(true); setLoading(true); setError('');
    try { await onRefresh(); }
    catch (e) { setError(formatAppError(e)); }
    finally { setLoading(false); }
  };
  return <>
    <Button type="text" icon={<HistoryOutlined />} aria-label="Журнал операций резервирования" title="Журнал операций резервирования" onClick={() => void open()} />
    <Modal destroyOnHidden open={visible} title="Журнал операций резервирования" width={960} footer={null} onCancel={() => setVisible(false)}>
      {error && <Alert type="error" showIcon title="Не удалось обновить журнал" description={error} />}
      <Table size="small" rowKey="id" dataSource={history} loading={loading} pagination={{ pageSize: 10 }} scroll={{ x: 700 }} locale={{ emptyText: 'Операций резервирования пока нет' }} columns={[
        { title: 'Дата', dataIndex: 'createdAt', render: value => new Date(value).toLocaleString() },
        { title: 'Операция', render: (_, job: models.BackupJob) => <Button type="link" size="small" onClick={() => setSelectedID(job.id)}>{backupKindLabel(job.kind)}</Button> },
        { title: 'Копия', render: (_, job: models.BackupJob) => job.copyId || job.id },
        { title: 'Состояние', dataIndex: 'state', render: backupStateLabel },
      ]} />
    </Modal>
    <Modal destroyOnHidden open={!!selected} title={selected ? backupKindLabel(selected.kind) : 'Этапы операции'} footer={null} onCancel={() => setSelectedID(undefined)}>
      {selected && <Space orientation="vertical" style={{ width: '100%' }}>
        <Typography.Text>ID операции: {selected.id}</Typography.Text>
        <Typography.Text>Копия: {selected.copyId || selected.id}</Typography.Text>
        <Typography.Text>{new Date(selected.createdAt).toLocaleString()}</Typography.Text>
        <BackupProgress stages={selected.stages} state={selected.state} error={selected.error} />
      </Space>}
    </Modal>
  </>;
}
