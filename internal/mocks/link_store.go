// Код сгенерирован mockery v2.53.6. НЕ РЕДАКТИРОВАТЬ.

package mocks

import (
	context "context"
	models "docflow/internal/models"

	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// LinkStore — это автосгенерированный мок-тип для типа LinkStore
type LinkStore struct {
	mock.Mock
}

// Create предоставляет мок-функцию с заданными полями: ctx, link
func (_m *LinkStore) Create(ctx context.Context, link *models.DocumentLink) error {
	ret := _m.Called(ctx, link)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.DocumentLink) error); ok {
		r0 = rf(ctx, link)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete предоставляет мок-функцию с заданными полями: ctx, id
func (_m *LinkStore) Delete(ctx context.Context, id uuid.UUID) error {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetByDocumentID предоставляет мок-функцию с заданными полями: ctx, docID
func (_m *LinkStore) GetByDocumentID(ctx context.Context, docID uuid.UUID) ([]models.DocumentLink, error) {
	ret := _m.Called(ctx, docID)

	if len(ret) == 0 {
		panic("no return value specified for GetByDocumentID")
	}

	var r0 []models.DocumentLink
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) ([]models.DocumentLink, error)); ok {
		return rf(ctx, docID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) []models.DocumentLink); ok {
		r0 = rf(ctx, docID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.DocumentLink)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, docID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGraph предоставляет мок-функцию с заданными полями: ctx, rootID
func (_m *LinkStore) GetGraph(ctx context.Context, rootID uuid.UUID) ([]models.DocumentLink, error) {
	ret := _m.Called(ctx, rootID)

	if len(ret) == 0 {
		panic("no return value specified for GetGraph")
	}

	var r0 []models.DocumentLink
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) ([]models.DocumentLink, error)); ok {
		return rf(ctx, rootID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) []models.DocumentLink); ok {
		r0 = rf(ctx, rootID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.DocumentLink)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uuid.UUID) error); ok {
		r1 = rf(ctx, rootID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewLinkStore создает новый экземпляр LinkStore. Он также регистрирует тестируемый интерфейс в моке и функцию очистки для проверки ожиданий мока.
// Первым аргументом обычно является значение *testing.T.
func NewLinkStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *LinkStore {
	mock := &LinkStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
