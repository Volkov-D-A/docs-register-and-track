// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// AttachmentStore — это автосгенерированный мок-тип для типа AttachmentStore
type AttachmentStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: a
func (_m *AttachmentStore) Create(a *models.Attachment) error {
	ret := _m.Called(a)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*models.Attachment) error); ok {
		r0 = rf(a)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *AttachmentStore) Delete(id uuid.UUID) error {
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

// GetByDocumentID предоставляет мок-функцию с заданными полями: docID
func (_m *AttachmentStore) GetByDocumentID(docID uuid.UUID) ([]models.Attachment, error) {
	ret := _m.Called(docID)

	if len(ret) == 0 {
		panic("no return value specified for GetByDocumentID")
	}

	var r0 []models.Attachment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) ([]models.Attachment, error)); ok {
		return rf(docID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) []models.Attachment); ok {
		r0 = rf(docID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Attachment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(docID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID предоставляет мок-функцию с заданными полями: id
func (_m *AttachmentStore) GetByID(id uuid.UUID) (*models.Attachment, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetByID")
	}

	var r0 *models.Attachment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (*models.Attachment, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) *models.Attachment); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Attachment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetContent предоставляет мок-функцию с заданными полями: id
func (_m *AttachmentStore) GetContent(id uuid.UUID) ([]byte, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetContent")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) ([]byte, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) []byte); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewAttachmentStore создает новый экземпляр AttachmentStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewAttachmentStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *AttachmentStore {
	mock := &AttachmentStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
