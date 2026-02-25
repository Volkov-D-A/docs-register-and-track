// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	time "time"

	uuid "github.com/google/uuid"
)

// DashboardStore — это автосгенерированный мок-тип для типа DashboardStore
type DashboardStore struct {
	mock.Mock
}

// GetAdminDocCounts предоставляет мок-функцию без полей
func (_m *DashboardStore) GetAdminDocCounts() (int, int, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAdminDocCounts")
	}

	var r0 int
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func() (int, int, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func() int); ok {
		r1 = rf()
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetAdminUserCount предоставляет мок-функцию без полей
func (_m *DashboardStore) GetAdminUserCount() (int, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAdminUserCount")
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

// GetDBSize предоставляет мок-функцию без полей
func (_m *DashboardStore) GetDBSize() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDBSize")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDocCountsByPeriod предоставляет мок-функцию с заданными полями: startDate, endDate
func (_m *DashboardStore) GetDocCountsByPeriod(startDate time.Time, endDate time.Time) (int, int, error) {
	ret := _m.Called(startDate, endDate)

	if len(ret) == 0 {
		panic("no return value specified for GetDocCountsByPeriod")
	}

	var r0 int
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) (int, int, error)); ok {
		return rf(startDate, endDate)
	}
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) int); ok {
		r0 = rf(startDate, endDate)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(time.Time, time.Time) int); ok {
		r1 = rf(startDate, endDate)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(time.Time, time.Time) error); ok {
		r2 = rf(startDate, endDate)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetExecutorFinishedCounts предоставляет мок-функцию с заданными полями: userID
func (_m *DashboardStore) GetExecutorFinishedCounts(userID uuid.UUID) (int, int, error) {
	ret := _m.Called(userID)

	if len(ret) == 0 {
		panic("no return value specified for GetExecutorFinishedCounts")
	}

	var r0 int
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (int, int, error)); ok {
		return rf(userID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) int); ok {
		r0 = rf(userID)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) int); ok {
		r1 = rf(userID)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(uuid.UUID) error); ok {
		r2 = rf(userID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetExecutorOverdueCount предоставляет мок-функцию с заданными полями: userID
func (_m *DashboardStore) GetExecutorOverdueCount(userID uuid.UUID) (int, error) {
	ret := _m.Called(userID)

	if len(ret) == 0 {
		panic("no return value specified for GetExecutorOverdueCount")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (int, error)); ok {
		return rf(userID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) int); ok {
		r0 = rf(userID)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetExecutorStatusCounts предоставляет мок-функцию с заданными полями: userID
func (_m *DashboardStore) GetExecutorStatusCounts(userID uuid.UUID) (int, int, error) {
	ret := _m.Called(userID)

	if len(ret) == 0 {
		panic("no return value specified for GetExecutorStatusCounts")
	}

	var r0 int
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (int, int, error)); ok {
		return rf(userID)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) int); ok {
		r0 = rf(userID)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) int); ok {
		r1 = rf(userID)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(uuid.UUID) error); ok {
		r2 = rf(userID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetExpiringAssignments предоставляет мок-функцию с заданными полями: userID, days
func (_m *DashboardStore) GetExpiringAssignments(userID *uuid.UUID, days int) ([]models.Assignment, error) {
	ret := _m.Called(userID, days)

	if len(ret) == 0 {
		panic("no return value specified for GetExpiringAssignments")
	}

	var r0 []models.Assignment
	var r1 error
	if rf, ok := ret.Get(0).(func(*uuid.UUID, int) ([]models.Assignment, error)); ok {
		return rf(userID, days)
	}
	if rf, ok := ret.Get(0).(func(*uuid.UUID, int) []models.Assignment); ok {
		r0 = rf(userID, days)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Assignment)
		}
	}

	if rf, ok := ret.Get(1).(func(*uuid.UUID, int) error); ok {
		r1 = rf(userID, days)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFinishedCountsByPeriod предоставляет мок-функцию с заданными полями: startDate, endDate
func (_m *DashboardStore) GetFinishedCountsByPeriod(startDate time.Time, endDate time.Time) (int, int, error) {
	ret := _m.Called(startDate, endDate)

	if len(ret) == 0 {
		panic("no return value specified for GetFinishedCountsByPeriod")
	}

	var r0 int
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) (int, int, error)); ok {
		return rf(startDate, endDate)
	}
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) int); ok {
		r0 = rf(startDate, endDate)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(time.Time, time.Time) int); ok {
		r1 = rf(startDate, endDate)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(time.Time, time.Time) error); ok {
		r2 = rf(startDate, endDate)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetOverdueCountByPeriod предоставляет мок-функцию с заданными полями: startDate, endDate
func (_m *DashboardStore) GetOverdueCountByPeriod(startDate time.Time, endDate time.Time) (int, error) {
	ret := _m.Called(startDate, endDate)

	if len(ret) == 0 {
		panic("no return value specified for GetOverdueCountByPeriod")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) (int, error)); ok {
		return rf(startDate, endDate)
	}
	if rf, ok := ret.Get(0).(func(time.Time, time.Time) int); ok {
		r0 = rf(startDate, endDate)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(time.Time, time.Time) error); ok {
		r1 = rf(startDate, endDate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDashboardStore создает новый экземпляр DashboardStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewDashboardStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *DashboardStore {
	mock := &DashboardStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
