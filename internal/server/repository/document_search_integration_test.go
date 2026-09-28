package repository

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDocumentSearchIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	user := insertIntegrationUser(t, db, "search-reader")
	org := insertOrganization(t, db, "Администрация города")
	repo := NewDocumentRepository(database.Wrap(db))
	scopes := map[models.DocumentKind]models.DocumentAccessScope{}
	for _, spec := range models.AllDocumentKindSpecs() {
		scopes[spec.Code] = models.DocumentAccessScope{}
	}
	noms := map[uuid.UUID]uuid.UUID{}
	insert := func(kind models.DocumentKind, content string) uuid.UUID {
		id, nom := uuid.New(), uuid.New()
		noms[id] = nom
		execSQL(t, db, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
            VALUES ($1, 'Search', $2, 2026, $3, '/', 'index_and_number')`, nom, id.String(), kind)
		execSQL(t, db, `INSERT INTO documents (id, kind, nomenclature_id, registration_number, registration_date, document_type, content, created_by, created_at)
            VALUES ($1, $2, $3, $4, '2026-09-28', 'Письмо', $5, $6, '2026-09-28T12:00:00Z')`, id, kind, nom, id.String(), content, user)
		return id
	}
	incoming := insert(models.DocumentKindIncomingLetter, "О ремонте дороги и ёлке")
	execSQL(t, db, `INSERT INTO incoming_document_details (document_id, incoming_number, incoming_date, sender_signatory)
        VALUES ($1, '123', CURRENT_DATE, 'Иванов И.И.')`, incoming)
	execSQL(t, db, `INSERT INTO document_correspondent_registrations (document_id, registration_number, registration_date, correspondent_org_id, position)
        VALUES ($1, '1', CURRENT_DATE, $2, 1), ($1, '2', CURRENT_DATE, $2, 2)`, incoming, org)
	execSQL(t, db, `INSERT INTO document_resolutions (document_id, resolution, position)
        VALUES ($1, 'Первичная резолюция', 1), ($1, 'Подготовить ответ', 2)`, incoming)
	outgoing := insert(models.DocumentKindOutgoingLetter, "Согласование проекта")
	execSQL(t, db, `INSERT INTO outgoing_document_details (document_id, outgoing_number, outgoing_date, sender_signatory, sender_executor, recipient_org_id, addressee)
        VALUES ($1, '456', CURRENT_DATE, 'Сидоров', 'Неиндексируемый', $2, 'Петров')`, outgoing, org)
	order := insert(models.DocumentKindAdministrativeOrder, "Скрытый старый текст")
	execSQL(t, db, `INSERT INTO administrative_order_details (document_id, order_number, order_date, title)
        VALUES ($1, '789', CURRENT_DATE, 'О ремонте здания')`, order)
	appeal := insert(models.DocumentKindCitizenAppeal, "Обращение гражданина")
	execSQL(t, db, `INSERT INTO document_resolutions (document_id, resolution) VALUES ($1, 'Проверить ремонт')`, appeal)
	search := func(query string, page, size int, access map[models.DocumentKind]models.DocumentAccessScope) *dto.DocumentSearchResult {
		result, err := repo.SearchDocuments(dto.DocumentSearchRequest{Query: query, Page: page, PageSize: size}, access)
		require.NoError(t, err)
		return result
	}
	ids := func(result *dto.DocumentSearchResult) []string {
		values := []string{}
		for _, item := range result.Items {
			values = append(values, item.ID)
		}
		return values
	}
	result := search("ремонт администрация Иванов ответ", 1, 20, scopes)
	require.Equal(t, []string{incoming.String()}, ids(result))
	require.Contains(t, result.Items[0].Resolution, "Подготовить ответ")
	require.Contains(t, result.Items[0].Correspondent, "Администрация")
	require.Equal(t, 1, result.TotalCount) // Multiple correspondent/resolution rows never duplicate a document.
	require.Equal(t, []string{outgoing.String()}, ids(search("Сидоров Петров администрация", 1, 20, scopes)))
	require.Empty(t, search("Неиндексируемый", 1, 20, scopes).Items)
	require.Equal(t, []string{incoming.String()}, ids(search("елка", 1, 20, scopes)))
	require.Equal(t, []string{incoming.String()}, ids(search("и", 1, 20, scopes)))
	require.Equal(t, []string{order.String()}, ids(search("здание ремонт", 1, 20, scopes)))
	require.Empty(t, search("Скрытый старый", 1, 20, scopes).Items)
	require.Equal(t, []string{appeal.String()}, ids(search("проверить", 1, 20, scopes)))
	result = search("ремонт", 1, 20, scopes)
	require.Equal(t, 3, result.TotalCount)
	require.Equal(t, appeal.String(), result.Items[2].ID) // Content/title outweigh a resolution.
	require.Greater(t, result.Items[0].Relevance, result.Items[2].Relevance)
	first := search("ремонт", 1, 1, scopes)
	second := search("ремонт", 2, 1, scopes)
	require.Equal(t, result.Items[0].ID, first.Items[0].ID)
	require.Equal(t, result.Items[1].ID, second.Items[0].ID)
	beyond := search("ремонт", 10, 1, scopes)
	require.Empty(t, beyond.Items)
	require.Equal(t, 3, beyond.TotalCount)
	restricted := map[models.DocumentKind]models.DocumentAccessScope{
		models.DocumentKindCitizenAppeal:       {}, // This lower ranked document is the only readable one.
		models.DocumentKindIncomingLetter:      {Restricted: true},
		models.DocumentKindAdministrativeOrder: {Restricted: true},
	}
	result = search("ремонт", 1, 1, restricted)
	require.Equal(t, []string{appeal.String()}, ids(result))
	require.Equal(t, 1, result.TotalCount)
	restricted[models.DocumentKindIncomingLetter] = models.DocumentAccessScope{Restricted: true, AllowedNomenclatureIDs: []string{noms[incoming].String()}}
	require.Equal(t, 2, search("ремонт", 1, 20, restricted).TotalCount)
	// User/substitution assignment edges use the same scope policy as journals.
	delegate := insertIntegrationUser(t, db, "search-delegate")
	execSQL(t, db, `INSERT INTO assignments (document_id, executor_id, content, status) VALUES ($1, $2, 'Read', 'new')`, order, delegate)
	restricted[models.DocumentKindAdministrativeOrder] = models.DocumentAccessScope{Restricted: true, AccessibleByUserIDs: []string{user.String(), delegate.String()}}
	require.Equal(t, 3, search("ремонт", 1, 20, restricted).TotalCount)
	require.Empty(t, search("ремонт", 1, 20, nil).Items)
	require.Empty(t, search("отсутствующий термин", 1, 20, scopes).Items)
	// Updates and organization renames are visible without asynchronous rebuilding.
	execSQL(t, db, `UPDATE organizations SET name = 'Муниципалитет' WHERE id = $1`, org)
	require.Empty(t, search("администрация", 1, 20, scopes).Items)
	require.Equal(t, 2, search("муниципалитет", 1, 20, scopes).TotalCount)
	execSQL(t, db, `UPDATE document_resolutions SET resolution = 'Обновленная резолюция' WHERE document_id = $1`, appeal)
	require.Equal(t, []string{appeal.String()}, ids(search("обновленная", 1, 20, scopes)))
	require.Empty(t, search("% _", 1, 20, scopes).Items)
}
