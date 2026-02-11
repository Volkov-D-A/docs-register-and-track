package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"docflow/internal/database"
	"docflow/internal/models"
)

type IncomingDocumentRepository struct {
	db *database.DB
}

func NewIncomingDocumentRepository(db *database.DB) *IncomingDocumentRepository {
	return &IncomingDocumentRepository{db: db}
}

func (r *IncomingDocumentRepository) GetList(filter models.DocumentFilter) (*models.PagedResult, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

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
		where = append(where, fmt.Sprintf("(d.sender_org_id = $%d OR d.recipient_org_id = $%d)", argIdx, argIdx))
		args = append(args, filter.OrgID)
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("d.incoming_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("d.incoming_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(d.subject ILIKE $%d OR d.content ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.IncomingNumber != "" {
		where = append(where, fmt.Sprintf("d.incoming_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.IncomingNumber+"%")
		argIdx++
	}
	if filter.OutgoingNumber != "" {
		where = append(where, fmt.Sprintf("d.outgoing_number_sender ILIKE $%d", argIdx))
		args = append(args, "%"+filter.OutgoingNumber+"%")
		argIdx++
	}
	if filter.SenderName != "" {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM organizations o WHERE o.id = d.sender_org_id AND o.name ILIKE $%d)", argIdx))
		args = append(args, "%"+filter.SenderName+"%")
		argIdx++
	}
	if filter.OutgoingDateFrom != "" {
		where = append(where, fmt.Sprintf("d.outgoing_date_sender >= $%d", argIdx))
		args = append(args, filter.OutgoingDateFrom)
		argIdx++
	}
	if filter.OutgoingDateTo != "" {
		where = append(where, fmt.Sprintf("d.outgoing_date_sender <= $%d", argIdx))
		args = append(args, filter.OutgoingDateTo)
		argIdx++
	}
	if filter.NoResolution {
		where = append(where, "d.resolution IS NULL")
	} else if filter.Resolution != "" {
		where = append(where, fmt.Sprintf("d.resolution ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Resolution+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Подсчёт общего количества
	var totalCount int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM incoming_documents d WHERE %s`, whereClause)
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Пагинация
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize

	// Основной запрос с JOIN'ами
	dataQuery := fmt.Sprintf(`
		SELECT d.id, d.nomenclature_id, n.name,
			d.incoming_number, d.incoming_date, d.outgoing_number_sender, d.outgoing_date_sender,
			d.intermediate_number, d.intermediate_date,
			d.document_type_id, dt.name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name, d.addressee,
			d.resolution,
			d.created_by, u.full_name,
			d.created_at, d.updated_at
		FROM incoming_documents d
		LEFT JOIN nomenclature n ON d.nomenclature_id = n.id
		LEFT JOIN document_types dt ON d.document_type_id = dt.id
		LEFT JOIN organizations so ON d.sender_org_id = so.id
		LEFT JOIN organizations ro ON d.recipient_org_id = ro.id
		LEFT JOIN users u ON d.created_by = u.id
		WHERE %s
		ORDER BY d.incoming_date DESC, d.incoming_number DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming documents: %w", err)
	}
	defer rows.Close()

	var items []models.IncomingDocument
	for rows.Next() {
		var doc models.IncomingDocument
		if err := rows.Scan(
			&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
			&doc.IncomingNumber, &doc.IncomingDate, &doc.OutgoingNumberSender, &doc.OutgoingDateSender,
			&doc.IntermediateNumber, &doc.IntermediateDate,
			&doc.DocumentTypeID, &doc.DocumentTypeName,
			&doc.Subject, &doc.PagesCount, &doc.Content,
			&doc.SenderOrgID, &doc.SenderOrgName, &doc.SenderSignatory, &doc.SenderExecutor,
			&doc.RecipientOrgID, &doc.RecipientOrgName, &doc.Addressee,
			&doc.Resolution,
			&doc.CreatedBy, &doc.CreatedByName,
			&doc.CreatedAt, &doc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		doc.FillIDStr()
		items = append(items, doc)
	}

	return &models.PagedResult{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

func (r *IncomingDocumentRepository) GetByID(id uuid.UUID) (*models.IncomingDocument, error) {
	doc := &models.IncomingDocument{}
	err := r.db.QueryRow(`
		SELECT d.id, d.nomenclature_id, n.name,
			d.incoming_number, d.incoming_date, d.outgoing_number_sender, d.outgoing_date_sender,
			d.intermediate_number, d.intermediate_date,
			d.document_type_id, dt.name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name, d.addressee,
			d.resolution,
			d.created_by, u.full_name,
			d.created_at, d.updated_at
		FROM incoming_documents d
		LEFT JOIN nomenclature n ON d.nomenclature_id = n.id
		LEFT JOIN document_types dt ON d.document_type_id = dt.id
		LEFT JOIN organizations so ON d.sender_org_id = so.id
		LEFT JOIN organizations ro ON d.recipient_org_id = ro.id
		LEFT JOIN users u ON d.created_by = u.id
		WHERE d.id = $1
	`, id).Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.IncomingNumber, &doc.IncomingDate, &doc.OutgoingNumberSender, &doc.OutgoingDateSender,
		&doc.IntermediateNumber, &doc.IntermediateDate,
		&doc.DocumentTypeID, &doc.DocumentTypeName,
		&doc.Subject, &doc.PagesCount, &doc.Content,
		&doc.SenderOrgID, &doc.SenderOrgName, &doc.SenderSignatory, &doc.SenderExecutor,
		&doc.RecipientOrgID, &doc.RecipientOrgName, &doc.Addressee,
		&doc.Resolution,
		&doc.CreatedBy, &doc.CreatedByName,
		&doc.CreatedAt, &doc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming document: %w", err)
	}

	doc.FillIDStr()
	return doc, nil
}

func (r *IncomingDocumentRepository) Create(
	nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy uuid.UUID,
	incomingNumber string, incomingDate time.Time,
	outgoingNumberSender string, outgoingDateSender time.Time,
	intermediateNumber *string, intermediateDate *time.Time,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor, addressee string,
	resolution *string,
) (*models.IncomingDocument, error) {
	var id uuid.UUID
	err := r.db.QueryRow(`
		INSERT INTO incoming_documents (
			nomenclature_id, incoming_number, incoming_date,
			outgoing_number_sender, outgoing_date_sender,
			intermediate_number, intermediate_date,
			document_type_id, subject, pages_count, content,
			sender_org_id, sender_signatory, sender_executor,
			recipient_org_id, addressee, resolution, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id
	`,
		nomenclatureID, incomingNumber, incomingDate,
		outgoingNumberSender, outgoingDateSender,
		intermediateNumber, intermediateDate,
		documentTypeID, subject, pagesCount, content,
		senderOrgID, senderSignatory, senderExecutor,
		recipientOrgID, addressee, resolution, createdBy,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create incoming document: %w", err)
	}

	return r.GetByID(id)
}

func (r *IncomingDocumentRepository) Update(
	id uuid.UUID,
	documentTypeID, senderOrgID, recipientOrgID uuid.UUID,
	outgoingNumberSender string, outgoingDateSender time.Time,
	intermediateNumber *string, intermediateDate *time.Time,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor, addressee string,
	resolution *string,
) (*models.IncomingDocument, error) {
	_, err := r.db.Exec(`
		UPDATE incoming_documents SET
			outgoing_number_sender = $1, outgoing_date_sender = $2,
			intermediate_number = $3, intermediate_date = $4,
			document_type_id = $5, subject = $6, pages_count = $7, content = $8,
			sender_org_id = $9, sender_signatory = $10, sender_executor = $11,
			recipient_org_id = $12, addressee = $13, resolution = $14,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $15
	`,
		outgoingNumberSender, outgoingDateSender,
		intermediateNumber, intermediateDate,
		documentTypeID, subject, pagesCount, content,
		senderOrgID, senderSignatory, senderExecutor,
		recipientOrgID, addressee, resolution,
		id,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update incoming document: %w", err)
	}

	return r.GetByID(id)
}

func (r *IncomingDocumentRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM incoming_documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete incoming document: %w", err)
	}
	return nil
}

// GetCount — общее количество для дашборда
func (r *IncomingDocumentRepository) GetCount() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM incoming_documents`).Scan(&count)
	return count, err
}
