package dto

import "time"

// DocumentSearchRequest searches card text across all readable document kinds.
type DocumentSearchRequest struct {
	Query    string `json:"query"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type DocumentSearchItem struct {
	ID                 string    `json:"id"`
	KindCode           string    `json:"kindCode"`
	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	Content            string    `json:"content"`
	Resolution         string    `json:"resolution"`
	Correspondent      string    `json:"correspondent"`
	Person             string    `json:"person"`
	Relevance          float64   `json:"relevance"`
}

type DocumentSearchResult struct {
	Items      []DocumentSearchItem `json:"items"`
	TotalCount int                  `json:"totalCount"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
}
