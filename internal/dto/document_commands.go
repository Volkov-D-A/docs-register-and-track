package dto

// IncomingLetterRegisterRequest описывает команду регистрации входящего письма.
type IncomingLetterRegisterRequest struct {
	NomenclatureID       string                               `json:"nomenclatureId"`
	IdempotencyKey       string                               `json:"idempotencyKey"`
	DocumentTypeID       string                               `json:"documentTypeId"`
	IncomingDate         string                               `json:"incomingDate"`
	Correspondents       []IncomingLetterCorrespondentRequest `json:"correspondents"`
	Content              string                               `json:"content"`
	PagesCount           int                                  `json:"pagesCount"`
	AttachmentPagesCount int                                  `json:"attachmentPagesCount"`
	SenderSignatory      string                               `json:"senderSignatory"`
	Resolution           string                               `json:"resolution"`
	ResolutionAuthor     string                               `json:"resolutionAuthor"`
	ResolutionExecutors  string                               `json:"resolutionExecutors"`
	RegistrationNumber   string                               `json:"registrationNumber"`
	AdminNumberOverride  *AdminNumberOverrideRequest          `json:"adminNumberOverride"`
}

// IncomingLetterUpdateRequest описывает команду обновления входящего письма.
type IncomingLetterUpdateRequest struct {
	ID                   string                               `json:"id"`
	IdempotencyKey       string                               `json:"idempotencyKey,omitempty"`
	DocumentTypeID       string                               `json:"documentTypeId"`
	Correspondents       []IncomingLetterCorrespondentRequest `json:"correspondents"`
	Content              string                               `json:"content"`
	PagesCount           int                                  `json:"pagesCount"`
	AttachmentPagesCount int                                  `json:"attachmentPagesCount"`
	SenderSignatory      string                               `json:"senderSignatory"`
	Resolution           string                               `json:"resolution"`
	ResolutionAuthor     string                               `json:"resolutionAuthor"`
	ResolutionExecutors  string                               `json:"resolutionExecutors"`
}

// IncomingLetterCorrespondentRequest описывает один набор реквизитов корреспондента.
type IncomingLetterCorrespondentRequest struct {
	RegistrationNumber string `json:"registrationNumber"`
	RegistrationDate   string `json:"registrationDate"`
	CorrespondentName  string `json:"correspondentName"`
}

// OutgoingLetterRegisterRequest описывает команду регистрации исходящего письма.
type OutgoingLetterRegisterRequest struct {
	NomenclatureID       string                      `json:"nomenclatureId"`
	IdempotencyKey       string                      `json:"idempotencyKey"`
	DocumentTypeID       string                      `json:"documentTypeId"`
	RecipientOrgName     string                      `json:"recipientOrgName"`
	Addressee            string                      `json:"addressee"`
	OutgoingDate         string                      `json:"outgoingDate"`
	Content              string                      `json:"content"`
	PagesCount           int                         `json:"pagesCount"`
	AttachmentPagesCount int                         `json:"attachmentPagesCount"`
	SenderSignatory      string                      `json:"senderSignatory"`
	SenderExecutor       string                      `json:"senderExecutor"`
	RegistrationNumber   string                      `json:"registrationNumber"`
	AdminNumberOverride  *AdminNumberOverrideRequest `json:"adminNumberOverride"`
}

// OutgoingLetterUpdateRequest описывает команду обновления исходящего письма.
type OutgoingLetterUpdateRequest struct {
	ID                   string `json:"id"`
	IdempotencyKey       string `json:"idempotencyKey,omitempty"`
	DocumentTypeID       string `json:"documentTypeId"`
	RecipientOrgName     string `json:"recipientOrgName"`
	Addressee            string `json:"addressee"`
	OutgoingDate         string `json:"outgoingDate"`
	Content              string `json:"content"`
	PagesCount           int    `json:"pagesCount"`
	AttachmentPagesCount int    `json:"attachmentPagesCount"`
	SenderSignatory      string `json:"senderSignatory"`
	SenderExecutor       string `json:"senderExecutor"`
}

// CitizenAppealRegisterRequest описывает команду регистрации обращения граждан.
type CitizenAppealRegisterRequest struct {
	NomenclatureID       string                              `json:"nomenclatureId"`
	IdempotencyKey       string                              `json:"idempotencyKey"`
	RegistrationDate     string                              `json:"registrationDate"`
	AppealDate           string                              `json:"appealDate"`
	ApplicantFullName    string                              `json:"applicantFullName"`
	RegistrationAddress  string                              `json:"registrationAddress"`
	AppealType           string                              `json:"appealType"`
	ApplicantCategory    string                              `json:"applicantCategory"`
	PagesCount           int                                 `json:"pagesCount"`
	AttachmentPagesCount int                                 `json:"attachmentPagesCount"`
	HasEnvelope          bool                                `json:"hasEnvelope"`
	ReceivedFromPOS      bool                                `json:"receivedFromPos"`
	Content              string                              `json:"content"`
	RegistrationNumber   string                              `json:"registrationNumber"`
	AdminNumberOverride  *AdminNumberOverrideRequest         `json:"adminNumberOverride"`
	Correspondents       []CitizenAppealCorrespondentRequest `json:"correspondents"`
	Resolutions          []CitizenAppealResolutionRequest    `json:"resolutions"`
}

// CitizenAppealUpdateRequest описывает команду обновления обращения граждан.
type CitizenAppealUpdateRequest struct {
	ID                   string                              `json:"id"`
	IdempotencyKey       string                              `json:"idempotencyKey,omitempty"`
	RegistrationNumber   string                              `json:"registrationNumber"`
	RegistrationDate     string                              `json:"registrationDate"`
	AppealDate           string                              `json:"appealDate"`
	ApplicantFullName    string                              `json:"applicantFullName"`
	RegistrationAddress  string                              `json:"registrationAddress"`
	AppealType           string                              `json:"appealType"`
	ApplicantCategory    string                              `json:"applicantCategory"`
	PagesCount           int                                 `json:"pagesCount"`
	AttachmentPagesCount int                                 `json:"attachmentPagesCount"`
	HasEnvelope          bool                                `json:"hasEnvelope"`
	ReceivedFromPOS      bool                                `json:"receivedFromPos"`
	Content              string                              `json:"content"`
	Correspondents       []CitizenAppealCorrespondentRequest `json:"correspondents"`
	Resolutions          []CitizenAppealResolutionRequest    `json:"resolutions"`
}

// CitizenAppealCorrespondentRequest описывает один набор внешних регистрационных реквизитов.
type CitizenAppealCorrespondentRequest struct {
	RegistrationNumber string `json:"registrationNumber"`
	RegistrationDate   string `json:"registrationDate"`
	CorrespondentName  string `json:"correspondentName"`
}

// CitizenAppealResolutionRequest описывает один набор резолюции.
type CitizenAppealResolutionRequest struct {
	Resolution          string `json:"resolution"`
	ResolutionAuthor    string `json:"resolutionAuthor"`
	ResolutionExecutors string `json:"resolutionExecutors"`
}

// AdministrativeOrderRegisterRequest описывает команду регистрации приказа.
type AdministrativeOrderRegisterRequest struct {
	NomenclatureID          string                      `json:"nomenclatureId"`
	IdempotencyKey          string                      `json:"idempotencyKey"`
	OrderDate               string                      `json:"orderDate"`
	Title                   string                      `json:"title"`
	PagesCount              int                         `json:"pagesCount"`
	ExecutionController     string                      `json:"executionController"`
	ExecutionDeadline       string                      `json:"executionDeadline"`
	IsActive                bool                        `json:"isActive"`
	CancelledAt             string                      `json:"cancelledAt"`
	AcknowledgmentFullNames []string                    `json:"acknowledgmentFullNames"`
	RegistrationNumber      string                      `json:"registrationNumber"`
	AdminNumberOverride     *AdminNumberOverrideRequest `json:"adminNumberOverride"`
}

// AdministrativeOrderUpdateRequest описывает команду обновления приказа.
type AdministrativeOrderUpdateRequest struct {
	ID                      string   `json:"id"`
	IdempotencyKey          string   `json:"idempotencyKey,omitempty"`
	OrderDate               string   `json:"orderDate"`
	Title                   string   `json:"title"`
	PagesCount              int      `json:"pagesCount"`
	ExecutionController     string   `json:"executionController"`
	ExecutionDeadline       string   `json:"executionDeadline"`
	IsActive                bool     `json:"isActive"`
	CancelledAt             string   `json:"cancelledAt"`
	AcknowledgmentFullNames []string `json:"acknowledgmentFullNames"`
}

type AdminNumberOverrideRequest struct {
	Mode   string `json:"mode"`
	Number int    `json:"number"`
	Suffix string `json:"suffix"`
}

type AdminDraftCreateRequest struct {
	NomenclatureID      string                      `json:"nomenclatureId"`
	RegistrationDate    string                      `json:"registrationDate"`
	AdminNumberOverride *AdminNumberOverrideRequest `json:"adminNumberOverride"`
	IdempotencyKey      string                      `json:"idempotencyKey,omitempty"`
}
