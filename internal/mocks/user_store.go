// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// UserStore — это автосгенерированный мок-тип для типа UserStore
type UserStore struct {
	mock.Mock
}

// CountUsers предоставляет мок-функцию без полей
func (_m *UserStore) CountUsers() (int, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CountUsers")
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

// Create предоставляет мок-функцию с заданными полями: req
func (_m *UserStore) Create(req models.CreateUserRequest) (*models.User, error) {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.User
	var r1 error
	if rf, ok := ret.Get(0).(func(models.CreateUserRequest) (*models.User, error)); ok {
		return rf(req)
	}
	if rf, ok := ret.Get(0).(func(models.CreateUserRequest) *models.User); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
		}
	}

	if rf, ok := ret.Get(1).(func(models.CreateUserRequest) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAll предоставляет мок-функцию без полей
func (_m *UserStore) GetAll() ([]models.User, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 []models.User
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.User, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.User); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.User)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID предоставляет мок-функцию с заданными полями: id
func (_m *UserStore) GetByID(id uuid.UUID) (*models.User, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetByID")
	}

	var r0 *models.User
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (*models.User, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) *models.User); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByLogin предоставляет мок-функцию с заданными полями: login
func (_m *UserStore) GetByLogin(login string) (*models.User, error) {
	ret := _m.Called(login)

	if len(ret) == 0 {
		panic("no return value specified for GetByLogin")
	}

	var r0 *models.User
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*models.User, error)); ok {
		return rf(login)
	}
	if rf, ok := ret.Get(0).(func(string) *models.User); ok {
		r0 = rf(login)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(login)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetExecutors предоставляет мок-функцию без полей
func (_m *UserStore) GetExecutors() ([]models.User, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetExecutors")
	}

	var r0 []models.User
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]models.User, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []models.User); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.User)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ResetPassword предоставляет мок-функцию с заданными полями: userID, newPassword
func (_m *UserStore) ResetPassword(userID uuid.UUID, newPassword string) error {
	ret := _m.Called(userID, newPassword)

	if len(ret) == 0 {
		panic("no return value specified for ResetPassword")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string) error); ok {
		r0 = rf(userID, newPassword)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Update предоставляет мок-функцию с заданными полями: req
func (_m *UserStore) Update(req models.UpdateUserRequest) (*models.User, error) {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.User
	var r1 error
	if rf, ok := ret.Get(0).(func(models.UpdateUserRequest) (*models.User, error)); ok {
		return rf(req)
	}
	if rf, ok := ret.Get(0).(func(models.UpdateUserRequest) *models.User); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.User)
		}
	}

	if rf, ok := ret.Get(1).(func(models.UpdateUserRequest) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdatePassword предоставляет мок-функцию с заданными полями: userID, newPasswordHash
func (_m *UserStore) UpdatePassword(userID uuid.UUID, newPasswordHash string) error {
	ret := _m.Called(userID, newPasswordHash)

	if len(ret) == 0 {
		panic("no return value specified for UpdatePassword")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string) error); ok {
		r0 = rf(userID, newPasswordHash)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateProfile предоставляет мок-функцию с заданными полями: userID, req
func (_m *UserStore) UpdateProfile(userID uuid.UUID, req models.UpdateProfileRequest) error {
	ret := _m.Called(userID, req)

	if len(ret) == 0 {
		panic("no return value specified for UpdateProfile")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, models.UpdateProfileRequest) error); ok {
		r0 = rf(userID, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewUserStore создает новый экземпляр UserStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewUserStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *UserStore {
	mock := &UserStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
