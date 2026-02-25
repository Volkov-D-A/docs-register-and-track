package repository

import (
	"docflow/internal/database"
	"docflow/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AcknowledgmentRepository struct {
	db *database.DB
}

func NewAcknowledgmentRepository(db *database.DB) *AcknowledgmentRepository {
	return &AcknowledgmentRepository{db: db}
}

func (r *AcknowledgmentRepository) Create(a *models.Acknowledgment) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Создание ознакомления
	query := `
		INSERT INTO acknowledgments (id, document_id, document_type, creator_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(query, a.ID, a.DocumentID, a.DocumentType, a.CreatorID, a.Content, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create acknowledgment: %w", err)
	}

	// 2. Создание пользователей ознакомления
	userQuery := `
		INSERT INTO acknowledgment_users (id, acknowledgment_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	for _, u := range a.Users {
		_, err = tx.Exec(userQuery, u.ID, a.ID, u.UserID, u.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to create acknowledgment user: %w", err)
		}
	}

	return tx.Commit()
}

func (r *AcknowledgmentRepository) GetByDocumentID(documentID uuid.UUID) ([]models.Acknowledgment, error) {
	query := `
		SELECT 
			a.id, a.document_id, a.document_type, a.creator_id, a.content, a.created_at, a.completed_at,
			u.full_name as creator_name,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM acknowledgments a
		JOIN users u ON a.creator_id = u.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE a.document_id = $1
		ORDER BY a.created_at DESC
	`
	rows, err := r.db.Query(query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Acknowledgment
	for rows.Next() {
		var a models.Acknowledgment
		var docNumber string
		err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentType, &a.CreatorID, &a.Content, &a.CreatedAt, &a.CompletedAt,
			&a.CreatorName, &docNumber,
		)
		if err != nil {
			return nil, err
		}
		a.DocumentNumber = docNumber

		// Загрузка пользователей
		users, err := r.GetUsersByAcknowledgmentID(a.ID)
		if err != nil {
			return nil, err
		}
		a.Users = users

		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AcknowledgmentRepository) GetUsersByAcknowledgmentID(ackID uuid.UUID) ([]models.AcknowledgmentUser, error) {
	query := `
		SELECT 
			au.id, au.acknowledgment_id, au.user_id, au.viewed_at, au.confirmed_at, au.created_at,
			u.full_name as user_name
		FROM acknowledgment_users au
		JOIN users u ON au.user_id = u.id
		WHERE au.acknowledgment_id = $1
	`
	rows, err := r.db.Query(query, ackID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AcknowledgmentUser
	for rows.Next() {
		var au models.AcknowledgmentUser
		err := rows.Scan(
			&au.ID, &au.AcknowledgmentID, &au.UserID, &au.ViewedAt, &au.ConfirmedAt, &au.CreatedAt,
			&au.UserName,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, au)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AcknowledgmentRepository) GetPendingForUser(userID uuid.UUID) ([]models.Acknowledgment, error) {
	// Выборка ознакомлений, которые пользователь ещё не подтвердил
	query := `
		SELECT 
			a.id, a.document_id, a.document_type, a.creator_id, a.content, a.created_at, a.completed_at,
			u.full_name as creator_name,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM acknowledgment_users au
		JOIN acknowledgments a ON au.acknowledgment_id = a.id
		JOIN users u ON a.creator_id = u.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE au.user_id = $1 AND au.confirmed_at IS NULL
		ORDER BY a.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Acknowledgment
	for rows.Next() {
		var a models.Acknowledgment
		var docNumber string
		err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentType, &a.CreatorID, &a.Content, &a.CreatedAt, &a.CompletedAt,
			&a.CreatorName, &docNumber,
		)
		if err != nil {
			return nil, err
		}
		a.DocumentNumber = docNumber

		// Загрузка пользователей для контекста
		users, err := r.GetUsersByAcknowledgmentID(a.ID)
		if err != nil {
			return nil, err
		}
		a.Users = users

		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AcknowledgmentRepository) MarkViewed(ackID, userID uuid.UUID) error {
	query := `
		UPDATE acknowledgment_users
		SET viewed_at = $1
		WHERE acknowledgment_id = $2 AND user_id = $3 AND viewed_at IS NULL
	`
	_, err := r.db.Exec(query, time.Now(), ackID, userID)
	return err
}

func (r *AcknowledgmentRepository) MarkConfirmed(ackID, userID uuid.UUID) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Обновление статуса пользователя
	query := `
		UPDATE acknowledgment_users
		SET confirmed_at = $1, viewed_at = COALESCE(viewed_at, $1)
		WHERE acknowledgment_id = $2 AND user_id = $3
	`
	_, err = tx.Exec(query, now, ackID, userID)
	if err != nil {
		return err
	}

	// 2. Проверка, все ли пользователи подтвердили
	checkQuery := `
		SELECT COUNT(*) 
		FROM acknowledgment_users 
		WHERE acknowledgment_id = $1 AND confirmed_at IS NULL
	`
	var remaining int
	err = tx.QueryRow(checkQuery, ackID).Scan(&remaining)
	if err != nil {
		return err
	}

	// 3. Если все подтвердили, обновляем completed_at основного ознакомления
	if remaining == 0 {
		updateQuery := `
			UPDATE acknowledgments
			SET completed_at = $1
			WHERE id = $2
		`
		_, err = tx.Exec(updateQuery, now, ackID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *AcknowledgmentRepository) GetAllActive() ([]models.Acknowledgment, error) {
	query := `
		SELECT 
			a.id, a.document_id, a.document_type, a.creator_id, a.content, a.created_at, a.completed_at,
			u.full_name as creator_name,
			COALESCE(inc.incoming_number, out.outgoing_number) as doc_number
		FROM acknowledgments a
		JOIN users u ON a.creator_id = u.id
		LEFT JOIN incoming_documents inc ON a.document_id = inc.id AND a.document_type = 'incoming'
		LEFT JOIN outgoing_documents out ON a.document_id = out.id AND a.document_type = 'outgoing'
		WHERE a.completed_at IS NULL
		ORDER BY a.created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Acknowledgment
	for rows.Next() {
		var a models.Acknowledgment
		var docNumber string
		err := rows.Scan(
			&a.ID, &a.DocumentID, &a.DocumentType, &a.CreatorID, &a.Content, &a.CreatedAt, &a.CompletedAt,
			&a.CreatorName, &docNumber,
		)
		if err != nil {
			return nil, err
		}
		a.DocumentNumber = docNumber

		users, err := r.GetUsersByAcknowledgmentID(a.ID)
		if err != nil {
			return nil, err
		}
		a.Users = users

		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AcknowledgmentRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM acknowledgments WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
