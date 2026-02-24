package models

import (
	"time"

	"github.com/google/uuid"
)

// Nomenclature — дело номенклатуры
type Nomenclature struct {
	ID         uuid.UUID `json:"-"`
	Name       string    `json:"name"`
	Index      string    `json:"index"`
	Year       int       `json:"year"`
	Direction  string    `json:"direction"` // incoming, outgoing
	NextNumber int       `json:"nextNumber"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Organization — организация (автозаполняемый справочник)
type Organization struct {
	ID        uuid.UUID `json:"-"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// DocumentType — тип документа
type DocumentType struct {
	ID        uuid.UUID `json:"-"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
