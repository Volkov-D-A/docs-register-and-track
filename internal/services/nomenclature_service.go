package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

var errServerNomenclatureClientNotConfigured = errors.New("docflow-server nomenclature client is not configured")

// NomenclatureService is the Wails adapter for server-owned nomenclature use cases.
type NomenclatureService struct {
	server serverclient.NomenclatureClient
}

func NewNomenclatureService() *NomenclatureService {
	return &NomenclatureService{}
}

func (s *NomenclatureService) SetServerClient(client serverclient.NomenclatureClient) {
	s.server = client
}

func (s *NomenclatureService) GetAll(year int, kindCode string) ([]dto.Nomenclature, error) {
	if s.server == nil {
		return nil, errServerNomenclatureClientNotConfigured
	}
	ctx, cancel := nomenclatureContext()
	defer cancel()
	return s.server.ListNomenclature(ctx, year, kindCode)
}

func (s *NomenclatureService) GetActiveForKind(kindCode string) ([]dto.Nomenclature, error) {
	if s.server == nil {
		return nil, errServerNomenclatureClientNotConfigured
	}
	ctx, cancel := nomenclatureContext()
	defer cancel()
	return s.server.ListActiveNomenclature(ctx, kindCode)
}

func (s *NomenclatureService) Create(name, index string, year int, kindCode, separator, numberingMode string, startNumber int) (*dto.Nomenclature, error) {
	if s.server == nil {
		return nil, errServerNomenclatureClientNotConfigured
	}
	ctx, cancel := nomenclatureContext()
	defer cancel()
	return s.server.CreateNomenclature(ctx, name, index, year, kindCode, separator, numberingMode, startNumber)
}

func (s *NomenclatureService) Update(id, name, index string, year int, kindCode, separator, numberingMode string, isActive bool) (*dto.Nomenclature, error) {
	if s.server == nil {
		return nil, errServerNomenclatureClientNotConfigured
	}
	ctx, cancel := nomenclatureContext()
	defer cancel()
	return s.server.UpdateNomenclature(ctx, id, name, index, year, kindCode, separator, numberingMode, isActive)
}

func (s *NomenclatureService) Delete(id string) error {
	if s.server == nil {
		return errServerNomenclatureClientNotConfigured
	}
	ctx, cancel := nomenclatureContext()
	defer cancel()
	return s.server.DeleteNomenclature(ctx, id)
}

func nomenclatureContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}
