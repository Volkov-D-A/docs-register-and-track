// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	time "time"

	uuid "github.com/google/uuid"
)

// AssignmentStore — это автосгенерированный мок-тип для типа AssignmentStore
type AssignmentStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: documentID, documentType, executorID, content, deadline, coExecutorIDs
func (_m *AssignmentStore) Create(documentID uuid.UUID, documentType string, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string) (*models.Assignment, error) {
	ret := _m.Called(documentID, documentType, executorID, content, deadline, coExecutorIDs)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *models.Assignment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, string, uuid.UUID, string, *time.Time, []string) (*models.Assignment, error)); ok {
		return rf(documentID, documentType, executorID, content, deadline, coExecutorIDs)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, string, uuid.UUID, string, *time.Time, []string) *models.Assignment); ok {
		r0 = rf(documentID, documentType, executorID, content, deadline, coExecutorIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Assignment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, string, uuid.UUID, string, *time.Time, []string) error); ok {
		r1 = rf(documentID, documentType, executorID, content, deadline, coExecutorIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete предоставляет мок-функцию с заданными полями: id
func (_m *AssignmentStore) Delete(id uuid.UUID) error {
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
func (_m *AssignmentStore) GetByID(id uuid.UUID) (*models.Assignment, error) {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for GetByID")
	}

	var r0 *models.Assignment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID) (*models.Assignment, error)); ok {
		return rf(id)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID) *models.Assignment); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Assignment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetList предоставляет мок-функцию с заданными полями: filter
func (_m *AssignmentStore) GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error) {
	ret := _m.Called(filter)

	if len(ret) == 0 {
		panic("no return value specified for GetList")
	}

	var r0 *models.PagedResult[models.Assignment]
	var r1 error
	if rf, ok := ret.Get(0).(func(models.AssignmentFilter) (*models.PagedResult[models.Assignment], error)); ok {
		return rf(filter)
	}
	if rf, ok := ret.Get(0).(func(models.AssignmentFilter) *models.PagedResult[models.Assignment]); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.PagedResult[models.Assignment])
		}
	}

	if rf, ok := ret.Get(1).(func(models.AssignmentFilter) error); ok {
		r1 = rf(filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update предоставляет мок-функцию с заданными полями: id, executorID, content, deadline, status, report, completedAt, coExecutorIDs
func (_m *AssignmentStore) Update(id uuid.UUID, executorID uuid.UUID, content string, deadline *time.Time, status string, report string, completedAt *time.Time, coExecutorIDs []string) (*models.Assignment, error) {
	ret := _m.Called(id, executorID, content, deadline, status, report, completedAt, coExecutorIDs)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *models.Assignment
	var r1 error
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, string, *time.Time, string, string, *time.Time, []string) (*models.Assignment, error)); ok {
		return rf(id, executorID, content, deadline, status, report, completedAt, coExecutorIDs)
	}
	if rf, ok := ret.Get(0).(func(uuid.UUID, uuid.UUID, string, *time.Time, string, string, *time.Time, []string) *models.Assignment); ok {
		r0 = rf(id, executorID, content, deadline, status, report, completedAt, coExecutorIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Assignment)
		}
	}

	if rf, ok := ret.Get(1).(func(uuid.UUID, uuid.UUID, string, *time.Time, string, string, *time.Time, []string) error); ok {
		r1 = rf(id, executorID, content, deadline, status, report, completedAt, coExecutorIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewAssignmentStore создает новый экземпляр AssignmentStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewAssignmentStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *AssignmentStore {
	mock := &AssignmentStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
