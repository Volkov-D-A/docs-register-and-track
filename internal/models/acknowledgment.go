package models

import (
	"time"

	"github.com/google/uuid"
)

// Acknowledgment - задача на ознакомление
type Acknowledgment struct {
	ID             uuid.UUID `json:"-"`
	IDStr          string    `json:"id"`
	DocumentID     uuid.UUID `json:"-"`
	DocumentIDStr  string    `json:"documentId"`
	DocumentType   string    `json:"documentType"` // 'incoming' or 'outgoing'
	DocumentNumber string    `json:"documentNumber,omitempty"`

	CreatorID    uuid.UUID `json:"-"`
	CreatorIDStr string    `json:"creatorId"`
	CreatorName  string    `json:"creatorName,omitempty"`

	Content     string     `json:"content"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Users involved in acknowledgment
	Users   []AcknowledgmentUser `json:"users,omitempty"`
	UserIDs []string             `json:"userIds,omitempty"` // For creation
}

func (a *Acknowledgment) FillIDStr() {
	a.IDStr = a.ID.String()
	a.DocumentIDStr = a.DocumentID.String()
	a.CreatorIDStr = a.CreatorID.String()
	for i := range a.Users {
		a.Users[i].FillIDStr()
	}
}

type AcknowledgmentUser struct {
	ID               uuid.UUID  `json:"-"`
	IDStr            string     `json:"id"`
	AcknowledgmentID uuid.UUID  `json:"-"`
	UserID           uuid.UUID  `json:"-"`
	UserIDStr        string     `json:"userId"`
	UserName         string     `json:"userName,omitempty"`
	ViewedAt         *time.Time `json:"viewedAt,omitempty"`
	ConfirmedAt      *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func (au *AcknowledgmentUser) FillIDStr() {
	au.IDStr = au.ID.String()
	au.UserIDStr = au.UserID.String()
}

type AcknowledgmentFilter struct {
	DocumentID   string `json:"documentId,omitempty"`
	UserID       string `json:"userId,omitempty"`
	Status       string `json:"status,omitempty"` // pending, completed
	ShowFinished bool   `json:"showFinished"`
}
