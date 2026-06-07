package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type userSubstitutionStoreStub struct {
	byPrincipal      map[uuid.UUID]*models.UserSubstitution
	activePrincipals []uuid.UUID
	isActive         map[[2]uuid.UUID]bool
	replaceCalls     []userSubstitutionReplaceCall
	err              error
}

type userSubstitutionReplaceCall struct {
	principalUserID  uuid.UUID
	substituteUserID *uuid.UUID
	startsAt         *time.Time
	endsAt           *time.Time
	isActive         bool
	createdBy        *uuid.UUID
}

func (s *userSubstitutionStoreStub) GetByPrincipalID(principalUserID uuid.UUID) (*models.UserSubstitution, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.byPrincipal == nil {
		return nil, nil
	}
	return s.byPrincipal[principalUserID], nil
}

func (s *userSubstitutionStoreStub) GetActivePrincipalIDs(substituteUserID uuid.UUID) ([]uuid.UUID, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.activePrincipals, nil
}

func (s *userSubstitutionStoreStub) IsActiveSubstitute(substituteUserID, principalUserID uuid.UUID) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	if s.isActive == nil {
		return false, nil
	}
	return s.isActive[[2]uuid.UUID{substituteUserID, principalUserID}], nil
}

func (s *userSubstitutionStoreStub) ReplaceForPrincipal(
	principalUserID uuid.UUID,
	substituteUserID *uuid.UUID,
	startsAt *time.Time,
	endsAt *time.Time,
	isActive bool,
	createdBy *uuid.UUID,
) (*models.UserSubstitution, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.replaceCalls = append(s.replaceCalls, userSubstitutionReplaceCall{
		principalUserID:  principalUserID,
		substituteUserID: substituteUserID,
		startsAt:         startsAt,
		endsAt:           endsAt,
		isActive:         isActive,
		createdBy:        createdBy,
	})
	if substituteUserID == nil {
		return nil, nil
	}
	return &models.UserSubstitution{
		ID:               uuid.New(),
		PrincipalUserID:  principalUserID,
		SubstituteUserID: *substituteUserID,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		IsActive:         isActive,
	}, nil
}
