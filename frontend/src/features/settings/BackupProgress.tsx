import { Alert, Timeline } from 'antd';
import { models } from '../../../wailsjs/go/models';

import { backupStateLabel } from './backupLabels';

export default function BackupProgress({ stages, state, error }: { stages?: models.BackupStage[]; state: string; error?: string }) {
  return <>
    <Timeline items={(stages?.length ? stages : [{ state, startedAt: '', error }]).map((stage, index, all) => ({
      color: stage.error ? 'red' : index === all.length - 1 && state !== 'completed' ? 'blue' : 'green',
      content: <>{backupStateLabel(stage.state)}{stage.startedAt && ` · ${new Date(stage.startedAt).toLocaleTimeString()}`}{stage.error && <Alert type="error" title={stage.error} />}</>,
    }))} />
  </>;
}
