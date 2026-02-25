// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// AcknowledgmentStore — это автосгенерированный мок-тип для типа AcknowledgmentStore
type AcknowledgmentStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: a
func (_m *AcknowledgmentStore) Create(a *models.Acknowledgment) error {
	ret := _m.Called(a)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*models.Acknowledgment) error); ok {
		r0 = rf(a)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *AcknowledgmentStore) Delete(id uuid.UUID) error {
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

// GetAllActive предоставляет мок-функцию без полей
func (_m *AcknowledgmentStore) GetAllActive() ([]models.Acknowledgment, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAllActive")
	}

	var r0 []models.Acknowledgment
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.Acknowledgment, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.Acknowledgment); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Acknowledgment)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByDocumentID предоставляет мок-функцию с заданными полями: documentID
func (_m *AcknowledgmentStore) GetByDocumentID(documentID uuid.UUID) ([]models.Acknowledgment, error) {
	ret := _m.Called(documentID)

	if len(ret) == 0 {
		panic("no return value specified for GetByDocumentID")
	}

	var r0 []models.Acknowledgment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) ([]models.Acknowledgment, error)); ok {
		return rf(documentID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) []models.Acknowledgment); ok {
		r0 = rf(documentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Acknowledgment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(documentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPendingForUser предоставляет мок-функцию с заданными полями: userID
func (_m *AcknowledgmentStore) GetPendingForUser(userID uuid.UUID) ([]models.Acknowledgment, error) {
	ret := _m.Called(userID)

	if len(ret) == 0 {
		panic("no return value specified for GetPendingForUser")
	}

	var r0 []models.Acknowledgment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) ([]models.Acknowledgment, error)); ok {
		return rf(userID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) []models.Acknowledgment); ok {
		r0 = rf(userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Acknowledgment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MarkConfirmed предоставляет мок-функцию с заданными полями: ackID, userID
func (_m *AcknowledgmentStore) MarkConfirmed(ackID uuid.UUID, userID uuid.UUID) error {
	ret := _m.Called(ackID, userID)

	if len(ret) == 0 {
		panic("no return value specified for MarkConfirmed")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID) error); ok {
		r0 = rf(ackID, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MarkViewed предоставляет мок-функцию с заданными полями: ackID, userID
func (_m *AcknowledgmentStore) MarkViewed(ackID uuid.UUID, userID uuid.UUID) error {
	ret := _m.Called(ackID, userID)

	if len(ret) == 0 {
		panic("no return value specified for MarkViewed")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID) error); ok {
		r0 = rf(ackID, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAcknowledgmentStore создает новый экземпляр AcknowledgmentStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewAcknowledgmentStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *AcknowledgmentStore {
	mock := &AcknowledgmentStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
