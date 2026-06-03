package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type queryIncomingDocStore struct {
	doc        *models.IncomingDocument
	list       *models.PagedResult[models.IncomingDocument]
	err        error
	lastID     uuid.UUID
	lastFilter models.DocumentFilter
}

func (s *queryIncomingDocStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error) {
	s.lastFilter = filter
	return s.list, s.err
}

func (s *queryIncomingDocStore) GetByID(id uuid.UUID) (*models.IncomingDocument, error) {
	s.lastID = id
	return s.doc, s.err
}

func (s *queryIncomingDocStore) Create(req models.CreateIncomingDocRequest) (*models.IncomingDocument, error) {
	return nil, nil
}

func (s *queryIncomingDocStore) Update(req models.UpdateIncomingDocRequest) (*models.IncomingDocument, error) {
	return nil, nil
}

func (s *queryIncomingDocStore) GetCount() (int, error) {
	return 0, nil
}

type queryOutgoingDocStore struct {
	doc        *models.OutgoingDocument
	list       *models.PagedResult[models.OutgoingDocument]
	err        error
	lastID     uuid.UUID
	lastFilter models.OutgoingDocumentFilter
}

func (s *queryOutgoingDocStore) GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error) {
	s.lastFilter = filter
	return s.list, s.err
}

func (s *queryOutgoingDocStore) GetByID(id uuid.UUID) (*models.OutgoingDocument, error) {
	s.lastID = id
	return s.doc, s.err
}

func (s *queryOutgoingDocStore) Create(req models.CreateOutgoingDocRequest) (*models.OutgoingDocument, error) {
	return nil, nil
}

func (s *queryOutgoingDocStore) Update(req models.UpdateOutgoingDocRequest) (*models.OutgoingDocument, error) {
	return nil, nil
}

func (s *queryOutgoingDocStore) GetCount() (int, error) {
	return 0, nil
}

type queryCitizenAppealDocStore struct {
	doc        *models.CitizenAppealDocument
	list       *models.PagedResult[models.CitizenAppealDocument]
	err        error
	lastID     uuid.UUID
	lastFilter models.DocumentFilter
}

func (s *queryCitizenAppealDocStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.CitizenAppealDocument], error) {
	s.lastFilter = filter
	return s.list, s.err
}

func (s *queryCitizenAppealDocStore) GetByID(id uuid.UUID) (*models.CitizenAppealDocument, error) {
	s.lastID = id
	return s.doc, s.err
}

func (s *queryCitizenAppealDocStore) Create(req models.CreateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	return nil, nil
}

func (s *queryCitizenAppealDocStore) Update(req models.UpdateCitizenAppealDocRequest) (*models.CitizenAppealDocument, error) {
	return nil, nil
}

func (s *queryCitizenAppealDocStore) GetCount() (int, error) {
	return 0, nil
}

type queryAdministrativeOrderDocStore struct {
	doc        *models.AdministrativeOrderDocument
	list       *models.PagedResult[models.AdministrativeOrderDocument]
	err        error
	lastID     uuid.UUID
	lastFilter models.DocumentFilter
}

func (s *queryAdministrativeOrderDocStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error) {
	s.lastFilter = filter
	return s.list, s.err
}

func (s *queryAdministrativeOrderDocStore) GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error) {
	s.lastID = id
	return s.doc, s.err
}

func (s *queryAdministrativeOrderDocStore) Create(req models.CreateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *queryAdministrativeOrderDocStore) Update(req models.UpdateAdministrativeOrderDocRequest) (*models.AdministrativeOrderDocument, error) {
	return nil, nil
}

func (s *queryAdministrativeOrderDocStore) GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *queryAdministrativeOrderDocStore) GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *queryAdministrativeOrderDocStore) MarkAcknowledgmentPerson(id uuid.UUID, acknowledgedBy uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error) {
	return nil, nil
}

func (s *queryAdministrativeOrderDocStore) CancelByLink(id uuid.UUID, cancelledAt time.Time) error {
	return nil
}

func (s *queryAdministrativeOrderDocStore) GetCount() (int, error) {
	return 0, nil
}

func TestIncomingLetterQueryHandler(t *testing.T) {
	docID := uuid.New()
	now := time.Now()
	store := &queryIncomingDocStore{
		doc: &models.IncomingDocument{
			ID:             docID,
			IncomingNumber: "IN-1",
			IncomingDate:   now,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		list: &models.PagedResult[models.IncomingDocument]{
			Items:      []models.IncomingDocument{{ID: docID, IncomingNumber: "IN-1", IncomingDate: now}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		},
	}
	handler := NewIncomingLetterQueryHandler(store)

	assert.Equal(t, models.DocumentKindIncomingLetter, handler.Kind())

	card, err := handler.GetCard(docID)
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, docID.String(), card.ID)
	assert.NotNil(t, card.IncomingLetter)
	assert.Equal(t, docID, store.lastID)

	res, err := handler.GetList(models.DocumentFilter{Search: "IN", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.TotalCount)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "IN", store.lastFilter.Search)
	assert.Equal(t, string(models.DocumentKindIncomingLetter), res.Items[0].KindCode)
}

func TestOutgoingLetterQueryHandler(t *testing.T) {
	docID := uuid.New()
	now := time.Now()
	store := &queryOutgoingDocStore{
		doc: &models.OutgoingDocument{
			ID:             docID,
			OutgoingNumber: "OUT-1",
			OutgoingDate:   now,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		list: &models.PagedResult[models.OutgoingDocument]{
			Items:      []models.OutgoingDocument{{ID: docID, OutgoingNumber: "OUT-1", OutgoingDate: now}},
			TotalCount: 1,
			Page:       2,
			PageSize:   30,
		},
	}
	handler := NewOutgoingLetterQueryHandler(store)

	assert.Equal(t, models.DocumentKindOutgoingLetter, handler.Kind())

	card, err := handler.GetCard(docID)
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, docID.String(), card.ID)
	assert.NotNil(t, card.OutgoingLetter)

	filter := models.DocumentFilter{
		NomenclatureIDs:        []string{"nom-a"},
		AllowedNomenclatureIDs: []string{"nom-b"},
		AccessibleByUserID:     "user-1",
		DocumentTypeID:         "type-1",
		OrgID:                  "org-1",
		DateFrom:               "2026-01-01",
		DateTo:                 "2026-01-31",
		Search:                 "OUT",
		OutgoingNumber:         "OUT-1",
		RecipientName:          "Recipient",
		Page:                   2,
		PageSize:               30,
	}
	res, err := handler.GetList(filter)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.TotalCount)
	assert.Equal(t, string(models.DocumentKindOutgoingLetter), store.lastFilter.KindCode)
	assert.Equal(t, filter.NomenclatureIDs, store.lastFilter.NomenclatureIDs)
	assert.Equal(t, filter.AllowedNomenclatureIDs, store.lastFilter.AllowedNomenclatureIDs)
	assert.Equal(t, filter.AccessibleByUserID, store.lastFilter.AccessibleByUserID)
	assert.Equal(t, filter.RecipientName, store.lastFilter.RecipientName)
}

func TestCitizenAppealQueryHandler(t *testing.T) {
	docID := uuid.New()
	now := time.Now()
	store := &queryCitizenAppealDocStore{
		doc: &models.CitizenAppealDocument{
			ID:                 docID,
			RegistrationNumber: "CA-1",
			RegistrationDate:   now,
			AppealDate:         now,
			ApplicantFullName:  "Иван Иванов",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		list: &models.PagedResult[models.CitizenAppealDocument]{
			Items:      []models.CitizenAppealDocument{{ID: docID, RegistrationNumber: "CA-1", RegistrationDate: now, AppealDate: now}},
			TotalCount: 1,
			Page:       1,
			PageSize:   10,
		},
	}
	handler := NewCitizenAppealQueryHandler(store)

	assert.Equal(t, models.DocumentKindCitizenAppeal, handler.Kind())

	card, err := handler.GetCard(docID)
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.NotNil(t, card.CitizenAppeal)

	res, err := handler.GetList(models.DocumentFilter{ApplicantName: "Иван"})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "Иван", store.lastFilter.ApplicantName)
	assert.Equal(t, string(models.DocumentKindCitizenAppeal), res.Items[0].KindCode)
}

func TestAdministrativeOrderQueryHandler(t *testing.T) {
	docID := uuid.New()
	now := time.Now()
	store := &queryAdministrativeOrderDocStore{
		doc: &models.AdministrativeOrderDocument{
			ID:          docID,
			OrderNumber: "ORD-1",
			OrderDate:   now,
			Title:       "Приказ",
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		list: &models.PagedResult[models.AdministrativeOrderDocument]{
			Items:      []models.AdministrativeOrderDocument{{ID: docID, OrderNumber: "ORD-1", OrderDate: now, Title: "Приказ", IsActive: true}},
			TotalCount: 1,
			Page:       1,
			PageSize:   10,
		},
	}
	handler := NewAdministrativeOrderQueryHandler(store)

	assert.Equal(t, models.DocumentKindAdministrativeOrder, handler.Kind())

	card, err := handler.GetCard(docID)
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.NotNil(t, card.AdministrativeOrder)

	res, err := handler.GetList(models.DocumentFilter{Search: "Приказ"})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "Приказ", store.lastFilter.Search)
	assert.Equal(t, string(models.DocumentKindAdministrativeOrder), res.Items[0].KindCode)
}

func TestDocumentKindQueryHandlers_ReturnRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repository failed")

	incoming := NewIncomingLetterQueryHandler(&queryIncomingDocStore{err: repoErr})
	card, err := incoming.GetCard(uuid.New())
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, card)
	list, err := incoming.GetList(models.DocumentFilter{})
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, list)

	outgoing := NewOutgoingLetterQueryHandler(&queryOutgoingDocStore{err: repoErr})
	card, err = outgoing.GetCard(uuid.New())
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, card)
	list, err = outgoing.GetList(models.DocumentFilter{})
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, list)
}
