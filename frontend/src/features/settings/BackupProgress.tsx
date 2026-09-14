import { Alert, Timeline } from 'antd';
import { models } from '../../../wailsjs/go/models';

const labels: Record<string, string> = {
  finalizing: 'Завершение восстановления', queued: 'Ожидание запуска', downloading: 'Скачивание', verifying: 'Проверка архива и базы данных',
  safety_snapshot: 'Создание и проверка страховочной копии', replacing: 'Замена данных',
  clearing_database: 'Подготовка базы данных', clearing_objects: 'Подготовка файлового хранилища',
  restoring: 'Восстановление данных и проверка вложений', migrating: 'Обновление схемы',
  rolling_back: 'Возврат исходного состояния', rolled_back: 'Исходное состояние восстановлено',
  rollback_failed: 'Не удалось восстановить исходное состояние', recovery_required: 'Требуется сброс окружения',
  completed: 'Завершено', failed: 'Ошибка', cancelled: 'Отменено', interrupted: 'Прервано', deleting: 'Удаление',
};

labels.snapshotting = 'Создание снимка';
labels.staged = 'Ожидает отправки';
labels.transferring = 'Передача на SMB';
labels.deleted = 'Удалено с SMB';

export default function BackupProgress({ stages, state, error }: { stages?: models.BackupStage[]; state: string; error?: string }) {
  return <>
    <Timeline items={(stages?.length ? stages : [{ state, startedAt: '', error }]).map((stage, index, all) => ({
      color: stage.error ? 'red' : index === all.length - 1 && state !== 'completed' ? 'blue' : 'green',
      content: <>{labels[stage.state] ?? (stage.state.startsWith('rollback_') ? 'Возврат исходного состояния: ' + stage.state : stage.state)}{stage.startedAt && ` · ${new Date(stage.startedAt).toLocaleTimeString()}`}{stage.error && <Alert type="error" title={stage.error} />}</>,
    }))} />
  </>;
}
