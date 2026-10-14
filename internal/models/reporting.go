package models

// ReportRequest describes one of the fixed, server-owned report templates.
type ReportRequest struct {
	Template         string `json:"template"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
	OrganizationID   string `json:"organizationId,omitempty"`
	DepartmentID     string `json:"departmentId,omitempty"`
	KindCode         string `json:"kindCode,omitempty"`
	NomenclatureID   string `json:"nomenclatureId,omitempty"`
	UserID           string `json:"userId,omitempty"`
	Status           string `json:"status,omitempty"`
	OnlyWithDeadline bool   `json:"onlyWithDeadline,omitempty"`
	Timezone         string `json:"timezone,omitempty"`
}

type ReportFilters struct {
	Kinds        []StatisticsOption `json:"kinds"`
	Nomenclature []StatisticsOption `json:"nomenclature"`
	Users        []StatisticsOption `json:"users"`
}

type ReportRow struct {
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Period   string  `json:"period,omitempty"`
	Incoming int     `json:"incoming,omitempty"`
	Outgoing int     `json:"outgoing,omitempty"`
	Overdue  int     `json:"overdue,omitempty"`
	Count    int     `json:"count"`
	Metric   float64 `json:"metric"`
}

type ReportResult struct {
	Template    string        `json:"template"`
	StartDate   string        `json:"startDate"`
	EndDate     string        `json:"endDate"`
	GeneratedAt string        `json:"generatedAt"`
	Definition  string        `json:"definition"`
	Filters     string        `json:"filters"`
	Version     int           `json:"version"`
	Timezone    string        `json:"timezone"`
	Rows        []ReportRow   `json:"rows"`
	Summary     ReportSummary `json:"summary"`
}

type ReportSummary struct {
	Count           int     `json:"count"`
	Incoming        int     `json:"incoming"`
	Outgoing        int     `json:"outgoing"`
	Overdue         int     `json:"overdue"`
	OpenOverdue     int     `json:"openOverdue"`
	Metric          float64 `json:"metric"`
	UniqueDocuments int     `json:"uniqueDocuments"`
}
