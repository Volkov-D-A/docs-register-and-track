package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// AdministrativeOrderRepository предоставляет методы для работы с приказами в БД.
type AdministrativeOrderRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *AdministrativeOrderRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

// NewAdministrativeOrderRepository создает новый экземпляр AdministrativeOrderRepository.
func NewAdministrativeOrderRepository(db *database.DB) *AdministrativeOrderRepository {
	return &AdministrativeOrderRepository{db: db}
}

// GetList возвращает список приказов с учетом фильтрации и пагинации.
func (r *AdministrativeOrderRepository) GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error) {
	where := []string{"d.kind = $1"}
	args := []interface{}{models.DocumentKindAdministrativeOrder}
	argIdx := 2

	scope := documentListAccessScope(filter.AccessScope, filter.AllowedNomenclatureIDs, filter.AccessibleByUserID, filter.AccessibleByUserIDs)
	applyDocumentListAccess(&where, &args, &argIdx, scope)

	if len(filter.NomenclatureIDs) > 0 {
		where = append(where, fmt.Sprintf("d.nomenclature_id = ANY($%d)", argIdx))
		args = append(args, pq.Array(filter.NomenclatureIDs))
		argIdx++
	}
	if filter.DateFrom != "" {
		where = append(where, fmt.Sprintf("ord.order_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != "" {
		where = append(where, fmt.Sprintf("ord.order_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(ord.title ILIKE $%d OR ord.order_number ILIKE $%d OR ord.execution_controller ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.OrderNumber != "" {
		where = append(where, fmt.Sprintf("ord.order_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.OrderNumber+"%")
		argIdx++
	}
	if filter.ExecutionController != "" {
		where = append(where, fmt.Sprintf("ord.execution_controller ILIKE $%d", argIdx))
		args = append(args, "%"+filter.ExecutionController+"%")
		argIdx++
	}
	if filter.OnlyPendingAcknowledgment {
		where = append(where, `EXISTS (
			SELECT 1
			FROM administrative_order_acknowledgment_people p
			WHERE p.document_id = d.id AND p.acknowledged_at IS NULL
		)`)
	}
	switch filter.OrderActiveStatus {
	case "active":
		where = append(where, "ord.is_active = true")
	case "inactive":
		where = append(where, "ord.is_active = false")
	}

	whereClause := strings.Join(where, " AND ")
	var totalCount int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM documents d
		JOIN administrative_order_details ord ON ord.document_id = d.id
		WHERE %s
	`, whereClause)
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count administrative orders: %w", err)
	}

	filter.Page, filter.PageSize = normalizePagination(filter.Page, filter.PageSize)
	offset := (filter.Page - 1) * filter.PageSize

	query := fmt.Sprintf(`
		SELECT
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			ord.order_number, ord.order_date, ord.title,
			ord.execution_controller, ord.execution_deadline, ord.is_active, ord.cancelled_at,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM documents d
		JOIN administrative_order_details ord ON ord.document_id = d.id
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN users u ON d.created_by = u.id
		WHERE %s
		ORDER BY d.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get administrative orders: %w", err)
	}
	defer rows.Close()

	items := make([]models.AdministrativeOrderDocument, 0)
	documentIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		doc, err := scanAdministrativeOrder(rows)
		if err != nil {
			return nil, err
		}
		documentIDs = append(documentIDs, doc.ID)
		items = append(items, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(documentIDs) > 0 {
		peopleByDocumentID, err := r.getAcknowledgmentPeopleByDocumentIDs(documentIDs)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].AcknowledgmentPeople = peopleByDocumentID[items[i].ID]
		}
	}

	return &models.PagedResult[models.AdministrativeOrderDocument]{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

// GetByID возвращает приказ по ID.
func (r *AdministrativeOrderRepository) GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error) {
	row := r.db.QueryRow(`
		SELECT
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			ord.order_number, ord.order_date, ord.title,
			ord.execution_controller, ord.execution_deadline, ord.is_active, ord.cancelled_at,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM documents d
		JOIN administrative_order_details ord ON ord.document_id = d.id
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN users u ON d.created_by = u.id
		WHERE d.id = $1 AND d.kind = $2
	`, id, models.DocumentKindAdministrativeOrder)

	doc, err := scanAdministrativeOrder(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get administrative order: %w", err)
	}

	people, err := r.GetAcknowledgmentPeople(id)
	if err != nil {
		return nil, err
	}
	doc.AcknowledgmentPeople = people
	return doc, nil
}

// Create создает приказ.
func (r *AdministrativeOrderRepository) Create(req models.CreateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return r.create(req, nil, "", "")
}
func (r *AdministrativeOrderRepository) CreateWithJournal(req models.CreateAdministrativeOrderDocRequest, action, detailsFormat string) (*models.AdministrativeOrderDocument, error) {
	return r.create(req, nil, action, detailsFormat)
}
func (r *AdministrativeOrderRepository) create(req models.CreateAdministrativeOrderDocRequest, effects []models.OutboxEvent, journalAction, journalDetailsFormat string) (*models.AdministrativeOrderDocument, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var registration *registrationNumberResult
	if req.AdminNumberOverride != nil {
		registration, err = resolveAdminRegistrationNumberTx(tx, req.CreatedBy, models.DocumentKindAdministrativeOrder, req.NomenclatureID, req.IdempotencyKey, req.AdminNumberOverride)
	} else {
		registration, err = resolveRegistrationNumberTx(tx, req.CreatedBy, models.DocumentKindAdministrativeOrder, req.NomenclatureID, req.IdempotencyKey, req.OrderNumber)
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
	req.OrderNumber = registration.Number

	var id uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO documents (
			kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8)
		RETURNING id
	`,
		models.DocumentKindAdministrativeOrder,
		req.NomenclatureID,
		req.IdempotencyKey,
		req.OrderNumber,
		req.OrderDate,
		models.DocumentTypeAdministrativeOrder,
		req.Title,
		req.CreatedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err, "idx_documents_created_by_kind_idempotency") {
			_ = tx.Rollback()
			existingID, lookupErr := findExistingDocumentIDByIdempotency(r.db, req.CreatedBy, models.DocumentKindAdministrativeOrder, req.IdempotencyKey)
			if lookupErr != nil {
				return nil, fmt.Errorf("failed to resolve idempotent document: %w", lookupErr)
			}
			return r.GetByID(existingID)
		}
		if isUniqueViolation(err, "idx_documents_kind_registration_number_year") {
			return nil, models.NewConflict("документ с таким регистрационным номером уже существует")
		}
		return nil, fmt.Errorf("failed to create administrative order root: %w", err)
	}

	if _, err = tx.Exec(`
		INSERT INTO administrative_order_details (
			document_id, order_number, order_date, title,
			execution_controller, execution_deadline, is_active, cancelled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		id, req.OrderNumber, req.OrderDate, req.Title,
		req.ExecutionController, req.ExecutionDeadline, req.IsActive, req.CancelledAt,
	); err != nil {
		return nil, fmt.Errorf("failed to create administrative order details: %w", err)
	}

	if err := replaceAdministrativeOrderAcknowledgmentPeopleTx(tx, id, req.AcknowledgmentFullNames, nil); err != nil {
		return nil, err
	}
	if journalAction != "" {
		if r.outbox == nil {
			return nil, fmt.Errorf("outbox repository is required for document journal")
		}
		payload := fmt.Sprintf(`{"documentId":"%s","userId":"%s","action":%q,"details":%q}`, id, req.CreatedBy, journalAction, fmt.Sprintf(journalDetailsFormat, req.OrderNumber))
		if err := r.outbox.EnqueueTx(tx, models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "administrative-order:" + id.String() + ":create:journal", Payload: payload}); err != nil {
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

// Update обновляет приказ.
func (r *AdministrativeOrderRepository) Update(req models.UpdateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return r.update(req, nil)
}
func (r *AdministrativeOrderRepository) UpdateWithOutbox(req models.UpdateAdministrativeOrderDocRequest, effects []models.OutboxEvent) (*models.AdministrativeOrderDocument, error) {
	return r.update(req, effects)
}
func (r *AdministrativeOrderRepository) update(req models.UpdateAdministrativeOrderDocRequest, effects []models.OutboxEvent) (*models.AdministrativeOrderDocument, error) {
	existingPeople, err := r.GetAcknowledgmentPeople(req.ID)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`
		UPDATE documents SET
			registration_date = $1,
			content = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND kind = $4
	`, req.OrderDate, req.Title, req.ID, models.DocumentKindAdministrativeOrder); err != nil {
		return nil, fmt.Errorf("failed to update administrative order root: %w", err)
	}

	if _, err = tx.Exec(`
		UPDATE administrative_order_details SET
			order_date = $1,
			title = $2,
			execution_controller = $3,
			execution_deadline = $4,
			is_active = $5,
			cancelled_at = $6
		WHERE document_id = $7
	`, req.OrderDate, req.Title, req.ExecutionController, req.ExecutionDeadline, req.IsActive, req.CancelledAt, req.ID); err != nil {
		return nil, fmt.Errorf("failed to update administrative order details: %w", err)
	}

	if err := replaceAdministrativeOrderAcknowledgmentPeopleTx(tx, req.ID, req.AcknowledgmentFullNames, existingPeople); err != nil {
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

// MarkAcknowledgmentPerson подтверждает ознакомление строки приказа.
func (r *AdministrativeOrderRepository) MarkAcknowledgmentPerson(id uuid.UUID, acknowledgedBy uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	var documentID uuid.UUID
	var acknowledgedAt time.Time
	err := r.db.QueryRow(`
		UPDATE administrative_order_acknowledgment_people
		SET acknowledged_at = COALESCE(acknowledged_at, CURRENT_TIMESTAMP),
			acknowledged_by = COALESCE(acknowledged_by, $2)
		WHERE id = $1
		RETURNING document_id, acknowledged_at
	`, id, acknowledgedBy).Scan(&documentID, &acknowledgedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to mark administrative order acknowledgment: %w", err)
	}

	person, err := r.GetAcknowledgmentPersonByID(id)
	if err != nil {
		return nil, err
	}
	if person != nil && person.AcknowledgedAt == nil {
		person.AcknowledgedAt = &acknowledgedAt
		person.AcknowledgedBy = &acknowledgedBy
		person.DocumentID = documentID
	}
	return person, nil
}

// MarkAcknowledgmentPersonWithOutbox stores the acknowledgement and its
// document-journal event in the same transaction.
func (r *AdministrativeOrderRepository) MarkAcknowledgmentPersonWithOutbox(id, acknowledgedBy uuid.UUID, effects []models.OutboxEvent) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var documentID uuid.UUID
	if err := tx.QueryRow(`UPDATE administrative_order_acknowledgment_people SET acknowledged_at = COALESCE(acknowledged_at, CURRENT_TIMESTAMP), acknowledged_by = COALESCE(acknowledged_by, $2) WHERE id = $1 RETURNING document_id`, id, acknowledgedBy).Scan(&documentID); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to mark administrative order acknowledgment: %w", err)
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAcknowledgmentPersonByID(id)
}

// GetAcknowledgmentPersonByID возвращает строку листа ознакомления по ID.
func (r *AdministrativeOrderRepository) GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	row := r.db.QueryRow(`
		SELECT
			p.id, p.document_id, p.full_name, p.acknowledged_at, p.acknowledged_by,
			COALESCE(NULLIF(u.full_name, ''), u.login, '') AS acknowledged_by_name,
			p.position, p.created_at
		FROM administrative_order_acknowledgment_people p
		LEFT JOIN users u ON u.id = p.acknowledged_by
		WHERE p.id = $1
	`, id)

	person, err := scanAdministrativeOrderAcknowledgmentPerson(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get administrative order acknowledgment person: %w", err)
	}
	return person, nil
}

// GetAcknowledgmentPeople возвращает лист ознакомления приказа.
func (r *AdministrativeOrderRepository) GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error) {
	rows, err := r.db.Query(`
		SELECT
			p.id, p.document_id, p.full_name, p.acknowledged_at, p.acknowledged_by,
			COALESCE(NULLIF(u.full_name, ''), u.login, '') AS acknowledged_by_name,
			p.position, p.created_at
		FROM administrative_order_acknowledgment_people p
		LEFT JOIN users u ON u.id = p.acknowledged_by
		WHERE p.document_id = $1
		ORDER BY p.position, p.created_at, p.full_name
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get administrative order acknowledgment people: %w", err)
	}
	defer rows.Close()

	result := make([]models.AdministrativeOrderAcknowledgmentPerson, 0)
	for rows.Next() {
		person, err := scanAdministrativeOrderAcknowledgmentPerson(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *person)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AdministrativeOrderRepository) getAcknowledgmentPeopleByDocumentIDs(documentIDs []uuid.UUID) (map[uuid.UUID][]models.AdministrativeOrderAcknowledgmentPerson, error) {
	result := make(map[uuid.UUID][]models.AdministrativeOrderAcknowledgmentPerson, len(documentIDs))
	if len(documentIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.Query(`
		SELECT
			p.id, p.document_id, p.full_name, p.acknowledged_at, p.acknowledged_by,
			COALESCE(NULLIF(u.full_name, ''), u.login, '') AS acknowledged_by_name,
			p.position, p.created_at
		FROM administrative_order_acknowledgment_people p
		LEFT JOIN users u ON u.id = p.acknowledged_by
		WHERE p.document_id = ANY($1)
		ORDER BY p.document_id, p.position, p.created_at, p.full_name
	`, pq.Array(documentIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to batch load administrative order acknowledgment people: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		person, err := scanAdministrativeOrderAcknowledgmentPerson(rows)
		if err != nil {
			return nil, err
		}
		result[person.DocumentID] = append(result[person.DocumentID], *person)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// CancelByLink помечает приказ недействующим при создании отменяющей связи.
func (r *AdministrativeOrderRepository) CancelByLink(id uuid.UUID, cancelledAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE documents d
		SET updated_at = CURRENT_TIMESTAMP
		FROM administrative_order_details ord
		WHERE ord.document_id = d.id
		  AND d.id = $1
		  AND d.kind = $2
		  AND (
			ord.is_active = true
			OR ord.cancelled_at IS DISTINCT FROM $3
		  )
	`, id, models.DocumentKindAdministrativeOrder, cancelledAt)
	if err != nil {
		return fmt.Errorf("failed to update administrative order root cancellation timestamp: %w", err)
	}

	_, err = r.db.Exec(`
		UPDATE administrative_order_details
		SET is_active = false,
			cancelled_at = $2
		WHERE document_id = $1
	`, id, cancelledAt)
	if err != nil {
		return fmt.Errorf("failed to cancel administrative order by link: %w", err)
	}
	return nil
}

// GetCount возвращает количество приказов.
func (r *AdministrativeOrderRepository) GetCount() (int, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE kind = $1`, models.DocumentKindAdministrativeOrder).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type administrativeOrderScanner interface {
	Scan(dest ...interface{}) error
}

func scanAdministrativeOrder(scanner administrativeOrderScanner) (*models.AdministrativeOrderDocument, error) {
	var doc models.AdministrativeOrderDocument
	var deadline sql.NullTime
	var cancelledAt sql.NullTime
	if err := scanner.Scan(
		&doc.ID, &doc.NomenclatureID, &doc.NomenclatureName,
		&doc.OrderNumber, &doc.OrderDate, &doc.Title,
		&doc.ExecutionController, &deadline, &doc.IsActive, &cancelledAt,
		&doc.CreatedBy, &doc.CreatedByName,
		&doc.CreatedAt, &doc.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if deadline.Valid {
		doc.ExecutionDeadline = &deadline.Time
	}
	if cancelledAt.Valid {
		doc.CancelledAt = &cancelledAt.Time
	}
	return &doc, nil
}

func scanAdministrativeOrderAcknowledgmentPerson(scanner administrativeOrderScanner) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	var person models.AdministrativeOrderAcknowledgmentPerson
	var acknowledgedAt sql.NullTime
	var acknowledgedBy uuid.NullUUID
	if err := scanner.Scan(
		&person.ID, &person.DocumentID, &person.FullName, &acknowledgedAt, &acknowledgedBy,
		&person.AcknowledgedByName, &person.Position, &person.CreatedAt,
	); err != nil {
		return nil, err
	}
	if acknowledgedAt.Valid {
		person.AcknowledgedAt = &acknowledgedAt.Time
	}
	if acknowledgedBy.Valid {
		person.AcknowledgedBy = &acknowledgedBy.UUID
	}
	return &person, nil
}

func replaceAdministrativeOrderAcknowledgmentPeopleTx(
	tx *sql.Tx,
	documentID uuid.UUID,
	names []string,
	existing []models.AdministrativeOrderAcknowledgmentPerson,
) error {
	preserved := make(map[string][]models.AdministrativeOrderAcknowledgmentPerson)
	for _, item := range existing {
		key := strings.ToLower(strings.TrimSpace(item.FullName))
		preserved[key] = append(preserved[key], item)
	}

	nameCounts := make(map[string]int)
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		nameCounts[strings.ToLower(name)]++
	}
	for _, item := range existing {
		if item.AcknowledgedAt == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(item.FullName))
		if nameCounts[key] == 0 {
			return models.NewBadRequest(fmt.Sprintf("нельзя удалить ФИО уже ознакомленного: %s", item.FullName))
		}
		nameCounts[key]--
	}

	if _, err := tx.Exec(`DELETE FROM administrative_order_acknowledgment_people WHERE document_id = $1`, documentID); err != nil {
		return fmt.Errorf("failed to clear administrative order acknowledgment people: %w", err)
	}

	position := 1
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		var acknowledgedAt *time.Time
		var acknowledgedBy *uuid.UUID
		key := strings.ToLower(name)
		if items := preserved[key]; len(items) > 0 {
			item := items[0]
			preserved[key] = items[1:]
			if item.AcknowledgedAt != nil {
				name = item.FullName
			}
			acknowledgedAt = item.AcknowledgedAt
			acknowledgedBy = item.AcknowledgedBy
		}

		if _, err := tx.Exec(`
			INSERT INTO administrative_order_acknowledgment_people (
				document_id, full_name, acknowledged_at, acknowledged_by, position
			) VALUES ($1, $2, $3, $4, $5)
		`, documentID, name, acknowledgedAt, acknowledgedBy, position); err != nil {
			return fmt.Errorf("failed to insert administrative order acknowledgment person: %w", err)
		}
		position++
	}

	return nil
}
