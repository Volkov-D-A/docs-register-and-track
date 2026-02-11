package models

import (
	"time"

	"github.com/google/uuid"
)

// Nomenclature — дело номенклатуры
type Nomenclature struct {
	ID         uuid.UUID `json:"-"`
	IDStr      string    `json:"id"`
	Name       string    `json:"name"`
	Index      string    `json:"index"`
	Year       int       `json:"year"`
	Direction  string    `json:"direction"` // incoming, outgoing
	NextNumber int       `json:"nextNumber"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (n *Nomenclature) FillIDStr() {
	n.IDStr = n.ID.String()
}

// Organization — организация (автозаполняемый справочник)
type Organization struct {
	ID        uuid.UUID `json:"-"`
	IDStr     string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (o *Organization) FillIDStr() {
	o.IDStr = o.ID.String()
}

// DocumentType — тип документа
type DocumentType struct {
	ID        uuid.UUID `json:"-"`
	IDStr     string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func (d *DocumentType) FillIDStr() {
	d.IDStr = d.ID.String()
}
