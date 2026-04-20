package dto

import "time"

// User описывает DTO пользователя.
type User struct {
	ID                    string      `json:"id"`
	Login                 string      `json:"login"`
	FullName              string      `json:"fullName"`
	IsDocumentParticipant bool        `json:"isDocumentParticipant"`
	IsActive              bool        `json:"isActive"`
	FailedLoginAttempts   int         `json:"failedLoginAttempts"`
	SystemPermissions     []string    `json:"systemPermissions"`
	CreatedAt             time.Time   `json:"createdAt"`
	UpdatedAt             time.Time   `json:"updatedAt"`
	Department            *Department `json:"department,omitempty"`
}

// Department описывает DTO подразделения.
type Department struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	NomenclatureIDs []string       `json:"nomenclatureIds"`
	Nomenclature    []Nomenclature `json:"nomenclature"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// Nomenclature описывает DTO номенклатуры дел.
type Nomenclature struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Index         string    `json:"index"`
	Year          int       `json:"year"`
	KindCode      string    `json:"kindCode"`
	Separator     string    `json:"separator"`
	NumberingMode string    `json:"numberingMode"`
	NextNumber    int       `json:"nextNumber"`
	IsActive      bool      `json:"isActive"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Organization описывает DTO организации.
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// DocumentType описывает DTO типа документа.
type DocumentType struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// DocumentKind описывает DTO системного вида документа.
type DocumentKind struct {
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	LegacyViewType       string   `json:"legacyViewType"`
	RegistrationFormCode string   `json:"registrationFormCode"`
	RegistryGroup        string   `json:"registryGroup"`
	SupportedActions     []string `json:"supportedActions"`
	AvailableActions     []string `json:"availableActions"`
}

// ResolutionExecutor описывает DTO исполнителя резолюции.
type ResolutionExecutor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// Новые структуры для ответов (DTO)

// IncomingDocument описывает DTO входящего документа.
type IncomingDocument struct {
	ID               string `json:"id"`
	NomenclatureID   string `json:"nomenclatureId"`
	NomenclatureName string `json:"nomenclatureName,omitempty"`

	IncomingNumber       string     `json:"incomingNumber"`
	IncomingDate         time.Time  `json:"incomingDate"`
	OutgoingNumberSender string     `json:"outgoingNumberSender"`
	OutgoingDateSender   time.Time  `json:"outgoingDateSender"`
	IntermediateNumber   *string    `json:"intermediateNumber,omitempty"`
	IntermediateDate     *time.Time `json:"intermediateDate,omitempty"`

	DocumentTypeID   string `json:"documentTypeId"`
	DocumentTypeName string `json:"documentTypeName,omitempty"`
	Content          string `json:"content"`
	PagesCount       int    `json:"pagesCount"`

	SenderOrgID     string `json:"senderOrgId"`
	SenderOrgName   string `json:"senderOrgName,omitempty"`
	SenderSignatory string `json:"senderSignatory"`

	Resolution          *string `json:"resolution,omitempty"`
	ResolutionAuthor    *string `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors *string `json:"resolutionExecutors,omitempty"`

	CreatedBy     string    `json:"createdBy"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	AttachmentsCount int `json:"attachmentsCount,omitempty"`
	AssignmentsCount int `json:"assignmentsCount,omitempty"`
}

// OutgoingDocument описывает DTO исходящего документа.
type OutgoingDocument struct {
	ID               string `json:"id"`
	NomenclatureID   string `json:"nomenclatureId"`
	NomenclatureName string `json:"nomenclatureName,omitempty"`

	OutgoingNumber string    `json:"outgoingNumber"`
	OutgoingDate   time.Time `json:"outgoingDate"`

	DocumentTypeID   string `json:"documentTypeId"`
	DocumentTypeName string `json:"documentTypeName,omitempty"`
	Content          string `json:"content"`
	PagesCount       int    `json:"pagesCount"`

	SenderSignatory string `json:"senderSignatory"`
	SenderExecutor  string `json:"senderExecutor"`

	RecipientOrgID   string `json:"recipientOrgId"`
	RecipientOrgName string `json:"recipientOrgName,omitempty"`
	Addressee        string `json:"addressee"`

	CreatedBy     string    `json:"createdBy"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	AttachmentsCount int `json:"attachmentsCount,omitempty"`
}

// DocumentCard описывает общую оболочку документа с привязанными деталями конкретного вида.
type DocumentCard struct {
	ID                 string    `json:"id"`
	KindCode           string    `json:"kindCode"`
	KindName           string    `json:"kindName"`
	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	NomenclatureID     string    `json:"nomenclatureId"`
	NomenclatureName   string    `json:"nomenclatureName,omitempty"`
	DocumentTypeID     string    `json:"documentTypeId"`
	DocumentTypeName   string    `json:"documentTypeName,omitempty"`
	Content            string    `json:"content"`
	CreatedBy          string    `json:"createdBy"`
	CreatedByName      string    `json:"createdByName,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`

	IncomingLetter *IncomingDocument `json:"incomingLetter,omitempty"`
	OutgoingLetter *OutgoingDocument `json:"outgoingLetter,omitempty"`
}

// DocumentListItem описывает общую строку списка документов с detail-полями для конкретного вида.
type DocumentListItem struct {
	ID                 string    `json:"id"`
	KindCode           string    `json:"kindCode"`
	KindName           string    `json:"kindName"`
	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	NomenclatureID     string    `json:"nomenclatureId"`
	NomenclatureName   string    `json:"nomenclatureName,omitempty"`
	DocumentTypeID     string    `json:"documentTypeId"`
	DocumentTypeName   string    `json:"documentTypeName,omitempty"`
	Content            string    `json:"content"`
	CreatedBy          string    `json:"createdBy"`
	CreatedByName      string    `json:"createdByName,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`

	IncomingNumber       string     `json:"incomingNumber,omitempty"`
	IncomingDate         *time.Time `json:"incomingDate,omitempty"`
	OutgoingNumber       string     `json:"outgoingNumber,omitempty"`
	OutgoingDate         *time.Time `json:"outgoingDate,omitempty"`
	OutgoingNumberSender string     `json:"outgoingNumberSender,omitempty"`
	OutgoingDateSender   *time.Time `json:"outgoingDateSender,omitempty"`
	IntermediateNumber   *string    `json:"intermediateNumber,omitempty"`
	IntermediateDate     *time.Time `json:"intermediateDate,omitempty"`
	SenderOrgName        string     `json:"senderOrgName,omitempty"`
	SenderSignatory      string     `json:"senderSignatory,omitempty"`
	Resolution           *string    `json:"resolution,omitempty"`
	ResolutionAuthor     *string    `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors  *string    `json:"resolutionExecutors,omitempty"`
	RecipientOrgName     string     `json:"recipientOrgName,omitempty"`
	Addressee            string     `json:"addressee,omitempty"`
	SenderExecutor       string     `json:"senderExecutor,omitempty"`
}

// DocumentLink описывает DTO связи между документами.
type DocumentLink struct {
	ID         string    `json:"id"`
	SourceKind string    `json:"sourceKind"`
	SourceID   string    `json:"sourceId"`
	TargetKind string    `json:"targetKind"`
	TargetID   string    `json:"targetId"`
	LinkType   string    `json:"linkType"`
	CreatedBy  string    `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`

	SourceNumber  string `json:"sourceNumber,omitempty"`
	TargetNumber  string `json:"targetNumber,omitempty"`
	TargetSubject string `json:"targetSubject,omitempty"`
}

// Attachment описывает DTO прикрепленного файла.
type Attachment struct {
	ID             string    `json:"id"`
	DocumentID     string    `json:"documentId"`
	Filename       string    `json:"filename"`
	Filepath       string    `json:"filepath"`
	FileSize       int64     `json:"fileSize"`
	ContentType    string    `json:"contentType"`
	UploadedBy     string    `json:"uploadedBy"`
	UploadedByName string    `json:"uploadedByName,omitempty"`
	UploadedAt     time.Time `json:"uploadedAt"`
}

// DownloadResponse описывает DTO ответа при скачивании файла.
type DownloadResponse struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // base64
}

// Assignment описывает DTO поручения.
type Assignment struct {
	ID           string `json:"id"`
	DocumentID   string `json:"documentId"`
	DocumentKind string `json:"documentKind"`

	ExecutorID   string `json:"executorId"`
	ExecutorName string `json:"executorName,omitempty"`

	Content     string     `json:"content"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Status      string     `json:"status"`
	Report      string     `json:"report,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	DocumentNumber  string `json:"documentNumber,omitempty"`
	DocumentSubject string `json:"documentSubject,omitempty"`

	CoExecutors   []User   `json:"coExecutors,omitempty"`
	CoExecutorIDs []string `json:"coExecutorIds,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Acknowledgment описывает DTO задачи на ознакомление.
type Acknowledgment struct {
	ID             string `json:"id"`
	DocumentID     string `json:"documentId"`
	DocumentKind   string `json:"documentKind"`
	DocumentNumber string `json:"documentNumber,omitempty"`

	CreatorID   string `json:"creatorId"`
	CreatorName string `json:"creatorName,omitempty"`

	Content     string     `json:"content"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	Users   []AcknowledgmentUser `json:"users,omitempty"`
	UserIDs []string             `json:"userIds,omitempty"`
}

// AcknowledgmentUser описывает DTO связи пользователя с задачей на ознакомление.
type AcknowledgmentUser struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	UserName    string     `json:"userName,omitempty"`
	ViewedAt    *time.Time `json:"viewedAt,omitempty"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// PagedResult описывает DTO постраничного результата.
type PagedResult[T any] struct {
	Items      []T `json:"items"`
	TotalCount int `json:"totalCount"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
}

// DashboardStats описывает DTO статистики для дашборда.
type DashboardStats struct {
	Role string `json:"role"`

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
	StorageObjects int    `json:"storageObjects,omitempty"`
	StorageSize    string `json:"storageSize,omitempty"`

	// Общий список (истекающие поручения)
	ExpiringAssignments []Assignment `json:"expiringAssignments,omitempty"`
}

// JournalEntry описывает DTO записи в журнале (истории) документа.
type JournalEntry struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"documentId"`
	UserName   string    `json:"userName,omitempty"`
	Action     string    `json:"action"`
	Details    string    `json:"details"`
	CreatedAt  time.Time `json:"createdAt"`
}

// AdminAuditLog описывает DTO записи журнала действий администраторов.
type AdminAuditLog struct {
	ID        string    `json:"id"`
	UserName  string    `json:"userName"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"createdAt"`
}

// AdminAuditLogPage описывает DTO страницы журнала действий администраторов.
type AdminAuditLogPage struct {
	Items []AdminAuditLog `json:"items"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
}
