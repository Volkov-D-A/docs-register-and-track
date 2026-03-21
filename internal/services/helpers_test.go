package services

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

	"github.com/google/uuid"
)

func TestFormatDocumentNumber(t *testing.T) {
	// Тестирование функции формирования номера документа на основе индекса и порядкового номера
	tests := []struct {
		name     string
		index    string
		number   int
		expected string
	}{
		{
			name:     "simple case",
			index:    "01-02",
			number:   123,
			expected: "01-02/123",
		},
		{
			name:     "empty index",
			index:    "",
			number:   456,
			expected: "/456",
		},
		{
			name:     "zero number",
			index:    "ABC",
			number:   0,
			expected: "ABC/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDocumentNumber(tt.index, tt.number)
			if result != tt.expected {
				t.Errorf("formatDocumentNumber(%q, %d) = %q; want %q", tt.index, tt.number, result, tt.expected)
			}
		})
	}
}

func TestParseUUID(t *testing.T) {
	// Тестирование функции безопасного парсинга UUID из строкового представления
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			input:   "123e4567-e89b-12d3-a456-426614174000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseUUID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// MockDepartmentStore для тестов
type MockDepartmentStore struct {
	GetNomsFunc func(departmentID uuid.UUID) ([]string, error)
}

func (m *MockDepartmentStore) GetAll() ([]models.Department, error) {
	return nil, nil
}

func (m *MockDepartmentStore) GetNomenclatureIDs(departmentID uuid.UUID) ([]string, error) {
	if m.GetNomsFunc != nil {
		return m.GetNomsFunc(departmentID)
	}
	return nil, nil
}

func (m *MockDepartmentStore) Create(name string, nomenclatureIDs []string) (*models.Department, error) {
	return nil, nil
}

func (m *MockDepartmentStore) Update(id uuid.UUID, name string, nomenclatureIDs []string) (*models.Department, error) {
	return nil, nil
}

func (m *MockDepartmentStore) Delete(id uuid.UUID) error {
	return nil
}

func TestFilterNomenclaturesByDepartment(t *testing.T) {
	// Тестирование функции фильтрации списка номенклатур (дел) в зависимости от прав подразделения
	depID := uuid.New()
	depIDPtr := &depID

	tests := []struct {
		name         string
		departmentID *uuid.UUID
		mockStore    *MockDepartmentStore
		filterNomIDs []string
		filterNomID  string
		wantIDs      []string
		wantEmpty    bool
		wantErr      bool
	}{
		{
			name:         "nil department ID",
			departmentID: nil,
			mockStore:    &MockDepartmentStore{},
			wantIDs:      nil,
			wantEmpty:    true,
			wantErr:      false,
		},
		{
			name:         "store error",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return nil, errors.New("db error")
				},
			},
			wantIDs:   nil,
			wantEmpty: false,
			wantErr:   true,
		},
		{
			name:         "no allowed nomenclatures",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return []string{}, nil
				},
			},
			wantIDs:   nil,
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name:         "no filters, returns all allowed",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return []string{"nom1", "nom2"}, nil
				},
			},
			wantIDs:   []string{"nom1", "nom2"},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name:         "filter with array, partial match",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return []string{"nom1", "nom2", "nom3"}, nil
				},
			},
			filterNomIDs: []string{"nom2", "nom4"},
			wantIDs:      []string{"nom2"},
			wantEmpty:    false,
			wantErr:      false,
		},
		{
			name:         "filter with array, no match",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return []string{"nom1"}, nil
				},
			},
			filterNomIDs: []string{"nom2"},
			wantIDs:      nil,
			wantEmpty:    true,
			wantErr:      false,
		},
		{
			name:         "filter with single id, match",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return []string{"nom1", "nom2"}, nil
				},
			},
			filterNomID: "nom2",
			wantIDs:     nil, // function returns nil for single id match (meaning it doesn't need to override the filter)
			wantEmpty:   false,
			wantErr:     false,
		},
		{
			name:         "filter with single id, no match",
			departmentID: depIDPtr,
			mockStore: &MockDepartmentStore{
				GetNomsFunc: func(d uuid.UUID) ([]string, error) {
					return []string{"nom1"}, nil
				},
			},
			filterNomID: "nom2",
			wantIDs:     nil, // function returns nil for empty match too but returns wantEmpty true
			wantEmpty:   true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIDs, gotEmpty, err := filterNomenclaturesByDepartment(tt.departmentID, tt.mockStore, tt.filterNomIDs, tt.filterNomID)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterNomenclaturesByDepartment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEmpty != tt.wantEmpty {
				t.Errorf("filterNomenclaturesByDepartment() gotEmpty = %v, want %v", gotEmpty, tt.wantEmpty)
			}
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("filterNomenclaturesByDepartment() gotIDs = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestApplyExecutorNomenclatureFilter(t *testing.T) {
	// Тестирование логики ограничения доступа к документам по подразделению для роли executor
	depRepo := &MockDepartmentStore{
		GetNomsFunc: func(d uuid.UUID) ([]string, error) {
			return []string{"A", "B"}, nil
		},
	}
	
	// mock auth pieces but AuthService struct directly
	userStore := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userStore)

	hash, _ := security.HashPassword("CorrectP@ssword1!")
	
	setupUser := func(role string, depIDStr string) *models.User {
		var dep *models.Department
		if depIDStr != "" {
			dID, _ := uuid.Parse(depIDStr)
			// Sometimes models ID is string, sometimes uuid.UUID. The compiler said "cannot use depID (variable of type string) as uuid.UUID value"
			// Wait, if it expects uuid.UUID, then I must set it as uuid.UUID? But wait, User.Department is a pointer. 
			// Let me just set it as the correct type. Let's do a trick: I will just use reflection or let's assume it's string because in previous tests users had string IDs? 
			// No, it explicitly said: "cannot use depID (variable of type string) as uuid.UUID value in struct literal" 
			// So Department.ID is uuid.UUID! I will write:
			dep = &models.Department{ID: dID}
		}
		u := &models.User{
			ID:           uuid.New(),
			Login:        "test_" + uuid.New().String(),
			PasswordHash: hash,
			IsActive:     true,
			Roles:        []string{role},
			Department:   dep,
		}
		userStore.On("GetByLogin", u.Login).Return(u, nil).Once()
		auth.Login(u.Login, "CorrectP@ssword1!")
		userStore.On("GetByID", u.ID).Return(u, nil).Maybe()
		return u
	}

	t.Run("admin bypasses filter", func(t *testing.T) {
		setupUser("admin", "")
		filtered, isEmpty, err := applyExecutorNomenclatureFilter(auth, depRepo, []string{"X"}, "")
		if err != nil {
			t.Errorf("expected no err, got %v", err)
		}
		if isEmpty {
			t.Errorf("expected not empty")
		}
		if len(filtered) != 1 || filtered[0] != "X" {
			t.Errorf("expected original ids")
		}
		auth.Logout()
	})

	t.Run("executor with valid dep", func(t *testing.T) {
		setupUser("executor", uuid.New().String())
		filtered, isEmpty, err := applyExecutorNomenclatureFilter(auth, depRepo, nil, "")
		if err != nil {
			t.Errorf("expected no err, got %v", err)
		}
		if isEmpty {
			t.Errorf("expected not empty")
		}
		if len(filtered) != 2 || filtered[0] != "A" || filtered[1] != "B" {
			t.Errorf("expected A, B")
		}
		auth.Logout()
	})

	t.Run("executor with specific missing filter", func(t *testing.T) {
		setupUser("executor", uuid.New().String())
		filtered, isEmpty, err := applyExecutorNomenclatureFilter(auth, depRepo, []string{"C"}, "")
		if err != nil {
			t.Errorf("expected no err, got %v", err)
		}
		if !isEmpty {
			t.Errorf("expected empty")
		}
		if len(filtered) != 0 {
			t.Errorf("expected nil")
		}
		auth.Logout()
	})

	t.Run("executor no dep", func(t *testing.T) {
		setupUser("executor", "")
		filtered, isEmpty, err := applyExecutorNomenclatureFilter(auth, depRepo, nil, "")
		if err != nil {
			t.Errorf("expected no err, got %v", err)
		}
		if !isEmpty {
			t.Errorf("expected empty")
		}
		if len(filtered) != 0 {
			t.Errorf("expected nil")
		}
		auth.Logout()
	})
	
	t.Run("not authenticated", func(t *testing.T) {
		auth.Logout()
		// If not authenticated, HasRole("executor") is false, so it bypasses
		filtered, isEmpty, err := applyExecutorNomenclatureFilter(auth, depRepo, []string{"X"}, "")
		if err != nil {
			t.Errorf("expected no err, got %v", err)
		}
		if isEmpty {
			t.Errorf("expected not empty")
		}
		if len(filtered) != 1 || filtered[0] != "X" {
			t.Errorf("expected original ids")
		}
	})
}
