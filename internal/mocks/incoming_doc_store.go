// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	time "time"

	uuid "github.com/google/uuid"
)

// IncomingDocStore — это автосгенерированный мок-тип для типа IncomingDocStore
type IncomingDocStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, incomingNumber, incomingDate, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution
func (_m *IncomingDocStore) Create(nomenclatureID uuid.UUID, documentTypeID uuid.UUID, senderOrgID uuid.UUID, recipientOrgID uuid.UUID, createdBy uuid.UUID, incomingNumber string, incomingDate time.Time, outgoingNumberSender string, outgoingDateSender time.Time, intermediateNumber *string, intermediateDate *time.Time, subject string, content string, pagesCount int, senderSignatory string, senderExecutor string, addressee string, resolution *string) (*models.IncomingDocument, error) {
	ret := _m.Called(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, incomingNumber, incomingDate, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.IncomingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, time.Time, *string, *time.Time, string, string, int, string, string, string, *string) (*models.IncomingDocument, error)); ok {
		return rf(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, incomingNumber, incomingDate, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, time.Time, *string, *time.Time, string, string, int, string, string, string, *string) *models.IncomingDocument); ok {
		r0 = rf(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, incomingNumber, incomingDate, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.IncomingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, time.Time, *string, *time.Time, string, string, int, string, string, string, *string) error); ok {
		r1 = rf(nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy, incomingNumber, incomingDate, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *IncomingDocStore) Delete(id uuid.UUID) error {
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
func (_m *IncomingDocStore) GetByID(id uuid.UUID) (*models.IncomingDocument, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetByID")
	}

	var r0 *models.IncomingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (*models.IncomingDocument, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) *models.IncomingDocument); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.IncomingDocument)
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
func (_m *IncomingDocStore) GetCount() (int, error) {
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
func (_m *IncomingDocStore) GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error) {
	ret := _m.Called(filter)

	if len(ret) == 0 {
		panic("no return value specified for GetList")
	}

	var r0 *models.PagedResult[models.IncomingDocument]
	var r1 error
	if rf, ok := ret.Get(0).(func(models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error)); ok {
		return rf(filter)
	}
	if rf, ok := ret.Get(0).(func(models.DocumentFilter) *models.PagedResult[models.IncomingDocument]); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.PagedResult[models.IncomingDocument])
		}
	}

	if rf, ok := ret.Get(1).(func(models.DocumentFilter) error); ok {
		r1 = rf(filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update предоставляет мок-функцию с заданными полями: id, documentTypeID, senderOrgID, recipientOrgID, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution
func (_m *IncomingDocStore) Update(id uuid.UUID, documentTypeID uuid.UUID, senderOrgID uuid.UUID, recipientOrgID uuid.UUID, outgoingNumberSender string, outgoingDateSender time.Time, intermediateNumber *string, intermediateDate *time.Time, subject string, content string, pagesCount int, senderSignatory string, senderExecutor string, addressee string, resolution *string) (*models.IncomingDocument, error) {
	ret := _m.Called(id, documentTypeID, senderOrgID, recipientOrgID, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.IncomingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, *string, *time.Time, string, string, int, string, string, string, *string) (*models.IncomingDocument, error)); ok {
		return rf(id, documentTypeID, senderOrgID, recipientOrgID, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, *string, *time.Time, string, string, int, string, string, string, *string) *models.IncomingDocument); ok {
		r0 = rf(id, documentTypeID, senderOrgID, recipientOrgID, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.IncomingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, *string, *time.Time, string, string, int, string, string, string, *string) error); ok {
		r1 = rf(id, documentTypeID, senderOrgID, recipientOrgID, outgoingNumberSender, outgoingDateSender, intermediateNumber, intermediateDate, subject, content, pagesCount, senderSignatory, senderExecutor, addressee, resolution)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewIncomingDocStore создает новый экземпляр IncomingDocStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewIncomingDocStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *IncomingDocStore {
	mock := &IncomingDocStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
