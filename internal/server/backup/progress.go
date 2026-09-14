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
		actor := actors[job.ID]
		for i, stage := range job.Stages {
			details := fmt.Sprintf("Операция %s; задание %s; копия %s; этап %s", map[string]string{"": "Создание копии", "verify": "Проверка", "restore": "Восстановление", "delete": "Удаление"}[job.Kind], job.ID, job.CopyID, stage.State)
			if stage.Error != "" {
				details += "; " + stage.Error
			}
			_, err = s.DB.ExecContext(ctx, `INSERT INTO admin_audit_log (user_id, user_name, action, details, created_at, outbox_deduplication_key) SELECT u.id, COALESCE(u.full_name, CASE WHEN $4 = 'schedule' THEN 'Расписание' ELSE 'Пользователь ' || $4 END), 'BACKUP', $1, $2, $3 FROM (SELECT 1) seed LEFT JOIN users u ON u.id::text = $4 ON CONFLICT (outbox_deduplication_key) WHERE outbox_deduplication_key IS NOT NULL DO NOTHING`, details, stage.StartedAt, fmt.Sprintf("backup:%s:%d", job.ID, i), actor)
			if err != nil {
				return
			}
		}
	}
}
