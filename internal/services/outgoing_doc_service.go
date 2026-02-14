package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type OutgoingDocumentService struct {
	ctx     context.Context
	repo    *repository.OutgoingDocumentRepository
	refRepo *repository.ReferenceRepository
	nomRepo *repository.NomenclatureRepository
	depRepo *repository.DepartmentRepository
	auth    *AuthService
}

func NewOutgoingDocumentService(
	repo *repository.OutgoingDocumentRepository,
	refRepo *repository.ReferenceRepository,
	nomRepo *repository.NomenclatureRepository,
	depRepo *repository.DepartmentRepository,
	auth *AuthService,
) *OutgoingDocumentService {
	return &OutgoingDocumentService{
		repo:    repo,
		refRepo: refRepo,
		nomRepo: nomRepo,
		depRepo: depRepo,
		auth:    auth,
	}
}

func (s *OutgoingDocumentService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// Register — регистрация нового исходящего документа
func (s *OutgoingDocumentService) Register(
	nomenclatureID, documentTypeID string,
	recipientOrgName, addressee string,
	outgoingDate string,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor string,
) (*models.OutgoingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав для регистрации документов")
	}

	// Парсинг UUID
	nomID, err := uuid.Parse(nomenclatureID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID номенклатуры: %w", err)
	}
	docTypeID, err := uuid.Parse(documentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	// Автосоздание организации получателя
	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	// Организация-отправитель (наша организация) — пока хардкодим или берем из настроек
	// В данном случае, sender_org_id в таблице outgoing_documents — это организация ОТ КОТОРОЙ уходит документ.
	// Обычно это "Наша Организация".
	// Для упрощения, пока создадим/найдем организацию "Наша Организация" или возьмем первую попавшуюся.
	// TODO: Вынести ID "Нашей Организации" в конфиг или настройки.
	senderOrg, err := s.refRepo.FindOrCreateOrganization("НАША ОРГАНИЗАЦИЯ")
	if err != nil {
		return nil, fmt.Errorf("ошибка вашей организации: %w", err)
	}

	// Получение рег. номера из номенклатуры
	nextNum, index, err := s.nomRepo.GetNextNumber(nomID)
	if err != nil {
		return nil, fmt.Errorf("ошибка автонумерации: %w", err)
	}
	outgoingNumber := fmt.Sprintf("%s-%d", index, nextNum)

	// Парсинг даты
	outDate, err := time.Parse("2006-01-02", outgoingDate)
	if err != nil {
		outDate = time.Now()
	}

	// ID текущего пользователя
	createdByStr := s.auth.GetCurrentUserID()
	createdBy, _ := uuid.Parse(createdByStr)

	return s.repo.Create(
		nomID, docTypeID, senderOrg.ID, recipientOrg.ID, createdBy,
		outgoingNumber, outDate,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
	)
}

// Update — редактирование исходящего документа
func (s *OutgoingDocumentService) Update(
	id, documentTypeID string,
	recipientOrgName, addressee string,
	outgoingDate string,
	subject, content string, pagesCount int,
	senderSignatory, senderExecutor string,
) (*models.OutgoingDocument, error) {
	if !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		return nil, fmt.Errorf("недостаточно прав для редактирования документов")
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("неверный ID документа: %w", err)
	}
	docTypeID, err := uuid.Parse(documentTypeID)
	if err != nil {
		return nil, fmt.Errorf("неверный ID типа документа: %w", err)
	}

	recipientOrg, err := s.refRepo.FindOrCreateOrganization(recipientOrgName)
	if err != nil {
		return nil, fmt.Errorf("ошибка организации получателя: %w", err)
	}

	senderOrg, err := s.refRepo.FindOrCreateOrganization("НАША ОРГАНИЗАЦИЯ")
	if err != nil {
		return nil, fmt.Errorf("ошибка вашей организации: %w", err)
	}

	outDate, err := time.Parse("2006-01-02", outgoingDate)
	if err != nil {
		outDate = time.Now()
	}

	return s.repo.Update(
		uid,
		docTypeID, senderOrg.ID, recipientOrg.ID,
		outDate,
		subject, content, pagesCount,
		senderSignatory, senderExecutor, addressee,
	)
}

// GetList — получение списка с фильтрацией
func (s *OutgoingDocumentService) GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	// Defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	// Если пользователь — исполнитель, ограничиваем видимость
	if s.auth.HasRole("executor") && !s.auth.HasRole("admin") && !s.auth.HasRole("clerk") {
		if user.DepartmentID != nil {
			allowedNomenclatures, err := s.depRepo.GetNomenclatureIDs(*user.DepartmentID)
			if err != nil {
				return nil, fmt.Errorf("failed to get allowed nomenclatures: %w", err)
			}

			if len(allowedNomenclatures) == 0 {
				return &models.PagedResult{
					Items:      []models.OutgoingDocument{},
					TotalCount: 0,
					Page:       filter.Page,
					PageSize:   filter.PageSize,
				}, nil
			}

			if len(filter.NomenclatureIDs) > 0 {
				var intersection []string
				allowedMap := make(map[string]bool)
				for _, id := range allowedNomenclatures {
					allowedMap[id] = true
				}
				for _, id := range filter.NomenclatureIDs {
					if allowedMap[id] {
						intersection = append(intersection, id)
					}
				}
				if len(intersection) == 0 {
					return &models.PagedResult{
						Items:      []models.OutgoingDocument{},
						TotalCount: 0,
						Page:       filter.Page,
						PageSize:   filter.PageSize,
					}, nil
				}
				filter.NomenclatureIDs = intersection
			} else {
				filter.NomenclatureIDs = allowedNomenclatures
			}
		} else {
			return &models.PagedResult{
				Items:      []models.OutgoingDocument{},
				TotalCount: 0,
				Page:       filter.Page,
				PageSize:   filter.PageSize,
			}, nil
		}
	}

	return s.repo.GetList(filter)
}

// GetByID — получение документа по ID
func (s *OutgoingDocumentService) GetByID(id string) (*models.OutgoingDocument, error) {
	if !s.auth.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.GetByID(uid)
}

// Delete — удаление документа
func (s *OutgoingDocumentService) Delete(id string) error {
	if !s.auth.HasRole("admin") {
		return fmt.Errorf("недостаточно прав")
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(uid)
}

// GetCount — количество для дашборда
func (s *OutgoingDocumentService) GetCount() (int, error) {
	if !s.auth.IsAuthenticated() {
		return 0, ErrNotAuthenticated
	}
	return s.repo.GetCount()
}
