package models

import "github.com/google/uuid"

// Department представляет собой отдел или подразделение организации.
type Department struct {
	ID              uuid.UUID `json:"-"`
	Name            string    `json:"name"`
	NomenclatureIDs []string  `json:"nomenclatureIds"`
}
