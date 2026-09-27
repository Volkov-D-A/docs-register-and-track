package models

import "github.com/google/uuid"

// Nomenclature — дело номенклатуры
type Nomenclature struct {
	ID            uuid.UUID `json:"-"`
	Name          string    `json:"name"`
	Index         string    `json:"index"`
	Year          int       `json:"year"`
	KindCode      string    `json:"kindCode"`
	Separator     string    `json:"separator"`
	NumberingMode string    `json:"numberingMode"`
	NextNumber    int       `json:"nextNumber"`
	IsActive      bool      `json:"isActive"`
}

// Organization — организация (автозаполняемый справочник)
type Organization struct {
	ID   uuid.UUID `json:"-"`
	Name string    `json:"name"`
}

// ResolutionExecutor — исполнитель резолюции (автозаполняемый справочник)
type ResolutionExecutor struct {
	ID   uuid.UUID `json:"-"`
	Name string    `json:"name"`
}
