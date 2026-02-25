package services

import (
	"errors"
	"reflect"
	"testing"

	"docflow/internal/models"

	"github.com/google/uuid"
)

func TestFormatDocumentNumber(t *testing.T) {
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
