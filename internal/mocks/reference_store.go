// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// ReferenceStore — это автосгенерированный мок-тип для типа ReferenceStore
type ReferenceStore struct {
	mock.Mock
}

// CreateDocumentType предоставляет мок-функцию с заданными полями: name
func (_m *ReferenceStore) CreateDocumentType(name string) (*models.DocumentType, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for CreateDocumentType")
	}

	var r0 *models.DocumentType
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*models.DocumentType, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *models.DocumentType); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.DocumentType)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteDocumentType предоставляет мок-функцию с заданными полями: id
func (_m *ReferenceStore) DeleteDocumentType(id uuid.UUID) error {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for DeleteDocumentType")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteOrganization предоставляет мок-функцию с заданными полями: id
func (_m *ReferenceStore) DeleteOrganization(id uuid.UUID) error {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for DeleteOrganization")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FindOrCreateOrganization предоставляет мок-функцию с заданными полями: name
func (_m *ReferenceStore) FindOrCreateOrganization(name string) (*models.Organization, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for FindOrCreateOrganization")
	}

	var r0 *models.Organization
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*models.Organization, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *models.Organization); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Organization)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllDocumentTypes предоставляет мок-функцию без полей
func (_m *ReferenceStore) GetAllDocumentTypes() ([]models.DocumentType, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAllDocumentTypes")
	}

	var r0 []models.DocumentType
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.DocumentType, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.DocumentType); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.DocumentType)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllOrganizations предоставляет мок-функцию без полей
func (_m *ReferenceStore) GetAllOrganizations() ([]models.Organization, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAllOrganizations")
	}

	var r0 []models.Organization
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.Organization, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.Organization); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Organization)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SearchOrganizations предоставляет мок-функцию с заданными полями: query
func (_m *ReferenceStore) SearchOrganizations(query string) ([]models.Organization, error) {
	ret := _m.Called(query)

	if len(ret) == 0 {
		panic("no return value specified for SearchOrganizations")
	}

	var r0 []models.Organization
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]models.Organization, error)); ok {
		return rf(query)
	}
	if rf, ok := ret.Get(0).(func(string) []models.Organization); ok {
		r0 = rf(query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Organization)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateDocumentType предоставляет мок-функцию с заданными полями: id, name
func (_m *ReferenceStore) UpdateDocumentType(id uuid.UUID, name string) error {
	ret := _m.Called(id, name)

	if len(ret) == 0 {
		panic("no return value specified for UpdateDocumentType")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string) error); ok {
		r0 = rf(id, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateOrganization предоставляет мок-функцию с заданными полями: id, name
func (_m *ReferenceStore) UpdateOrganization(id uuid.UUID, name string) error {
	ret := _m.Called(id, name)

	if len(ret) == 0 {
		panic("no return value specified for UpdateOrganization")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string) error); ok {
		r0 = rf(id, name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewReferenceStore создает новый экземпляр ReferenceStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewReferenceStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReferenceStore {
	mock := &ReferenceStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
