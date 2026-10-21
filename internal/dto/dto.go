package dto

import "time"

// User описывает DTO пользователя.
type User struct {
	ID                    string      `json:"id"`
	Login                 string      `json:"login"`
	FullName              string      `json:"fullName"`
	LastName              string      `json:"lastName"`
	FirstName             string      `json:"firstName"`
	Patronymic            string      `json:"patronymic"`
	NoPatronymic          bool        `json:"noPatronymic"`
	IsDocumentParticipant bool        `json:"isDocumentParticipant"`
	IsActive              bool        `json:"isActive"`
	FailedLoginAttempts   int         `json:"failedLoginAttempts"`
	TemporaryPassword     string      `json:"temporaryPassword,omitempty"`
	SystemPermissions     []string    `json:"systemPermissions"`
	Department            *Department `json:"department,omitempty"`
}

// Department описывает DTO подразделения.
type Department struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	NomenclatureIDs []string `json:"nomenclatureIds"`
}

// Nomenclature описывает DTO номенклатуры дел.
type Nomenclature struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Index         string `json:"index"`
	Year          int    `json:"year"`
	KindCode      string `json:"kindCode"`
	Separator     string `json:"separator"`
	NumberingMode string `json:"numberingMode"`
	NextNumber    int    `json:"nextNumber"`
	IsActive      bool   `json:"isActive"`
}

// Organization описывает DTO организации.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CurrentAccessSummary описывает текущие права пользователя для навигации и UI.
type CurrentAccessSummary struct {
	Sections          AccessSections              `json:"sections"`
	DocumentKinds     []DocumentKindAccessSummary `json:"documentKinds"`
	SystemPermissions []string                    `json:"systemPermissions"`
}

// AccessSections описывает доступность основных разделов приложения.
type AccessSections struct {
	Dashboard  bool `json:"dashboard"`
	Incoming   bool `json:"incoming"`
	Outgoing   bool `json:"outgoing"`
	Appeals    bool `json:"appeals"`
	Orders     bool `json:"orders"`
	Controls   bool `json:"controls"`
	References bool `json:"references"`
	Statistics bool `json:"statistics"`
	Reports    bool `json:"reports"`
	Settings   bool `json:"settings"`
}

// DocumentKindAccessSummary описывает доступность действий для конкретного вида документа.
type DocumentKindAccessSummary struct {
	Code             string   `json:"code"`
	Name             string   `json:"name"`
	AvailableActions []string `json:"availableActions"`
}

// ResolutionExecutor описывает DTO исполнителя резолюции.
type ResolutionExecutor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Новые структуры для ответов (DTO)

// IncomingDocument описывает DTO входящего документа.
type IncomingDocument struct {
	ID               string `json:"id"`
	NomenclatureID   string `json:"nomenclatureId"`
	NomenclatureName string `json:"nomenclatureName,omitempty"`

	IncomingNumber string    `json:"incomingNumber"`
	IncomingDate   time.Time `json:"incomingDate"`

	DocumentTypeID       string `json:"documentTypeId"`
	Content              string `json:"content"`
	PagesCount           int    `json:"pagesCount"`
	AttachmentPagesCount int    `json:"attachmentPagesCount"`

	Correspondents []DocumentCorrespondentRegistration `json:"correspondents,omitempty"`

	SenderSignatory string `json:"senderSignatory"`

	Resolution          *string   `json:"resolution,omitempty"`
	ResolutionAuthor    *string   `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors *string   `json:"resolutionExecutors,omitempty"`
	CreatedByName       string    `json:"createdByName,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// DocumentCorrespondentRegistration описывает DTO регистрационных реквизитов корреспондента.
type DocumentCorrespondentRegistration struct {
	ID                 string    `json:"id"`
	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	CorrespondentName  string    `json:"correspondentName,omitempty"`
	Position           int       `json:"position"`
}

// DocumentResolution описывает DTO набора резолюции документа.
type DocumentResolution struct {
	ID                  string  `json:"id"`
	Resolution          *string `json:"resolution,omitempty"`
	ResolutionAuthor    *string `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors *string `json:"resolutionExecutors,omitempty"`
	Position            int     `json:"position"`
}

// CitizenAppealDocument описывает DTO обращения граждан.
type CitizenAppealDocument struct {
	ID               string `json:"id"`
	NomenclatureID   string `json:"nomenclatureId"`
	NomenclatureName string `json:"nomenclatureName,omitempty"`

	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	AppealDate         time.Time `json:"appealDate"`

	DocumentTypeID string `json:"documentTypeId"`
	Content        string `json:"content"`
	PagesCount     int    `json:"pagesCount"`

	ApplicantFullName    string `json:"applicantFullName"`
	RegistrationAddress  string `json:"registrationAddress"`
	AppealType           string `json:"appealType"`
	ApplicantCategory    string `json:"applicantCategory"`
	AttachmentPagesCount int    `json:"attachmentPagesCount"`
	HasEnvelope          bool   `json:"hasEnvelope"`
	ReceivedFromPOS      bool   `json:"receivedFromPos"`

	Correspondents []DocumentCorrespondentRegistration `json:"correspondents,omitempty"`
	Resolutions    []DocumentResolution                `json:"resolutions,omitempty"`
	CreatedByName  string                              `json:"createdByName,omitempty"`
	CreatedAt      time.Time                           `json:"createdAt"`
	UpdatedAt      time.Time                           `json:"updatedAt"`
}

// OutgoingDocument описывает DTO исходящего документа.
type OutgoingDocument struct {
	ID               string `json:"id"`
	NomenclatureID   string `json:"nomenclatureId"`
	NomenclatureName string `json:"nomenclatureName,omitempty"`

	OutgoingNumber string    `json:"outgoingNumber"`
	OutgoingDate   time.Time `json:"outgoingDate"`

	DocumentTypeID       string `json:"documentTypeId"`
	Content              string `json:"content"`
	PagesCount           int    `json:"pagesCount"`
	AttachmentPagesCount int    `json:"attachmentPagesCount"`

	SenderSignatory  string    `json:"senderSignatory"`
	SenderExecutor   string    `json:"senderExecutor"`
	RecipientOrgName string    `json:"recipientOrgName,omitempty"`
	Addressee        string    `json:"addressee"`
	CreatedByName    string    `json:"createdByName,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// AdministrativeOrderDocument описывает DTO приказа.
type AdministrativeOrderDocument struct {
	ID               string `json:"id"`
	NomenclatureID   string `json:"nomenclatureId"`
	NomenclatureName string `json:"nomenclatureName,omitempty"`

	OrderNumber string    `json:"orderNumber"`
	OrderDate   time.Time `json:"orderDate"`
	Title       string    `json:"title"`

	DocumentTypeID string `json:"documentTypeId"`
	Content        string `json:"content"`
	PagesCount     int    `json:"pagesCount"`

	ExecutionController string     `json:"executionController"`
	ExecutionDeadline   *time.Time `json:"executionDeadline,omitempty"`
	IsActive            bool       `json:"isActive"`
	CancelledAt         *time.Time `json:"cancelledAt,omitempty"`

	AcknowledgmentPeople []AdministrativeOrderAcknowledgmentPerson `json:"acknowledgmentPeople,omitempty"`
	CreatedByName        string                                    `json:"createdByName,omitempty"`
	CreatedAt            time.Time                                 `json:"createdAt"`
	UpdatedAt            time.Time                                 `json:"updatedAt"`
}

// AdministrativeOrderAcknowledgmentPerson описывает DTO строки листа ознакомления приказа.
type AdministrativeOrderAcknowledgmentPerson struct {
	ID                 string     `json:"id"`
	FullName           string     `json:"fullName"`
	AcknowledgedAt     *time.Time `json:"acknowledgedAt,omitempty"`
	AcknowledgedByName string     `json:"acknowledgedByName,omitempty"`
	Position           int        `json:"position"`
	CreatedAt          time.Time  `json:"createdAt"`
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
	Content            string    `json:"content"`
	CreatedByName      string    `json:"createdByName,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`

	IncomingLetter      *IncomingDocument            `json:"incomingLetter,omitempty"`
	OutgoingLetter      *OutgoingDocument            `json:"outgoingLetter,omitempty"`
	CitizenAppeal       *CitizenAppealDocument       `json:"citizenAppeal,omitempty"`
	AdministrativeOrder *AdministrativeOrderDocument `json:"administrativeOrder,omitempty"`
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
	Content            string    `json:"content"`
	PagesCount         int       `json:"pagesCount"`
	CreatedByName      string    `json:"createdByName,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`

	IncomingNumber              string                                    `json:"incomingNumber,omitempty"`
	IncomingDate                *time.Time                                `json:"incomingDate,omitempty"`
	AppealDate                  *time.Time                                `json:"appealDate,omitempty"`
	OutgoingNumber              string                                    `json:"outgoingNumber,omitempty"`
	OutgoingDate                *time.Time                                `json:"outgoingDate,omitempty"`
	Correspondents              []DocumentCorrespondentRegistration       `json:"correspondents,omitempty"`
	SenderSignatory             string                                    `json:"senderSignatory,omitempty"`
	Resolution                  *string                                   `json:"resolution,omitempty"`
	ResolutionAuthor            *string                                   `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors         *string                                   `json:"resolutionExecutors,omitempty"`
	Resolutions                 []DocumentResolution                      `json:"resolutions,omitempty"`
	RecipientOrgName            string                                    `json:"recipientOrgName,omitempty"`
	Addressee                   string                                    `json:"addressee,omitempty"`
	SenderExecutor              string                                    `json:"senderExecutor,omitempty"`
	ApplicantFullName           string                                    `json:"applicantFullName,omitempty"`
	RegistrationAddress         string                                    `json:"registrationAddress,omitempty"`
	AppealType                  string                                    `json:"appealType,omitempty"`
	ApplicantCategory           string                                    `json:"applicantCategory,omitempty"`
	AttachmentPagesCount        int                                       `json:"attachmentPagesCount,omitempty"`
	HasEnvelope                 bool                                      `json:"hasEnvelope,omitempty"`
	ReceivedFromPOS             bool                                      `json:"receivedFromPos,omitempty"`
	OrderNumber                 string                                    `json:"orderNumber,omitempty"`
	OrderDate                   *time.Time                                `json:"orderDate,omitempty"`
	Title                       string                                    `json:"title,omitempty"`
	ExecutionController         string                                    `json:"executionController,omitempty"`
	ExecutionDeadline           *time.Time                                `json:"executionDeadline,omitempty"`
	IsActive                    bool                                      `json:"isActive"`
	CancelledAt                 *time.Time                                `json:"cancelledAt,omitempty"`
	PendingAcknowledgmentsCount int                                       `json:"pendingAcknowledgmentsCount,omitempty"`
	AcknowledgmentPeople        []AdministrativeOrderAcknowledgmentPerson `json:"acknowledgmentPeople,omitempty"`
}

// DocumentLink описывает DTO связи между документами.
type DocumentLink struct {
	ID         string `json:"id"`
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId"`
	TargetKind string `json:"targetKind"`
	LinkType   string `json:"linkType"`

	SourceNumber  string `json:"sourceNumber,omitempty"`
	TargetNumber  string `json:"targetNumber,omitempty"`
	SourceSubject string `json:"sourceSubject,omitempty"`
	TargetSubject string `json:"targetSubject,omitempty"`
}

// Attachment описывает DTO прикрепленного файла.
// PagedResult описывает DTO постраничного результата.
type PagedResult[T any] struct {
	Items      []T    `json:"items"`
	TotalCount int    `json:"totalCount"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

// UserEvent описывает DTO персонального события пользователя.
type UserEvent struct {
	ID             string     `json:"id"`
	DocumentID     string     `json:"documentId"`
	DocumentKind   string     `json:"documentKind"`
	DocumentNumber string     `json:"documentNumber,omitempty"`
	DocumentDate   *time.Time `json:"documentDate,omitempty"`
	EntityType     string     `json:"entityType"`
	EventType      string     `json:"eventType"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

// JournalEntry описывает DTO записи в журнале (истории) документа.
type JournalEntry struct {
	ID        string    `json:"id"`
	UserName  string    `json:"userName,omitempty"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"createdAt"`
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

// WorkspaceDocuments contains only the four latest readable registrations.
type WorkspaceDocuments struct {
	Available bool                `json:"available"`
	Items     []WorkspaceDocument `json:"items"`
}

type WorkspaceDocument struct {
	ID             string    `json:"id"`
	DocumentKind   string    `json:"documentKind"`
	DocumentNumber string    `json:"documentNumber"`
	DocumentDate   time.Time `json:"documentDate"`
	RegisteredAt   time.Time `json:"registeredAt"`
	Description    string    `json:"description"`
	Correspondents []string  `json:"correspondents"`
}
