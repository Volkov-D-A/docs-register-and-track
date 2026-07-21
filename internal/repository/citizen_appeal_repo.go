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

// CitizenAppealRepository предоставляет методы для работы с обращениями граждан в БД.
type CitizenAppealRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *CitizenAppealRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

// NewCitizenAppealRepository создает новый экземпляр CitizenAppealRepository.
func NewCitizenAppealRepository(db *database.DB) *CitizenAppealRepository {
	return &CitizenAppealRepository{db: db}
}

const citizenAppealSelectBase = `
	SELECT d.id, d.nomenclature_id, n.index || ' — ' || n.name,
		d.registration_number, d.registration_date,
		ca.appeal_date,
		d.document_type, d.document_type,
		d.content, d.pages_count,
		ca.applicant_full_name, ca.registration_address,
		ca.appeal_type, ca.applicant_category,
		ca.appeal_pages_count, ca.attachment_pages_count,
		ca.has_envelope, ca.received_from_pos,
		d.created_by, u.full_name,
		d.created_at, d.updated_at
	FROM documents d
	JOIN citizen_appeal_details ca ON ca.document_id = d.id
	LEFT JOIN nomenclature n ON d.nomenclature_id = n.id
	LEFT JOIN users u ON d.created_by = u.id`

func scanCitizenAppealDoc(scanner interface{ Scan(...interface{}) error }) (*models.CitizenAppealDocument, error) {
	doc := &models.CitizenAppealDocument{}
	err := scanner.Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.RegistrationNumber, &doc.RegistrationDate,
		&doc.AppealDate,
		&doc.DocumentTypeID, &doc.DocumentTypeName,
		&doc.Content, &doc.PagesCount,
		&doc.ApplicantFullName, &doc.RegistrationAddress,
		&doc.AppealType, &doc.ApplicantCategory,
		&doc.AppealPagesCount, &doc.AttachmentPagesCount,
		&doc.HasEnvelope, &doc.ReceivedFromPOS,
		&doc.CreatedBy, &doc.CreatedByName,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	return doc, err
}

func (r *CitizenAppealRepository) hydrateCommonDetails(doc *models.CitizenAppealDocument) error {
	if doc == nil {
		return nil
	}
	correspondents, err := r.loadCorrespondents(doc.ID)
	if err != nil {
		return err
	}
	doc.Correspondents = correspondents

	resolutions, err := r.loadResolutions(doc.ID)
	if err != nil {
		return err
	}
	doc.Resolutions = resolutions
	return nil
}

func (r *CitizenAppealRepository) loadCorrespondents(documentID uuid.UUID) ([]models.DocumentCorrespondentRegistration, error) {
	rows, err := r.db.Query(`
		SELECT cr.id, cr.document_id, cr.registration_number, cr.registration_date,
			cr.correspondent_org_id, o.name, cr.position
		FROM document_correspondent_registrations cr
		JOIN organizations o ON o.id = cr.correspondent_org_id
		WHERE cr.document_id = $1
		ORDER BY cr.position, cr.created_at, cr.id
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document correspondents: %w", err)
	}
	defer rows.Close()

	items := make([]models.DocumentCorrespondentRegistration, 0)
	for rows.Next() {
		var item models.DocumentCorrespondentRegistration
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.RegistrationNumber, &item.RegistrationDate,
			&item.CorrespondentOrgID, &item.CorrespondentName, &item.Position,
		); err != nil {
			return nil, fmt.Errorf("scan document correspondent error: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document correspondents rows error: %w", err)
	}

	return items, nil
}

func (r *CitizenAppealRepository) loadResolutions(documentID uuid.UUID) ([]models.DocumentResolution, error) {
	return loadDocumentResolutions(r.db, documentID)
}

// GetList возвращает список обращений граждан с учетом фильтрации и пагинации.
func (r *CitizenAppealRepository) GetList(filter models.DocumentFilter) (*models.PagedResult[models.CitizenAppealDocument], error) {
	where := []string{"1=1", "d.kind = 'citizen_appeal'"}
	args := []interface{}{}
	argIdx := 1

	scope := documentListAccessScope(filter.AccessScope, filter.AllowedNomenclatureIDs, filter.AccessibleByUserID, filter.AccessibleByUserIDs)
	applyDocumentListAccess(&where, &args, &argIdx, scope)

	if len(filter.NomenclatureIDs) > 0 {
		where = append(where, fmt.Sprintf("d.nomenclature_id = ANY($%d)", argIdx))
		args = append(args, pq.Array(filter.NomenclatureIDs))
		argIdx++
	} else if filter.NomenclatureID != "" {
		where = append(where, fmt.Sprintf("d.nomenclature_id = $%d", argIdx))
		args = append(args, filter.NomenclatureID)
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("d.registration_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("d.registration_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.AppealDateFrom != "" {
		where = append(where, fmt.Sprintf("ca.appeal_date >= $%d", argIdx))
		args = append(args, filter.AppealDateFrom)
		argIdx++
	}
	if filter.AppealDateTo != "" {
		where = append(where, fmt.Sprintf("ca.appeal_date <= $%d", argIdx))
		args = append(args, filter.AppealDateTo)
		argIdx++
	}
	if filter.RegistrationNumber != "" {
		where = append(where, fmt.Sprintf("d.registration_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.RegistrationNumber+"%")
		argIdx++
	} else if filter.IncomingNumber != "" {
		where = append(where, fmt.Sprintf("d.registration_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.IncomingNumber+"%")
		argIdx++
	}
	if filter.ApplicantName != "" {
		where = append(where, fmt.Sprintf("ca.applicant_full_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.ApplicantName+"%")
		argIdx++
	} else if filter.SenderName != "" {
		where = append(where, fmt.Sprintf("ca.applicant_full_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.SenderName+"%")
		argIdx++
	}
	if filter.AppealType != "" {
		where = append(where, fmt.Sprintf("ca.appeal_type = $%d", argIdx))
		args = append(args, filter.AppealType)
		argIdx++
	}
	if filter.OrgID != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM document_correspondent_registrations cr
			WHERE cr.document_id = d.id AND cr.correspondent_org_id = $%d
		)`, argIdx))
		args = append(args, filter.OrgID)
		argIdx++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf(`(
			d.registration_number ILIKE $%d
			OR d.content ILIKE $%d
			OR ca.applicant_full_name ILIKE $%d
			OR ca.registration_address ILIKE $%d
			OR ca.applicant_category ILIKE $%d
			OR EXISTS (
				SELECT 1
				FROM document_correspondent_registrations cr
				JOIN organizations o ON o.id = cr.correspondent_org_id
				WHERE cr.document_id = d.id
				  AND (cr.registration_number ILIKE $%d OR o.name ILIKE $%d)
			)
		)`, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.NoResolution {
		where = append(where, `NOT EXISTS (
			SELECT 1
			FROM document_resolutions dr
			WHERE dr.document_id = d.id
			  AND dr.resolution IS NOT NULL
			  AND btrim(dr.resolution) <> ''
		)`)
	} else if filter.Resolution != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM document_resolutions dr
			WHERE dr.document_id = d.id AND dr.resolution ILIKE $%d
		)`, argIdx))
		args = append(args, "%"+filter.Resolution+"%")
		argIdx++
	}

	var totalCount int
	if err := applyDocumentCursor(&where, &args, &argIdx, filter.CursorPagination, filter.Cursor); err != nil {
		return nil, err
	}
	filter.Page, filter.PageSize = normalizePagination(filter.Page, filter.PageSize)
	whereClause := strings.Join(where, " AND ")
	if !filter.CursorPagination {
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM documents d
			JOIN citizen_appeal_details ca ON ca.document_id = d.id WHERE %s`, whereClause)
		if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
			return nil, fmt.Errorf("failed to count citizen appeals: %w", err)
		}
	}
	limit := filter.PageSize
	if filter.CursorPagination {
		limit++
	}

	dataQuery := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY d.created_at DESC, d.id DESC
		LIMIT $%d%s
	`, citizenAppealSelectBase, whereClause, argIdx, map[bool]string{true: "", false: fmt.Sprintf(" OFFSET $%d", argIdx+1)}[filter.CursorPagination])
	args = append(args, limit)
	if !filter.CursorPagination {
		args = append(args, (filter.Page-1)*filter.PageSize)
	}

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get citizen appeals: %w", err)
	}
	defer rows.Close()

	items := make([]models.CitizenAppealDocument, 0)
	documentIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		doc, err := scanCitizenAppealDoc(rows)
		if err != nil {
			return nil, fmt.Errorf("scan citizen appeal error: %w", err)
		}
		documentIDs = append(documentIDs, doc.ID)
		items = append(items, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("citizen appeals rows error: %w", err)
	}
	hasMore := filter.CursorPagination && len(items) > filter.PageSize
	if hasMore {
		items = items[:filter.PageSize]
		documentIDs = documentIDs[:filter.PageSize]
	}
	if len(documentIDs) > 0 {
		correspondentsByDocumentID, err := loadDocumentCorrespondentsByDocumentIDs(r.db, documentIDs)
		if err != nil {
			return nil, err
		}
		resolutionsByDocumentID, err := loadDocumentResolutionsByDocumentIDs(r.db, documentIDs)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].Correspondents = correspondentsByDocumentID[items[i].ID]
			items[i].Resolutions = resolutionsByDocumentID[items[i].ID]
		}
	}

	nextCursor := ""
	if hasMore {
		var err error
		last := items[len(items)-1]
		nextCursor, err = models.EncodeDocumentCursor(last.CreatedAt, last.ID)
		if err != nil {
			return nil, err
		}
	}
	return &models.PagedResult[models.CitizenAppealDocument]{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// GetByID возвращает обращения граждан по ID.
func (r *CitizenAppealRepository) GetByID(id uuid.UUID) (*models.CitizenAppealDocument, error) {
	query := citizenAppealSelectBase + " WHERE d.id = $1 AND d.kind = $2"
	doc, err := scanCitizenAppealDoc(r.db.QueryRow(query, id, models.DocumentKindCitizenAppeal))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get citizen appeal: %w", err)
	}

	if err := r.hydrateCommonDetails(doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// GetByIDs loads graph card data in batches rather than one query per node.
func (r *CitizenAppealRepository) GetByIDs(ids []uuid.UUID) ([]models.CitizenAppealDocument, error) {
	if len(ids) == 0 {
		return []models.CitizenAppealDocument{}, nil
	}
	rows, err := r.db.Query(citizenAppealSelectBase+` WHERE d.id = ANY($1) AND d.kind = $2 ORDER BY d.id`, pq.Array(ids), models.DocumentKindCitizenAppeal)
	if err != nil {
		return nil, fmt.Errorf("failed to get citizen appeals by IDs: %w", err)
	}
	defer rows.Close()
	items := make([]models.CitizenAppealDocument, 0, len(ids))
	documentIDs := make([]uuid.UUID, 0, len(ids))
	for rows.Next() {
		doc, err := scanCitizenAppealDoc(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *doc)
		documentIDs = append(documentIDs, doc.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	correspondents, err := loadDocumentCorrespondentsByDocumentIDs(r.db, documentIDs)
	if err != nil {
		return nil, err
	}
	resolutions, err := loadDocumentResolutionsByDocumentIDs(r.db, documentIDs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Correspondents = correspondents[items[i].ID]
		items[i].Resolutions = resolutions[items[i].ID]
	}
	return items, nil
}

// Create создает новое обращения граждан в базе данных.
func (r *CitizenAppealRepository) Create(req models.CreateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	return r.create(req, nil, "", "")
}
func (r *CitizenAppealRepository) CreateWithJournal(req models.CreateCitizenAppealDocRequest, action, detailsFormat string) (*models.CitizenAppealDocument, error) {
	return r.create(req, nil, action, detailsFormat)
}
func (r *CitizenAppealRepository) create(req models.CreateCitizenAppealDocRequest, effects []models.OutboxEvent, journalAction, journalDetailsFormat string) (*models.CitizenAppealDocument, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var registration *registrationNumberResult
	if req.AdminNumberOverride != nil {
		registration, err = resolveAdminRegistrationNumberTx(tx, req.CreatedBy, models.DocumentKindCitizenAppeal, req.NomenclatureID, req.IdempotencyKey, req.AdminNumberOverride)
	} else {
		registration, err = resolveRegistrationNumberTx(tx, req.CreatedBy, models.DocumentKindCitizenAppeal, req.NomenclatureID, req.IdempotencyKey, req.RegistrationNumber)
	}
	if err != nil {
		return nil, err
	}
	if registration.Existing != uuid.Nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit idempotent transaction: %w", err)
		}
		return r.GetByID(registration.Existing)
	}
	req.RegistrationNumber = registration.Number

	pagesCount := req.AppealPagesCount + req.AttachmentPagesCount
	if pagesCount <= 0 {
		pagesCount = 1
	}

	var id uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO documents (
			kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`,
		models.DocumentKindCitizenAppeal, req.NomenclatureID, req.IdempotencyKey, req.RegistrationNumber, req.RegistrationDate,
		models.DocumentTypeCitizenAppeal, req.Content, pagesCount, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err, "idx_documents_created_by_kind_idempotency") {
			_ = tx.Rollback()
			existingID, lookupErr := findExistingDocumentIDByIdempotency(r.db, req.CreatedBy, models.DocumentKindCitizenAppeal, req.IdempotencyKey)
			if lookupErr != nil {
				return nil, fmt.Errorf("failed to resolve idempotent document: %w", lookupErr)
			}
			return r.GetByID(existingID)
		}
		if isUniqueViolation(err, "idx_documents_kind_registration_number_year") {
			return nil, models.NewConflict("документ с таким регистрационным номером уже существует")
		}
		return nil, fmt.Errorf("failed to create citizen appeal root: %w", err)
	}

	if _, err = tx.Exec(`
		INSERT INTO citizen_appeal_details (
			document_id, appeal_date, applicant_full_name, registration_address,
			appeal_type, applicant_category, appeal_pages_count, attachment_pages_count,
			has_envelope, received_from_pos
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`,
		id, req.AppealDate, req.ApplicantFullName, req.RegistrationAddress,
		req.AppealType, req.ApplicantCategory, req.AppealPagesCount, req.AttachmentPagesCount,
		req.HasEnvelope, req.ReceivedFromPOS,
	); err != nil {
		return nil, fmt.Errorf("failed to create citizen appeal details: %w", err)
	}

	if err := replaceCorrespondents(tx, id, req.Correspondents); err != nil {
		return nil, err
	}
	if err := replaceResolutions(tx, id, req.Resolutions); err != nil {
		return nil, err
	}
	if journalAction != "" {
		if r.outbox == nil {
			return nil, fmt.Errorf("outbox repository is required for document journal")
		}
		payload := fmt.Sprintf(`{"documentId":"%s","userId":"%s","action":%q,"details":%q}`, id, req.CreatedBy, journalAction, fmt.Sprintf(journalDetailsFormat, req.RegistrationNumber))
		if err := r.outbox.EnqueueTx(tx, models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "citizen-appeal:" + id.String() + ":create:journal", Payload: payload}); err != nil {
			return nil, err
		}
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(id)
}

// Update обновляет данные существующего обращения граждан.
func (r *CitizenAppealRepository) Update(req models.UpdateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	return r.update(req, nil)
}
func (r *CitizenAppealRepository) UpdateWithOutbox(req models.UpdateCitizenAppealDocRequest, effects []models.OutboxEvent) (*models.CitizenAppealDocument, error) {
	return r.update(req, effects)
}
func (r *CitizenAppealRepository) update(req models.UpdateCitizenAppealDocRequest, effects []models.OutboxEvent) (*models.CitizenAppealDocument, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	pagesCount := req.AppealPagesCount + req.AttachmentPagesCount
	if pagesCount <= 0 {
		pagesCount = 1
	}

	if _, err = tx.Exec(`
		UPDATE documents SET
			registration_number = $1,
			registration_date = $2,
			content = $3,
			pages_count = $4,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND kind = $6
	`,
		req.RegistrationNumber, req.RegistrationDate, req.Content, pagesCount,
		req.ID, models.DocumentKindCitizenAppeal,
	); err != nil {
		return nil, fmt.Errorf("failed to update citizen appeal root: %w", err)
	}

	if _, err = tx.Exec(`
		UPDATE citizen_appeal_details SET
			appeal_date = $1,
			applicant_full_name = $2,
			registration_address = $3,
			appeal_type = $4,
			applicant_category = $5,
			appeal_pages_count = $6,
			attachment_pages_count = $7,
			has_envelope = $8,
			received_from_pos = $9
		WHERE document_id = $10
	`,
		req.AppealDate, req.ApplicantFullName, req.RegistrationAddress,
		req.AppealType, req.ApplicantCategory,
		req.AppealPagesCount, req.AttachmentPagesCount,
		req.HasEnvelope, req.ReceivedFromPOS,
		req.ID,
	); err != nil {
		return nil, fmt.Errorf("failed to update citizen appeal details: %w", err)
	}

	if err := replaceCorrespondents(tx, req.ID, req.Correspondents); err != nil {
		return nil, err
	}
	if err := replaceResolutions(tx, req.ID, req.Resolutions); err != nil {
		return nil, err
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(req.ID)
}

// GetCount — общее количество обращений граждан.
func (r *CitizenAppealRepository) GetCount() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE kind = $1`, models.DocumentKindCitizenAppeal).Scan(&count)
	return count, err
}
