package models

const (
	SystemPermissionAdmin            = "admin"
	SystemPermissionReferences       = "references"
	SystemPermissionStatsIncoming    = "stats_incoming"
	SystemPermissionStatsOutgoing    = "stats_outgoing"
	SystemPermissionStatsAssignments = "stats_assignments"
	SystemPermissionStatsSystem      = "stats_system"
)

// UserDocumentPermissionRule описывает прямое назначение действия пользователю.
type UserDocumentPermissionRule struct {
	KindCode  string `json:"kindCode"`
	Action    string `json:"action"`
	IsAllowed bool   `json:"isAllowed"`
}

// UserSystemPermissionRule описывает прямое системное право пользователя.
type UserSystemPermissionRule struct {
	Permission string `json:"permission"`
	IsAllowed  bool   `json:"isAllowed"`
}

// UserDocumentAccessProfile содержит прямые document-domain права пользователя.
type UserDocumentAccessProfile struct {
	SystemPermissions []UserSystemPermissionRule   `json:"systemPermissions"`
	Permissions       []UserDocumentPermissionRule `json:"permissions"`
}

// UpdateUserDocumentAccessRequest описывает запрос на замену прямых прав пользователя.
type UpdateUserDocumentAccessRequest struct {
	UserID            string                       `json:"userId"`
	SystemPermissions []UserSystemPermissionRule   `json:"systemPermissions"`
	Permissions       []UserDocumentPermissionRule `json:"permissions"`
}
