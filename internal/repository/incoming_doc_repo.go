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

// IncomingDocumentRepository предоставляет методы для работы с входящими документами в БД.
type IncomingDocumentRepository struct {
	db *database.DB
}

// NewIncomingDocumentRepository создает новый экземпляр IncomingDocumentRepository.
func NewIncomingDocumentRepository(db *database.DB) *IncomingDocumentRepository {
	return &IncomingDocumentRepository{db: db}
}

// incomingDocSelectBase — базовый SELECT с JOIN'ами для входящих документов.
const incomingDocSelectBase = `
	SELECT d.id, d.nomenclature_id, n.index || ' — ' || n.name,
		inc.incoming_number, inc.incoming_date, inc.outgoing_number_sender, inc.outgoing_date_sender,
		inc.intermediate_number, inc.intermediate_date,
		d.document_type_id, dt.name,
		d.content, d.pages_count,
		inc.sender_org_id, so.name, inc.sender_signatory,
		inc.resolution, inc.resolution_author, inc.resolution_executors,
		d.created_by, u.full_name,
		d.created_at, d.updated_at
	FROM documents d
	JOIN incoming_document_details inc ON inc.document_id = d.id
	LEFT JOIN nomenclature n ON d.nomenclature_id = n.id
	LEFT JOIN document_types dt ON d.document_type_id = dt.id
	LEFT JOIN organizations so ON inc.sender_org_id = so.id
	LEFT JOIN users u ON d.created_by = u.id`

// scanIncomingDoc сканирует строку результата в структуру IncomingDocument.
func scanIncomingDoc(scanner interface{ Scan(...interface{}) error }) (*models.IncomingDocument, error) {
	doc := &models.IncomingDocument{}
	err := scanner.Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.IncomingNumber, &doc.IncomingDate, &doc.OutgoingNumberSender, &doc.OutgoingDateSender,
		&doc.IntermediateNumber, &doc.IntermediateDate,
		&doc.DocumentTypeID, &doc.DocumentTypeName,
		&doc.Content, &doc.PagesCount,
		&doc.SenderOrgID, &doc.SenderOrgName, &doc.SenderSignatory,
		&doc.Resolution, &doc.ResolutionAuthor, &doc.ResolutionExecutors,
		&doc.CreatedBy, &doc.CreatedByName,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	return doc, err
}

// GetList возвращает список входящих документов с учетом фильтрации и пагинации.
func (r *IncomingDocumentRepository) GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	where = append(where, "d.kind = 'incoming_letter'")

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
	} else if filter.NomenclatureID != "" {
		where = append(where, fmt.Sprintf("d.nomenclature_id = $%d", argIdx))
		args = append(args, filter.NomenclatureID)
		argIdx++
	}
	if filter.DocumentTypeID != "" {
		where = append(where, fmt.Sprintf("d.document_type_id = $%d", argIdx))
		args = append(args, filter.DocumentTypeID)
		argIdx++
	}
	if filter.OrgID != "" {
		where = append(where, fmt.Sprintf("inc.sender_org_id = $%d", argIdx))
		args = append(args, filter.OrgID)
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("inc.incoming_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("inc.incoming_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(d.content ILIKE $%d OR inc.incoming_number ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.IncomingNumber != "" {
		where = append(where, fmt.Sprintf("inc.incoming_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.IncomingNumber+"%")
		argIdx++
	}
	if filter.OutgoingNumber != "" {
		where = append(where, fmt.Sprintf("inc.outgoing_number_sender ILIKE $%d", argIdx))
		args = append(args, "%"+filter.OutgoingNumber+"%")
		argIdx++
	}
	if filter.SenderName != "" {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM organizations o WHERE o.id = inc.sender_org_id AND o.name ILIKE $%d)", argIdx))
		args = append(args, "%"+filter.SenderName+"%")
		argIdx++
	}
	if filter.OutgoingDateFrom != "" {
		where = append(where, fmt.Sprintf("inc.outgoing_date_sender >= $%d", argIdx))
		args = append(args, filter.OutgoingDateFrom)
		argIdx++
	}
	if filter.OutgoingDateTo != "" {
		where = append(where, fmt.Sprintf("inc.outgoing_date_sender <= $%d", argIdx))
		args = append(args, filter.OutgoingDateTo)
		argIdx++
	}
	if filter.NoResolution {
		where = append(where, "inc.resolution IS NULL")
	} else if filter.Resolution != "" {
		where = append(where, fmt.Sprintf("inc.resolution ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Resolution+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Подсчёт общего количества
	var totalCount int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM documents d
		JOIN incoming_document_details inc ON inc.document_id = d.id
		WHERE %s
	`, whereClause)
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Пагинация
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize

	// Основной запрос с использованием базового SELECT
	dataQuery := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY d.created_at DESC
		LIMIT $%d OFFSET $%d
	`, incomingDocSelectBase, whereClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming documents: %w", err)
	}
	defer rows.Close()

	items := make([]models.IncomingDocument, 0)
	for rows.Next() {
		doc, err := scanIncomingDoc(rows)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		items = append(items, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return &models.PagedResult[models.IncomingDocument]{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

// GetByID возвращает входящий документ по его ID.
func (r *IncomingDocumentRepository) GetByID(id uuid.UUID) (*models.IncomingDocument, error) {
	query := incomingDocSelectBase + " WHERE d.id = $1"
	doc, err := scanIncomingDoc(r.db.QueryRow(query, id))

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming document: %w", err)
	}

	return doc, nil
}

// Create создает новый входящий документ в базе данных.
func (r *IncomingDocumentRepository) Create(req models.CreateIncomingDocRequest) (*models.IncomingDocument, error) {
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
		models.DocumentKindIncomingLetter, req.NomenclatureID, req.IncomingNumber, req.IncomingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create document root: %w", err)
	}

	if _, err = tx.Exec(`
		INSERT INTO incoming_document_details (
			document_id, incoming_number, incoming_date,
			outgoing_number_sender, outgoing_date_sender,
			intermediate_number, intermediate_date,
			sender_org_id, sender_signatory,
			resolution, resolution_author, resolution_executors
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`,
		id, req.IncomingNumber, req.IncomingDate,
		req.OutgoingNumberSender, req.OutgoingDateSender,
		req.IntermediateNumber, req.IntermediateDate,
		req.SenderOrgID, req.SenderSignatory,
		req.Resolution, req.ResolutionAuthor, req.ResolutionExecutors,
	); err != nil {
		return nil, fmt.Errorf("failed to create incoming document details: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(id)
}

// Update обновляет данные существующего входящего документа.
func (r *IncomingDocumentRepository) Update(req models.UpdateIncomingDocRequest) (*models.IncomingDocument, error) {
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
		req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindIncomingLetter,
	); err != nil {
		return nil, fmt.Errorf("failed to update document root: %w", err)
	}

	if _, err = tx.Exec(`
		UPDATE incoming_document_details SET
			outgoing_number_sender = $1,
			outgoing_date_sender = $2,
			intermediate_number = $3,
			intermediate_date = $4,
			sender_org_id = $5,
			sender_signatory = $6,
			resolution = $7,
			resolution_author = $8,
			resolution_executors = $9
		WHERE document_id = $10
	`,
		req.OutgoingNumberSender, req.OutgoingDateSender,
		req.IntermediateNumber, req.IntermediateDate,
		req.SenderOrgID, req.SenderSignatory,
		req.Resolution, req.ResolutionAuthor, req.ResolutionExecutors,
		req.ID,
	); err != nil {
		return nil, fmt.Errorf("failed to update incoming document details: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(req.ID)
}

// Delete удаляет входящий документ по его ID.
func (r *IncomingDocumentRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM documents WHERE id = $1 AND kind = $2`, id, models.DocumentKindIncomingLetter)
	if err != nil {
		return fmt.Errorf("failed to delete incoming document: %w", err)
	}
	return nil
}

// GetCount — общее количество для дашборда
func (r *IncomingDocumentRepository) GetCount() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE kind = $1`, models.DocumentKindIncomingLetter).Scan(&count)
	return count, err
}
