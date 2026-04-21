package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// OutgoingDocumentRepository предоставляет методы для работы с исходящими документами в БД.
type OutgoingDocumentRepository struct {
	db *database.DB
}

// NewOutgoingDocumentRepository создает новый экземпляр OutgoingDocumentRepository.
func NewOutgoingDocumentRepository(db *database.DB) *OutgoingDocumentRepository {
	return &OutgoingDocumentRepository{db: db}
}

// GetList возвращает список исходящих документов с учетом фильтрации и пагинации.
func (r *OutgoingDocumentRepository) GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	where = append(where, "d.kind = 'outgoing_letter'")

	if filter.AccessibleByUserID != "" {
		accessClauses := make([]string, 0, 2)
		if len(filter.AllowedNomenclatureIDs) > 0 {
			accessClauses = append(accessClauses, fmt.Sprintf("d.nomenclature_id = ANY($%d)", argIdx))
			args = append(args, pq.Array(filter.AllowedNomenclatureIDs))
			argIdx++
		}

		accessClauses = append(accessClauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM assignments a
			WHERE a.document_id = d.id
			  AND (
				a.executor_id = $%d
				OR EXISTS (
					SELECT 1
					FROM assignment_co_executors ce
					WHERE ce.assignment_id = a.id AND ce.user_id = $%d
				)
			  )
		)`, argIdx, argIdx))
		args = append(args, filter.AccessibleByUserID)
		argIdx++

		accessClauses = append(accessClauses, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM acknowledgment_users au
			JOIN acknowledgments a ON au.acknowledgment_id = a.id
			WHERE au.user_id = $%d
			  AND a.document_id = d.id
		)`, argIdx))
		args = append(args, filter.AccessibleByUserID)
		argIdx++

		where = append(where, "("+strings.Join(accessClauses, " OR ")+")")
	}

	if len(filter.NomenclatureIDs) > 0 {
		where = append(where, fmt.Sprintf("d.nomenclature_id = ANY($%d)", argIdx))
		args = append(args, pq.Array(filter.NomenclatureIDs))
		argIdx++
	}
	if filter.DocumentTypeID != "" {
		where = append(where, fmt.Sprintf("d.document_type_id = $%d", argIdx))
		args = append(args, filter.DocumentTypeID)
		argIdx++
	}
	if filter.OrgID != "" {
		where = append(where, fmt.Sprintf("out.recipient_org_id = $%d", argIdx))
		args = append(args, filter.OrgID)
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("out.outgoing_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("out.outgoing_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(d.content ILIKE $%d OR out.outgoing_number ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.OutgoingNumber != "" {
		where = append(where, fmt.Sprintf("out.outgoing_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.OutgoingNumber+"%")
		argIdx++
	}
	if filter.RecipientName != "" {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM organizations o WHERE o.id = out.recipient_org_id AND o.name ILIKE $%d)", argIdx))
		args = append(args, "%"+filter.RecipientName+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Подсчёт
	var totalCount int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM documents d
		JOIN outgoing_document_details out ON out.document_id = d.id
		WHERE %s
	`, whereClause)
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Пагинация по умолчанию
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Данные
	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			out.outgoing_number, out.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.content, d.pages_count,
			out.sender_signatory, out.sender_executor,
			out.recipient_org_id, ro.name as recipient_org_name, out.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM documents d
		JOIN outgoing_document_details out ON out.document_id = d.id
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations ro ON out.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id
		WHERE %s
		ORDER BY d.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing documents: %w", err)
	}
	defer rows.Close()

	items := make([]models.OutgoingDocument, 0)
	for rows.Next() {
		var doc models.OutgoingDocument
		err := rows.Scan(
			&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
			&doc.OutgoingNumber, &doc.OutgoingDate,
			&doc.DocumentTypeID, &doc.DocumentTypeName,
			&doc.Content, &doc.PagesCount,
			&doc.SenderSignatory, &doc.SenderExecutor,
			&doc.RecipientOrgID, &doc.RecipientOrgName, &doc.Addressee,
			&doc.CreatedBy, &doc.CreatedByName,
			&doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.PagedResult[models.OutgoingDocument]{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

// GetByID возвращает исходящий документ по его ID.
func (r *OutgoingDocumentRepository) GetByID(id uuid.UUID) (*models.OutgoingDocument, error) {
	var doc models.OutgoingDocument
	err := r.db.QueryRow(`
		SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			out.outgoing_number, out.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.content, d.pages_count,
			out.sender_signatory, out.sender_executor,
			out.recipient_org_id, ro.name as recipient_org_name, out.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM documents d
		JOIN outgoing_document_details out ON out.document_id = d.id
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations ro ON out.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id
		WHERE d.id = $1 AND d.kind = $2
	`, id, models.DocumentKindOutgoingLetter).Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.OutgoingNumber, &doc.OutgoingDate,
		&doc.DocumentTypeID, &doc.DocumentTypeName,
		&doc.Content, &doc.PagesCount,
		&doc.SenderSignatory, &doc.SenderExecutor,
		&doc.RecipientOrgID, &doc.RecipientOrgName, &doc.Addressee,
		&doc.CreatedBy, &doc.CreatedByName,
		&doc.CreatedAt, &doc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing document: %w", err)
	}

	return &doc, nil
}

// Create создает новый исходящий документ в базе данных.
func (r *OutgoingDocumentRepository) Create(req models.CreateOutgoingDocRequest) (*models.OutgoingDocument, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var id uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO documents (
			kind, nomenclature_id, registration_number, registration_date, document_type_id, content, pages_count, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		models.DocumentKindOutgoingLetter, req.NomenclatureID, req.OutgoingNumber, req.OutgoingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create document root: %w", err)
	}

	if _, err = tx.Exec(`
		INSERT INTO outgoing_document_details (
			document_id, outgoing_number, outgoing_date,
			sender_signatory, sender_executor,
			recipient_org_id, addressee
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
	`,
		id, req.OutgoingNumber, req.OutgoingDate,
		req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee,
	); err != nil {
		return nil, fmt.Errorf("failed to create outgoing document details: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(id)
}

// Update обновляет данные существующего исходящего документа.
func (r *OutgoingDocumentRepository) Update(req models.UpdateOutgoingDocRequest) (*models.OutgoingDocument, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`
		UPDATE documents SET
			document_type_id = $1,
			content = $2,
			pages_count = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND kind = $5
	`,
		req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindOutgoingLetter,
	); err != nil {
		return nil, fmt.Errorf("failed to update document root: %w", err)
	}

	if _, err = tx.Exec(`
		UPDATE outgoing_document_details SET
			outgoing_date = $1,
			sender_signatory = $2,
			sender_executor = $3,
			recipient_org_id = $4,
			addressee = $5
		WHERE document_id = $6
	`,
		req.OutgoingDate, req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee, req.ID,
	); err != nil {
		return nil, fmt.Errorf("failed to update outgoing document details: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(req.ID)
}

// GetCount возвращает общее количество исходящих документов (для дашборда).
func (r *OutgoingDocumentRepository) GetCount() (int, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE kind = $1`, models.DocumentKindOutgoingLetter).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
