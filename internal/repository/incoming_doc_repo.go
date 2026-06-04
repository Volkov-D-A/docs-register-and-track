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
		inc.incoming_number, inc.incoming_date,
		d.document_type, d.document_type,
		d.content, d.pages_count,
		inc.sender_signatory,
		d.created_by, u.full_name,
		d.created_at, d.updated_at
	FROM documents d
	JOIN incoming_document_details inc ON inc.document_id = d.id
	LEFT JOIN nomenclature n ON d.nomenclature_id = n.id
	LEFT JOIN users u ON d.created_by = u.id`

// scanIncomingDoc сканирует строку результата в структуру IncomingDocument.
func scanIncomingDoc(scanner interface{ Scan(...interface{}) error }) (*models.IncomingDocument, error) {
	doc := &models.IncomingDocument{}
	err := scanner.Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.IncomingNumber, &doc.IncomingDate,
		&doc.DocumentTypeID, &doc.DocumentTypeName,
		&doc.Content, &doc.PagesCount,
		&doc.SenderSignatory,
		&doc.CreatedBy, &doc.CreatedByName,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	return doc, err
}

func (r *IncomingDocumentRepository) loadResolution(documentID uuid.UUID) (*models.DocumentResolution, error) {
	resolution := &models.DocumentResolution{}
	err := r.db.QueryRow(`
		SELECT id, document_id, resolution, resolution_author, resolution_executors, position
		FROM document_resolutions
		WHERE document_id = $1
		ORDER BY position, created_at, id
		LIMIT 1
	`, documentID).Scan(
		&resolution.ID, &resolution.DocumentID, &resolution.Resolution,
		&resolution.ResolutionAuthor, &resolution.ResolutionExecutors, &resolution.Position,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document resolution: %w", err)
	}
	return resolution, nil
}

func loadFirstDocumentResolutionsByDocumentIDs(db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}, documentIDs []uuid.UUID) (map[uuid.UUID]*models.DocumentResolution, error) {
	result := make(map[uuid.UUID]*models.DocumentResolution, len(documentIDs))
	if len(documentIDs) == 0 {
		return result, nil
	}

	rows, err := db.Query(`
		SELECT id, document_id, resolution, resolution_author, resolution_executors, position
		FROM (
			SELECT id, document_id, resolution, resolution_author, resolution_executors, position,
				ROW_NUMBER() OVER (PARTITION BY document_id ORDER BY position, created_at, id) AS rn
			FROM document_resolutions
			WHERE document_id = ANY($1)
		) ranked
		WHERE rn = 1
		ORDER BY document_id, position, id
	`, pq.Array(documentIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to batch load first document resolutions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.DocumentResolution{}
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.Resolution,
			&item.ResolutionAuthor, &item.ResolutionExecutors, &item.Position,
		); err != nil {
			return nil, fmt.Errorf("scan first document resolution error: %w", err)
		}
		result[item.DocumentID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("first document resolutions rows error: %w", err)
	}

	return result, nil
}

func applyResolution(doc *models.IncomingDocument, resolution *models.DocumentResolution) {
	if doc == nil || resolution == nil {
		return
	}
	doc.Resolution = resolution.Resolution
	doc.ResolutionAuthor = resolution.ResolutionAuthor
	doc.ResolutionExecutors = resolution.ResolutionExecutors
}

func (r *IncomingDocumentRepository) loadResolutions(documentID uuid.UUID) ([]models.DocumentResolution, error) {
	return loadDocumentResolutions(r.db, documentID)
}

func loadDocumentResolutions(db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}, documentID uuid.UUID) ([]models.DocumentResolution, error) {
	rows, err := db.Query(`
		SELECT id, document_id, resolution, resolution_author, resolution_executors, position
		FROM document_resolutions
		WHERE document_id = $1
		ORDER BY position, created_at, id
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document resolutions: %w", err)
	}
	defer rows.Close()

	items := make([]models.DocumentResolution, 0)
	for rows.Next() {
		var item models.DocumentResolution
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.Resolution,
			&item.ResolutionAuthor, &item.ResolutionExecutors, &item.Position,
		); err != nil {
			return nil, fmt.Errorf("scan document resolution error: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document resolutions rows error: %w", err)
	}

	return items, nil
}

func loadDocumentResolutionsByDocumentIDs(db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}, documentIDs []uuid.UUID) (map[uuid.UUID][]models.DocumentResolution, error) {
	result := make(map[uuid.UUID][]models.DocumentResolution, len(documentIDs))
	if len(documentIDs) == 0 {
		return result, nil
	}

	rows, err := db.Query(`
		SELECT id, document_id, resolution, resolution_author, resolution_executors, position
		FROM document_resolutions
		WHERE document_id = ANY($1)
		ORDER BY document_id, position, created_at, id
	`, pq.Array(documentIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to batch load document resolutions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.DocumentResolution
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.Resolution,
			&item.ResolutionAuthor, &item.ResolutionExecutors, &item.Position,
		); err != nil {
			return nil, fmt.Errorf("scan document resolution error: %w", err)
		}
		result[item.DocumentID] = append(result[item.DocumentID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document resolutions rows error: %w", err)
	}

	return result, nil
}

func (r *IncomingDocumentRepository) loadCorrespondents(documentID uuid.UUID) ([]models.DocumentCorrespondentRegistration, error) {
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

func loadDocumentCorrespondentsByDocumentIDs(db interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}, documentIDs []uuid.UUID) (map[uuid.UUID][]models.DocumentCorrespondentRegistration, error) {
	result := make(map[uuid.UUID][]models.DocumentCorrespondentRegistration, len(documentIDs))
	if len(documentIDs) == 0 {
		return result, nil
	}

	rows, err := db.Query(`
		SELECT cr.id, cr.document_id, cr.registration_number, cr.registration_date,
			cr.correspondent_org_id, o.name, cr.position
		FROM document_correspondent_registrations cr
		JOIN organizations o ON o.id = cr.correspondent_org_id
		WHERE cr.document_id = ANY($1)
		ORDER BY cr.document_id, cr.position, cr.created_at, cr.id
	`, pq.Array(documentIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to batch load document correspondents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.DocumentCorrespondentRegistration
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.RegistrationNumber, &item.RegistrationDate,
			&item.CorrespondentOrgID, &item.CorrespondentName, &item.Position,
		); err != nil {
			return nil, fmt.Errorf("scan document correspondent error: %w", err)
		}
		result[item.DocumentID] = append(result[item.DocumentID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document correspondents rows error: %w", err)
	}

	return result, nil
}

func replaceCorrespondents(tx *sql.Tx, documentID uuid.UUID, items []models.DocumentCorrespondentRegistration) error {
	if _, err := tx.Exec(`DELETE FROM document_correspondent_registrations WHERE document_id = $1`, documentID); err != nil {
		return fmt.Errorf("failed to clear document correspondents: %w", err)
	}

	for i, item := range items {
		position := item.Position
		if position <= 0 {
			position = i + 1
		}
		if _, err := tx.Exec(`
			INSERT INTO document_correspondent_registrations (
				document_id, registration_number, registration_date, correspondent_org_id, position
			) VALUES ($1, $2, $3, $4, $5)
		`, documentID, item.RegistrationNumber, item.RegistrationDate, item.CorrespondentOrgID, position); err != nil {
			return fmt.Errorf("failed to save document correspondent: %w", err)
		}
	}

	return nil
}

func replaceResolution(tx *sql.Tx, documentID uuid.UUID, resolution, author, executors *string) error {
	if resolution == nil && author == nil && executors == nil {
		return replaceResolutions(tx, documentID, nil)
	}

	return replaceResolutions(tx, documentID, []models.DocumentResolution{{
		Resolution:          resolution,
		ResolutionAuthor:    author,
		ResolutionExecutors: executors,
		Position:            1,
	}})
}

func replaceResolutions(tx *sql.Tx, documentID uuid.UUID, items []models.DocumentResolution) error {
	if _, err := tx.Exec(`DELETE FROM document_resolutions WHERE document_id = $1`, documentID); err != nil {
		return fmt.Errorf("failed to clear document resolutions: %w", err)
	}

	for i, item := range items {
		position := item.Position
		if position <= 0 {
			position = i + 1
		}
		if _, err := tx.Exec(`
		INSERT INTO document_resolutions (
			document_id, resolution, resolution_author, resolution_executors, position
		) VALUES ($1, $2, $3, $4, $5)
	`, documentID, item.Resolution, item.ResolutionAuthor, item.ResolutionExecutors, position); err != nil {
			return fmt.Errorf("failed to save document resolution: %w", err)
		}
	}

	return nil
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
		where = append(where, fmt.Sprintf("d.document_type = $%d", argIdx))
		args = append(args, filter.DocumentTypeID)
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
		where = append(where, fmt.Sprintf(`(
			d.content ILIKE $%d
			OR inc.incoming_number ILIKE $%d
			OR EXISTS (
				SELECT 1
				FROM document_correspondent_registrations cr
				WHERE cr.document_id = d.id AND cr.registration_number ILIKE $%d
			)
		)`, argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.IncomingNumber != "" {
		where = append(where, fmt.Sprintf("inc.incoming_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.IncomingNumber+"%")
		argIdx++
	}
	if filter.OutgoingNumber != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM document_correspondent_registrations cr
			WHERE cr.document_id = d.id AND cr.registration_number ILIKE $%d
		)`, argIdx))
		args = append(args, "%"+filter.OutgoingNumber+"%")
		argIdx++
	}
	if filter.SenderName != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM document_correspondent_registrations cr
			JOIN organizations o ON o.id = cr.correspondent_org_id
			WHERE cr.document_id = d.id AND o.name ILIKE $%d
		)`, argIdx))
		args = append(args, "%"+filter.SenderName+"%")
		argIdx++
	}
	if filter.OutgoingDateFrom != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM document_correspondent_registrations cr
			WHERE cr.document_id = d.id AND cr.registration_date >= $%d
		)`, argIdx))
		args = append(args, filter.OutgoingDateFrom)
		argIdx++
	}
	if filter.OutgoingDateTo != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM document_correspondent_registrations cr
			WHERE cr.document_id = d.id AND cr.registration_date <= $%d
		)`, argIdx))
		args = append(args, filter.OutgoingDateTo)
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
	documentIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		doc, err := scanIncomingDoc(rows)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		documentIDs = append(documentIDs, doc.ID)
		items = append(items, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	if len(documentIDs) > 0 {
		correspondentsByDocumentID, err := loadDocumentCorrespondentsByDocumentIDs(r.db, documentIDs)
		if err != nil {
			return nil, err
		}
		resolutionsByDocumentID, err := loadFirstDocumentResolutionsByDocumentIDs(r.db, documentIDs)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].Correspondents = correspondentsByDocumentID[items[i].ID]
			applyResolution(&items[i], resolutionsByDocumentID[items[i].ID])
		}
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

	doc.Correspondents, err = r.loadCorrespondents(doc.ID)
	if err != nil {
		return nil, err
	}
	resolution, err := r.loadResolution(doc.ID)
	if err != nil {
		return nil, err
	}
	applyResolution(doc, resolution)

	return doc, nil
}

// Create создает новый входящий документ в базе данных.
func (r *IncomingDocumentRepository) Create(req models.CreateIncomingDocRequest) (*models.IncomingDocument, error) {
	req.DocumentTypeID = models.NormalizeDocumentType(req.DocumentTypeID)
	if !models.IsAllowedDocumentType(req.DocumentTypeID) {
		return nil, models.NewBadRequest("неверный тип документа")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	registration, err := resolveRegistrationNumberTx(tx, req.CreatedBy, models.DocumentKindIncomingLetter, req.NomenclatureID, req.IdempotencyKey, req.IncomingNumber)
	if err != nil {
		return nil, err
	}
	if registration.Existing != uuid.Nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit idempotent transaction: %w", err)
		}
		return r.GetByID(registration.Existing)
	}
	req.IncomingNumber = registration.Number

	var id uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO documents (
			kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`,
		models.DocumentKindIncomingLetter, req.NomenclatureID, req.IdempotencyKey, req.IncomingNumber, req.IncomingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err, "idx_documents_created_by_kind_idempotency") {
			_ = tx.Rollback()
			existingID, lookupErr := findExistingDocumentIDByIdempotency(r.db, req.CreatedBy, models.DocumentKindIncomingLetter, req.IdempotencyKey)
			if lookupErr != nil {
				return nil, fmt.Errorf("failed to resolve idempotent document: %w", lookupErr)
			}
			return r.GetByID(existingID)
		}
		if isUniqueViolation(err, "idx_documents_kind_registration_number_year") {
			return nil, models.NewConflict("документ с таким регистрационным номером уже существует")
		}
		return nil, fmt.Errorf("failed to create document root: %w", err)
	}

	if _, err = tx.Exec(`
		INSERT INTO incoming_document_details (
			document_id, incoming_number, incoming_date,
			sender_signatory
		) VALUES ($1,$2,$3,$4)
	`,
		id, req.IncomingNumber, req.IncomingDate,
		req.SenderSignatory,
	); err != nil {
		return nil, fmt.Errorf("failed to create incoming document details: %w", err)
	}

	if err := replaceCorrespondents(tx, id, req.Correspondents); err != nil {
		return nil, err
	}
	if err := replaceResolution(tx, id, req.Resolution, req.ResolutionAuthor, req.ResolutionExecutors); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(id)
}

// Update обновляет данные существующего входящего документа.
func (r *IncomingDocumentRepository) Update(req models.UpdateIncomingDocRequest) (*models.IncomingDocument, error) {
	req.DocumentTypeID = models.NormalizeDocumentType(req.DocumentTypeID)
	if !models.IsAllowedDocumentType(req.DocumentTypeID) {
		return nil, models.NewBadRequest("неверный тип документа")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`
		UPDATE documents SET
			document_type = $1,
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
			sender_signatory = $1
		WHERE document_id = $2
	`,
		req.SenderSignatory,
		req.ID,
	); err != nil {
		return nil, fmt.Errorf("failed to update incoming document details: %w", err)
	}

	if err := replaceCorrespondents(tx, req.ID, req.Correspondents); err != nil {
		return nil, err
	}
	if err := replaceResolution(tx, req.ID, req.Resolution, req.ResolutionAuthor, req.ResolutionExecutors); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(req.ID)
}

// GetCount — общее количество для дашборда
func (r *IncomingDocumentRepository) GetCount() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE kind = $1`, models.DocumentKindIncomingLetter).Scan(&count)
	return count, err
}
