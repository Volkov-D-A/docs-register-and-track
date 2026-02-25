// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// DepartmentStore — это автосгенерированный мок-тип для типа DepartmentStore
type DepartmentStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: name, nomenclatureIDs
func (_m *DepartmentStore) Create(name string, nomenclatureIDs []string) (*models.Department, error) {
	ret := _m.Called(name, nomenclatureIDs)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.Department
	var r1 error
	if rf, ok := ret.Get(0).(func(string, []string) (*models.Department, error)); ok {
		return rf(name, nomenclatureIDs)
	}
	if rf, ok := ret.Get(0).(func(string, []string) *models.Department); ok {
		r0 = rf(name, nomenclatureIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Department)
		}
	}

	if rf, ok := ret.Get(1).(func(string, []string) error); ok {
		r1 = rf(name, nomenclatureIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *DepartmentStore) Delete(id uuid.UUID) error {
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

// GetAll предоставляет мок-функцию без полей
func (_m *DepartmentStore) GetAll() ([]models.Department, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 []models.Department
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.Department, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.Department); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Department)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNomenclatureIDs предоставляет мок-функцию с заданными полями: departmentID
func (_m *DepartmentStore) GetNomenclatureIDs(departmentID uuid.UUID) ([]string, error) {
	ret := _m.Called(departmentID)

	if len(ret) == 0 {
		panic("no return value specified for GetNomenclatureIDs")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) ([]string, error)); ok {
		return rf(departmentID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) []string); ok {
		r0 = rf(departmentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(departmentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update предоставляет мок-функцию с заданными полями: id, name, nomenclatureIDs
func (_m *DepartmentStore) Update(id uuid.UUID, name string, nomenclatureIDs []string) (*models.Department, error) {
	ret := _m.Called(id, name, nomenclatureIDs)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.Department
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string, []string) (*models.Department, error)); ok {
		return rf(id, name, nomenclatureIDs)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, string, []string) *models.Department); ok {
		r0 = rf(id, name, nomenclatureIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Department)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, string, []string) error); ok {
		r1 = rf(id, name, nomenclatureIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDepartmentStore создает новый экземпляр DepartmentStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewDepartmentStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *DepartmentStore {
	mock := &DepartmentStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
