// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// NomenclatureStore — это автосгенерированный мок-тип для типа NomenclatureStore
type NomenclatureStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: name, index, year, direction
func (_m *NomenclatureStore) Create(name string, index string, year int, direction string) (*models.Nomenclature, error) {
	ret := _m.Called(name, index, year, direction)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.Nomenclature
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string, int, string) (*models.Nomenclature, error)); ok {
		return rf(name, index, year, direction)
	}
	if rf, ok := ret.Get(0).(func(string, string, int, string) *models.Nomenclature); ok {
		r0 = rf(name, index, year, direction)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Nomenclature)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string, int, string) error); ok {
		r1 = rf(name, index, year, direction)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *NomenclatureStore) Delete(id uuid.UUID) error {
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

// GetActiveByDirection предоставляет мок-функцию с заданными полями: direction, year
func (_m *NomenclatureStore) GetActiveByDirection(direction string, year int) ([]models.Nomenclature, error) {
	ret := _m.Called(direction, year)

	if len(ret) == 0 {
		panic("no return value specified for GetActiveByDirection")
	}

	var r0 []models.Nomenclature
	var r1 error
	if rf, ok := ret.Get(0).(func(string, int) ([]models.Nomenclature, error)); ok {
		return rf(direction, year)
	}
	if rf, ok := ret.Get(0).(func(string, int) []models.Nomenclature); ok {
		r0 = rf(direction, year)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Nomenclature)
		}
	}

	if rf, ok := ret.Get(1).(func(string, int) error); ok {
		r1 = rf(direction, year)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAll предоставляет мок-функцию с заданными полями: year, direction
func (_m *NomenclatureStore) GetAll(year int, direction string) ([]models.Nomenclature, error) {
	ret := _m.Called(year, direction)

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 []models.Nomenclature
	var r1 error
	if rf, ok := ret.Get(0).(func(int, string) ([]models.Nomenclature, error)); ok {
		return rf(year, direction)
	}
	if rf, ok := ret.Get(0).(func(int, string) []models.Nomenclature); ok {
		r0 = rf(year, direction)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Nomenclature)
		}
	}

	if rf, ok := ret.Get(1).(func(int, string) error); ok {
		r1 = rf(year, direction)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID предоставляет мок-функцию с заданными полями: id
func (_m *NomenclatureStore) GetByID(id uuid.UUID) (*models.Nomenclature, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetByID")
	}

	var r0 *models.Nomenclature
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (*models.Nomenclature, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) *models.Nomenclature); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Nomenclature)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNextNumber предоставляет мок-функцию с заданными полями: id
func (_m *NomenclatureStore) GetNextNumber(id uuid.UUID) (int, string, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetNextNumber")
	}

	var r0 int
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (int, string, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) int); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) string); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(uuid.UUID) error); ok {
		r2 = rf(id)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Update предоставляет мок-функцию с заданными полями: id, name, index, year, direction, isActive
func (_m *NomenclatureStore) Update(id uuid.UUID, name string, index string, year int, direction string, isActive bool) (*models.Nomenclature, error) {
	ret := _m.Called(id, name, index, year, direction, isActive)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.Nomenclature
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string, string, int, string, bool) (*models.Nomenclature, error)); ok {
		return rf(id, name, index, year, direction, isActive)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, string, string, int, string, bool) *models.Nomenclature); ok {
		r0 = rf(id, name, index, year, direction, isActive)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Nomenclature)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, string, string, int, string, bool) error); ok {
		r1 = rf(id, name, index, year, direction, isActive)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewNomenclatureStore создает новый экземпляр NomenclatureStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewNomenclatureStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *NomenclatureStore {
	mock := &NomenclatureStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
