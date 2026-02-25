// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	time "time"

	uuid "github.com/google/uuid"
)

// OutgoingDocStore — это автосгенерированный мок-тип для типа OutgoingDocStore
type OutgoingDocStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, outgoingNumber, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee
func (_m *OutgoingDocStore) Create(nomenclatureID uuid.UUID, documentTypeID uuid.UUID, senderOrgID uuid.UUID, recipientOrgID uuid.UUID, createdBy uuid.UUID, outgoingNumber string, outgoingDate time.Time, subject string, content string, pagesCount int, senderSignatory string, senderExecutor string, addressee string) (*models.OutgoingDocument, error) {
	ret := _m.Called(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, outgoingNumber, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.OutgoingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, string, int, string, string, string) (*models.OutgoingDocument, error)); ok {
		return rf(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, outgoingNumber, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, string, int, string, string, string) *models.OutgoingDocument); ok {
		r0 = rf(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, outgoingNumber, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.OutgoingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, string, int, string, string, string) error); ok {
		r1 = rf(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, outgoingNumber, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *OutgoingDocStore) Delete(id uuid.UUID) error {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetByID предоставляет мок-функцию с заданными полями: id
func (_m *OutgoingDocStore) GetByID(id uuid.UUID) (*models.OutgoingDocument, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetByID")
	}

	var r0 *models.OutgoingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (*models.OutgoingDocument, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) *models.OutgoingDocument); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.OutgoingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCount предоставляет мок-функцию без полей
func (_m *OutgoingDocStore) GetCount() (int, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetCount")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func() (int, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetList предоставляет мок-функцию с заданными полями: filter
func (_m *OutgoingDocStore) GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error) {
	ret := _m.Called(filter)

	if len(ret) == 0 {
		panic("no return value specified for GetList")
	}

	var r0 *models.PagedResult[models.OutgoingDocument]
	var r1 error
	if rf, ok := ret.Get(0).(func(models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error)); ok {
		return rf(filter)
	}
	if rf, ok := ret.Get(0).(func(models.OutgoingDocumentFilter) *models.PagedResult[models.OutgoingDocument]); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.PagedResult[models.OutgoingDocument])
		}
	}

	if rf, ok := ret.Get(1).(func(models.OutgoingDocumentFilter) error); ok {
		r1 = rf(filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update предоставляет мок-функцию с заданными полями: id, documentTypeID, senderOrgID, recipientOrgID, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee
func (_m *OutgoingDocStore) Update(id uuid.UUID, documentTypeID uuid.UUID, senderOrgID uuid.UUID, recipientOrgID uuid.UUID, outgoingDate time.Time, subject string, content string, pagesCount int, senderSignatory string, senderExecutor string, addressee string) (*models.OutgoingDocument, error) {
	ret := _m.Called(id, documentTypeID, senderOrgID, recipientOrgID, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.OutgoingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, string, string, int, string, string, string) (*models.OutgoingDocument, error)); ok {
		return rf(id, documentTypeID, senderOrgID, recipientOrgID, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, string, string, int, string, string, string) *models.OutgoingDocument); ok {
		r0 = rf(id, documentTypeID, senderOrgID, recipientOrgID, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.OutgoingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, string, string, int, string, string, string) error); ok {
		r1 = rf(id, documentTypeID, senderOrgID, recipientOrgID, outgoingDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewOutgoingDocStore создает новый экземпляр OutgoingDocStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewOutgoingDocStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *OutgoingDocStore {
	mock := &OutgoingDocStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
