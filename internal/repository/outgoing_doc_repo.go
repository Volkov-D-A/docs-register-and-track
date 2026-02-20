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

type OutgoingDocumentRepository struct {
	db *database.DB
}

func NewOutgoingDocumentRepository(db *database.DB) *OutgoingDocumentRepository {
	return &OutgoingDocumentRepository{db: db}
}

func (r *OutgoingDocumentRepository) GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error) {
	where := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

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
		where = append(where, fmt.Sprintf("d.recipient_org_id = $%d", argIdx))
		args = append(args, filter.OrgID)
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("d.outgoing_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("d.outgoing_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(d.subject ILIKE $%d OR d.content ILIKE $%d OR d.outgoing_number ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.OutgoingNumber != "" {
		where = append(where, fmt.Sprintf("d.outgoing_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.OutgoingNumber+"%")
		argIdx++
	}
	if filter.RecipientName != "" {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM organizations o WHERE o.id = d.recipient_org_id AND o.name ILIKE $%d)", argIdx))
		args = append(args, "%"+filter.RecipientName+"%")
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	var totalCount int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM outgoing_documents d WHERE %s`, whereClause)
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Pagination defaults
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Data
	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			d.outgoing_number, d.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name as sender_org_name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name as recipient_org_name, d.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM outgoing_documents d
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations so ON d.sender_org_id = so.id
		JOIN organizations ro ON d.recipient_org_id = ro.id
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

	var items []models.OutgoingDocument
	for rows.Next() {
		var doc models.OutgoingDocument
		err := rows.Scan(
			&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
			&doc.OutgoingNumber, &doc.OutgoingDate,
			&doc.DocumentTypeID, &doc.DocumentTypeName,
			&doc.Subject, &doc.PagesCount, &doc.Content,
			&doc.SenderOrgID, &doc.SenderOrgName, &doc.SenderSignatory, &doc.SenderExecutor,
			&doc.RecipientOrgID, &doc.RecipientOrgName, &doc.Addressee,
			&doc.CreatedBy, &doc.CreatedByName,
			&doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		doc.FillIDStr()
		items = append(items, doc)
	}

	return &models.PagedResult[models.OutgoingDocument]{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

func (r *OutgoingDocumentRepository) GetByID(id uuid.UUID) (*models.OutgoingDocument, error) {
	var doc models.OutgoingDocument
	err := r.db.QueryRow(`
		SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			d.outgoing_number, d.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name as sender_org_name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name as recipient_org_name, d.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM outgoing_documents d
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations so ON d.sender_org_id = so.id
		JOIN organizations ro ON d.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id
		WHERE d.id = $1
	`, id).Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.OutgoingNumber, &doc.OutgoingDate,
		&doc.DocumentTypeID, &doc.DocumentTypeName,
		&doc.Subject, &doc.PagesCount, &doc.Content,
		&doc.SenderOrgID, &doc.SenderOrgName, &doc.SenderSignatory, &doc.SenderExecutor,
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

	doc.FillIDStr()
	return &doc, nil
}

func (r *OutgoingDocumentRepository) Create(
	nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy uuid.UUID,
	outgoingNumber string, outgoingDate time.Time,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor, addressee string,
) (*models.OutgoingDocument, error) {
	var id uuid.UUID
	err := r.db.QueryRow(`
		INSERT INTO outgoing_documents (
			nomenclature_id, outgoing_number, outgoing_date,
			document_type_id, subject, pages_count, content,
			sender_org_id, sender_signatory, sender_executor,
			recipient_org_id, addressee, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id
	`,
		nomenclatureID, outgoingNumber, outgoingDate,
		documentTypeID, subject, pagesCount, content,
		senderOrgID, senderSignatory, senderExecutor,
		recipientOrgID, addressee, createdBy,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create outgoing document: %w", err)
	}

	return r.GetByID(id)
}

func (r *OutgoingDocumentRepository) Update(
	id uuid.UUID,
	documentTypeID, senderOrgID, recipientOrgID uuid.UUID,
	outgoingDate time.Time,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor, addressee string,
) (*models.OutgoingDocument, error) {
	_, err := r.db.Exec(`
		UPDATE outgoing_documents SET
			document_type_id = $1, subject = $2, pages_count = $3, content = $4,
			sender_org_id = $5, sender_signatory = $6, sender_executor = $7,
			recipient_org_id = $8, addressee = $9, outgoing_date = $10,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $11
	`,
		documentTypeID, subject, pagesCount, content,
		senderOrgID, senderSignatory, senderExecutor,
		recipientOrgID, addressee, outgoingDate,
		id,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update outgoing document: %w", err)
	}

	return r.GetByID(id)
}

func (r *OutgoingDocumentRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM outgoing_documents WHERE id = $1`, id)
	return err
}

func (r *OutgoingDocumentRepository) GetCount() (int, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM outgoing_documents`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
