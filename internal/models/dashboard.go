package models

// DashboardStats — общая структура для статистики
type DashboardStats struct {
	Role string `json:"role"` // executor, clerk, admin

	// Executor Stats
	MyAssignmentsNew          int `json:"myAssignmentsNew,omitempty"`
	MyAssignmentsInProgress   int `json:"myAssignmentsInProgress,omitempty"`
	MyAssignmentsOverdue      int `json:"myAssignmentsOverdue,omitempty"`
	MyAssignmentsFinished     int `json:"myAssignmentsFinished,omitempty"`
	MyAssignmentsFinishedLate int `json:"myAssignmentsFinishedLate,omitempty"`

	// Clerk Stats
	IncomingCount              int `json:"incomingCount,omitempty"`
	OutgoingCount              int `json:"outgoingCount,omitempty"`
	AllAssignmentsOverdue      int `json:"allAssignmentsOverdue,omitempty"`
	AllAssignmentsFinished     int `json:"allAssignmentsFinished,omitempty"`
	AllAssignmentsFinishedLate int `json:"allAssignmentsFinishedLate,omitempty"`

	// Admin Stats
	UserCount      int    `json:"userCount,omitempty"`
	TotalDocuments int    `json:"totalDocuments,omitempty"`
	DBSize         string `json:"dbSize,omitempty"`

	// Common List (Expiring assignments)
	ExpiringAssignments []Assignment `json:"expiringAssignments,omitempty"`
}
