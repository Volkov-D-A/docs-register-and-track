package models

import (
	"time"

	"github.com/google/uuid"
)

type Department struct {
	ID              uuid.UUID      `json:"-"`
	IDStr           string         `json:"id"`
	Name            string         `json:"name"`
	NomenclatureIDs []string       `json:"nomenclatureIds"`
	Nomenclature    []Nomenclature `json:"nomenclature"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

func (d *Department) FillIDStr() {
	d.IDStr = d.ID.String()
}
