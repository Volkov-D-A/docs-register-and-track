// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// IncomingDocStore — это автосгенерированный мок-тип для типа IncomingDocStore
type IncomingDocStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: req
func (_m *IncomingDocStore) Create(req models.CreateIncomingDocRequest) (*models.IncomingDocument, error) {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.IncomingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(models.CreateIncomingDocRequest) (*models.IncomingDocument, error)); ok {
		return rf(req)
	}
	if rf, ok := ret.Get(0).(func(models.CreateIncomingDocRequest) *models.IncomingDocument); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.IncomingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(models.CreateIncomingDocRequest) error); ok {
		r1 = rf(req)
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

// Update предоставляет мок-функцию с заданными полями: req
func (_m *IncomingDocStore) Update(req models.UpdateIncomingDocRequest) (*models.IncomingDocument, error) {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.IncomingDocument
	var r1 error
	if rf, ok := ret.Get(0).(func(models.UpdateIncomingDocRequest) (*models.IncomingDocument, error)); ok {
		return rf(req)
	}
	if rf, ok := ret.Get(0).(func(models.UpdateIncomingDocRequest) *models.IncomingDocument); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.IncomingDocument)
		}
	}

	if rf, ok := ret.Get(1).(func(models.UpdateIncomingDocRequest) error); ok {
		r1 = rf(req)
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
