// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"
)

// SettingsStore — это автосгенерированный мок-тип для типа SettingsStore
type SettingsStore struct {
	mock.Mock
}

// Get предоставляет мок-функцию с заданными полями: key
func (_m *SettingsStore) Get(key string) (*models.SystemSetting, error) {
	ret := _m.Called(key)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *models.SystemSetting
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*models.SystemSetting, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) *models.SystemSetting); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.SystemSetting)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAll предоставляет мок-функцию без полей
func (_m *SettingsStore) GetAll() ([]models.SystemSetting, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 []models.SystemSetting
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.SystemSetting, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.SystemSetting); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.SystemSetting)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update предоставляет мок-функцию с заданными полями: key, value
func (_m *SettingsStore) Update(key string, value string) error {
	ret := _m.Called(key, value)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewSettingsStore создает новый экземпляр SettingsStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewSettingsStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *SettingsStore {
	mock := &SettingsStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
