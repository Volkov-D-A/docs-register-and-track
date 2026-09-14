package backup

import (
	"context"
	"fmt"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func appendStage(stages []models.BackupStage, state, issue string, at time.Time) []models.BackupStage {
	if len(stages) > 0 && stages[len(stages)-1].State == state && stages[len(stages)-1].Error == issue {
		return stages
	}
	return append(stages, models.BackupStage{State: state, Error: issue, StartedAt: at})
}

// Replay the durable local journal only outside the snapshot/replacement barrier.
// Deduplication also allows replay after restoring an older database.
func (s *Service) flushAudit(ctx context.Context) {
	if s.DB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	jobs, err := s.Jobs()
	if err != nil {
		return
	}
	actors := make(map[string]string)
	entries, err := s.jobs()
	if err != nil {
		return
	}
	for _, entry := range entries {
		actors[entry.ID] = entry.Actor
	}
	ops, err := s.operations()
	if err != nil {
		return
	}
	for _, op := range ops {
		actors[op.ID] = op.Actor
	}
	for _, job := range jobs {
		operation := map[string]string{"": "Создание копии", "restore": "Восстановление", "delete": "Удаление"}[job.Kind]
		if operation == "" || !operationFinished(job.State) {
			continue
		}
		actor := actors[job.ID]
		result := map[string]string{
			"completed": "Успешно", "failed": "Ошибка", "cancelled": "Отменено",
			"interrupted": "Прервано перезапуском", "rolled_back": "Исходное состояние восстановлено",
			"rollback_failed": "Не удалось вернуть исходное состояние", "recovery_required": "Требуется сброс окружения",
		}[job.State]
		details := fmt.Sprintf("Операция %s; задание %s; копия %s; результат: %s", operation, job.ID, job.CopyID, result)
		if job.Error != "" {
			details += "; " + job.Error
		}
		// Keep the terminal stage key compatible with previously recorded results.
		key := fmt.Sprintf("backup:%s:%d", job.ID, len(job.Stages)-1)
		_, err = s.DB.ExecContext(ctx, `INSERT INTO admin_audit_log (user_id, user_name, action, details, created_at, outbox_deduplication_key) SELECT u.id, COALESCE(u.full_name, CASE WHEN $4 = 'schedule' THEN 'Расписание' ELSE 'Пользователь ' || $4 END), 'BACKUP', $1, $2, $3 FROM (SELECT 1) seed LEFT JOIN users u ON u.id::text = $4 ON CONFLICT (outbox_deduplication_key) WHERE outbox_deduplication_key IS NOT NULL DO NOTHING`, details, job.UpdatedAt, key, actor)
		if err != nil {
			return
		}
	}
}
