package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

var errServerDepartmentClientNotConfigured = errors.New("docflow-server department client is not configured")

// DepartmentService предоставляет бизнес-логику для работы с подразделениями.
type DepartmentService struct {
	server serverclient.DepartmentClient
}

func (s *DepartmentService) SetServerClient(client serverclient.DepartmentClient) {
	s.server = client
}

// NewDepartmentService создает новый экземпляр DepartmentService.
func NewDepartmentService() *DepartmentService {
	return &DepartmentService{}
}

// GetAllDepartments возвращает список всех подразделений.
func (s *DepartmentService) GetAllDepartments() ([]dto.Department, error) {
	if s.server == nil {
		return nil, errServerDepartmentClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.ListDepartments(ctx)
}

// CreateDepartment создает новое подразделение.
func (s *DepartmentService) CreateDepartment(name string, nomenclatureIDs []string) (*dto.Department, error) {
	if s.server == nil {
		return nil, errServerDepartmentClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.CreateDepartment(ctx, name, nomenclatureIDs)
}

// UpdateDepartment обновляет данные существующего подразделения.
func (s *DepartmentService) UpdateDepartment(id, name string, nomenclatureIDs []string) (*dto.Department, error) {
	if s.server == nil {
		return nil, errServerDepartmentClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.UpdateDepartment(ctx, id, name, nomenclatureIDs)
}

// DeleteDepartment удаляет подразделение по его ID.
func (s *DepartmentService) DeleteDepartment(id string) error {
	if s.server == nil {
		return errServerDepartmentClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.DeleteDepartment(ctx, id)
}
