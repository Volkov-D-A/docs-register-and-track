package models

// DashboardStats — общая структура для статистики
type DashboardStats struct {
	Role string `json:"role"` // роль: executor, clerk, admin

	// Статистика исполнителя
	MyAssignmentsNew          int `json:"myAssignmentsNew,omitempty"`
	MyAssignmentsInProgress   int `json:"myAssignmentsInProgress,omitempty"`
	MyAssignmentsOverdue      int `json:"myAssignmentsOverdue,omitempty"`
	MyAssignmentsFinished     int `json:"myAssignmentsFinished,omitempty"`
	MyAssignmentsFinishedLate int `json:"myAssignmentsFinishedLate,omitempty"`

	// Статистика делопроизводителя
	IncomingCount              int `json:"incomingCount,omitempty"`
	OutgoingCount              int `json:"outgoingCount,omitempty"`
	AllAssignmentsOverdue      int `json:"allAssignmentsOverdue,omitempty"`
	AllAssignmentsFinished     int `json:"allAssignmentsFinished,omitempty"`
	AllAssignmentsFinishedLate int `json:"allAssignmentsFinishedLate,omitempty"`

	// Статистика администратора
	UserCount      int    `json:"userCount,omitempty"`
	TotalDocuments int    `json:"totalDocuments,omitempty"`
	DBSize         string `json:"dbSize,omitempty"`

	// Общий список (истекающие поручения)
	ExpiringAssignments []Assignment `json:"expiringAssignments,omitempty"`
}
