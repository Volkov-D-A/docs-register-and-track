package mocks

import (
	models "github.com/Volkov-D-A/docs-register-and-track/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// UserStore records user lookups for document service fixtures.
type UserStore struct {
	mock.Mock
}

// GetByID provides a mock function with given fields: id
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

// NewUserStore creates a new instance of UserStore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUserStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *UserStore {
	mock := &UserStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
