package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// ReleaseNoteService предоставляет бизнес-логику для показа релизных изменений.
type ReleaseNoteService struct {
	repo         ReleaseNoteStore
	auth         *AuthService
	auditService *AdminAuditLogService
}

// NewReleaseNoteService создает новый экземпляр ReleaseNoteService.
func NewReleaseNoteService(repo ReleaseNoteStore, auth *AuthService, auditService *AdminAuditLogService) *ReleaseNoteService {
	return &ReleaseNoteService{
		repo:         repo,
		auth:         auth,
		auditService: auditService,
	}
}

// GetCurrent возвращает текущий релиз приложения для текущего пользователя.
func (s *ReleaseNoteService) GetCurrent() (*models.ReleaseNote, error) {
	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	return s.repo.GetCurrentForUser(userID)
}

// GetAll возвращает все релизы приложения для административного управления.
func (s *ReleaseNoteService) GetAll() ([]models.ReleaseNote, error) {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return nil, err
	}
	return s.repo.GetAll()
}

// Create создает новый релиз приложения.
func (s *ReleaseNoteService) Create(req models.CreateReleaseNoteRequest) (*models.ReleaseNote, error) {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return nil, err
	}

	req.Version = strings.TrimSpace(req.Version)
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}
	if len(req.Changes) == 0 {
		return nil, fmt.Errorf("at least one release change is required")
	}
	for i := range req.Changes {
		req.Changes[i].Title = strings.TrimSpace(req.Changes[i].Title)
		req.Changes[i].Description = strings.TrimSpace(req.Changes[i].Description)
		if req.Changes[i].Title == "" || req.Changes[i].Description == "" {
			return nil, fmt.Errorf("release changes must contain title and description")
		}
	}

	releasedAt, err := time.Parse(time.RFC3339, req.ReleasedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid release date: %w", err)
	}

	note, err := s.repo.Create(req, releasedAt)
	if err != nil {
		return nil, err
	}

	userID, userName := s.auth.GetCurrentAuditInfo()
	s.auditService.LogAction(userID, userName, "RELEASE_NOTE_CREATE", fmt.Sprintf("Создан релиз %s", note.Version))

	if note.IsCurrent {
		s.auditService.LogAction(userID, userName, "RELEASE_NOTE_SET_CURRENT", fmt.Sprintf("Текущим релизом назначен %s", note.Version))
	}

	return note, nil
}

// MarkCurrentViewed сохраняет факт ознакомления текущего пользователя с текущим релизом.
func (s *ReleaseNoteService) MarkCurrentViewed() error {
	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}

	note, err := s.repo.GetCurrentForUser(userID)
	if err != nil {
		return err
	}
	if note == nil {
		return nil
	}

	return s.repo.MarkViewed(note.ID, userID)
}

// MarkViewed сохраняет факт ознакомления текущего пользователя с конкретным релизом.
func (s *ReleaseNoteService) MarkViewed(releaseNoteID string) error {
	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}

	noteUUID, err := uuid.Parse(releaseNoteID)
	if err != nil {
		return fmt.Errorf("invalid release note ID: %w", err)
	}

	return s.repo.MarkViewed(noteUUID, userID)
}

// SetCurrent делает релиз текущим.
func (s *ReleaseNoteService) SetCurrent(releaseNoteID string) error {
	if err := s.auth.RequireActiveRole("admin"); err != nil {
		return err
	}

	noteUUID, err := uuid.Parse(releaseNoteID)
	if err != nil {
		return fmt.Errorf("invalid release note ID: %w", err)
	}

	if err := s.repo.SetCurrent(noteUUID); err != nil {
		return err
	}

	releases, err := s.repo.GetAll()
	if err == nil {
		for _, release := range releases {
			if release.ID == noteUUID {
				userID, userName := s.auth.GetCurrentAuditInfo()
				s.auditService.LogAction(userID, userName, "RELEASE_NOTE_SET_CURRENT", fmt.Sprintf("Текущим релизом назначен %s", release.Version))
				break
			}
		}
	}

	return nil
}
