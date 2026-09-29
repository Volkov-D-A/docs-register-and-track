package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"time"
)

func (r *AssignmentRepository) CreateRecipientTaskWithOutbox(a *models.Assignment, effects []models.OutboxEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Создание ознакомления
	query := `
		INSERT INTO assignments (id, document_id, creator_id, content, created_at, updated_at, type, deadline)
		VALUES ($1, $2, $3, '', $4, $4, 'acknowledgment', $5)
	`
	_, err = tx.Exec(query, a.ID, a.DocumentID, a.CreatorID, a.CreatedAt, a.Deadline)
	if err != nil {
		return fmt.Errorf("failed to create acknowledgment: %w", err)
	}

	// 2. Создание пользователей ознакомления
	userQuery := `
		INSERT INTO assignment_recipients (id, assignment_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	for _, u := range a.Users {
		_, err = tx.Exec(userQuery, u.ID, a.ID, u.UserID, u.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to create acknowledgment user: %w", err)
		}
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *AssignmentRepository) GetRecipientsByAssignmentIDs(ackIDs []uuid.UUID) (map[uuid.UUID][]models.AssignmentRecipient, error) {
	result := make(map[uuid.UUID][]models.AssignmentRecipient, len(ackIDs))
	if len(ackIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT 
			au.assignment_id, au.user_id, au.confirmed_at,
			u.full_name as user_name, au.confirmed_by
		FROM assignment_recipients au
		JOIN users u ON au.user_id = u.id
		WHERE au.assignment_id = ANY($1)
		ORDER BY au.assignment_id, au.created_at, au.id
	`
	rows, err := r.db.Query(query, pq.Array(ackIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var au models.AssignmentRecipient
		var confirmedBy uuid.NullUUID
		err := rows.Scan(
			&au.AssignmentID, &au.UserID, &au.ConfirmedAt,
			&au.UserName, &confirmedBy,
		)
		if err != nil {
			return nil, err
		}

		if confirmedBy.Valid {
			au.ConfirmedBy = &confirmedBy.UUID
		}
		result[au.AssignmentID] = append(result[au.AssignmentID], au)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AssignmentRepository) ConfirmRecipientWithEffects(ackID, userID uuid.UUID, effects models.AssignmentConfirmationEffects) error {
	if r.outbox == nil {
		return ErrOutboxNotConfigured
	}
	return r.confirmRecipient(ackID, userID, effects.ActorID, effects.UserEvents)
}

func (r *AssignmentRepository) confirmRecipient(ackID, userID, actorID uuid.UUID, userEvents []models.CreateUserEventRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Serialize recipient confirmations on the parent; the last concurrent
	// confirmation must observe every preceding confirmation before completing it.
	var documentID uuid.UUID
	if err := tx.QueryRow(`SELECT document_id FROM assignments WHERE id = $1 AND type = 'acknowledgment' FOR UPDATE`, ackID).Scan(&documentID); err != nil {
		if err == sql.ErrNoRows {
			return models.ErrForbidden
		}
		return err
	}
	if actorID == uuid.Nil {
		actorID = userID
	}
	now := time.Now()

	// 1. Обновление статуса пользователя
	query := `
		UPDATE assignment_recipients
		SET confirmed_at = $1, confirmed_by = $4
		WHERE assignment_id = $2 AND user_id = $3 AND confirmed_at IS NULL
	`
	res, err := tx.Exec(query, now, ackID, userID, actorID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var exists bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM assignment_recipients WHERE assignment_id = $1 AND user_id = $2)`, ackID, userID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return models.ErrAlreadyConfirmed
		}
		return models.ErrForbidden
	}

	// 2. Проверка, все ли пользователи подтвердили
	checkQuery := `
		SELECT COUNT(*) 
		FROM assignment_recipients
		WHERE assignment_id = $1 AND confirmed_at IS NULL
	`
	var remaining int
	err = tx.QueryRow(checkQuery, ackID).Scan(&remaining)
	if err != nil {
		return err
	}

	// 3. Если все подтвердили, обновляем completed_at основного ознакомления
	if remaining == 0 {
		updateQuery := `
			UPDATE assignments
			SET completed_at = $1, updated_at = $1, status = 'finished'
			WHERE id = $2 AND completed_at IS NULL
		`
		_, err = tx.Exec(updateQuery, now, ackID)
		if err != nil {
			return err
		}
	}
	payload, err := json.Marshal(models.CreateJournalEntryRequest{DocumentID: documentID, UserID: actorID, Action: "ASSIGNMENT_ACKNOWLEDGED", Details: "Ознакомление подтверждено"})
	if err != nil {
		return err
	}
	if err := r.outbox.EnqueueTx(tx, models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "assignment:" + ackID.String() + ":confirmed:" + userID.String() + ":journal", Payload: string(payload)}); err != nil {
		return err
	}
	for _, request := range userEvents {
		payload, err := json.Marshal(struct {
			Request models.CreateUserEventRequest `json:"request"`
		}{Request: request})
		if err != nil {
			return err
		}
		key := "assignment:" + ackID.String() + ":confirmed:" + userID.String() + ":event:" + request.RecipientUserID.String()
		if err := r.outbox.EnqueueTx(tx, models.OutboxEvent{EventType: models.OutboxEventUserEvent, DeduplicationKey: key, Payload: string(payload)}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *AssignmentRepository) UpdateRecipientTaskDeadline(id uuid.UUID, deadline *time.Time, effects []models.OutboxEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE assignments SET deadline=$2, updated_at=NOW() WHERE id=$1 AND type='acknowledgment' AND status='new'`, id, deadline)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return models.NewConflict("ознакомление уже завершено; обновите данные")
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return err
	}
	return tx.Commit()
}
