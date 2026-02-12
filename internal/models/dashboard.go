package models

// DashboardStats — общая структура для статистики
type DashboardStats struct {
	Role string `json:"role"` // executor, clerk, admin

	// Executor Stats
	MyAssignmentsNew        int `json:"myAssignmentsNew,omitempty"`
	MyAssignmentsInProgress int `json:"myAssignmentsInProgress,omitempty"`
	MyAssignmentsOverdue    int `json:"myAssignmentsOverdue,omitempty"`

	// Clerk Stats
	IncomingCountMonth    int `json:"incomingCountMonth,omitempty"`
	OutgoingCountMonth    int `json:"outgoingCountMonth,omitempty"`
	AllAssignmentsOverdue int `json:"allAssignmentsOverdue,omitempty"`

	// Admin Stats
	UserCount      int    `json:"userCount,omitempty"`
	TotalDocuments int    `json:"totalDocuments,omitempty"`
	DBSize         string `json:"dbSize,omitempty"`

	// Common List (Expiring assignments)
	ExpiringAssignments []Assignment `json:"expiringAssignments,omitempty"`
}
