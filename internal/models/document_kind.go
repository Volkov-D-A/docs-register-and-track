package models

// DocumentKindAction описывает системное действие над документом.
type DocumentKindAction string

const (
	DocumentActionCreate      DocumentKindAction = "create"
	DocumentActionRead        DocumentKindAction = "read"
	DocumentActionUpdate      DocumentKindAction = "update"
	DocumentActionDelete      DocumentKindAction = "delete"
	DocumentActionAssign      DocumentKindAction = "assign"
	DocumentActionAcknowledge DocumentKindAction = "acknowledge"
	DocumentActionUpload      DocumentKindAction = "upload"
	DocumentActionLink        DocumentKindAction = "link"
	DocumentActionViewJournal DocumentKindAction = "view_journal"
)

// DocumentKindSpec описывает системный вид документа и его метаданные.
type DocumentKindSpec struct {
	Code                 DocumentKind         `json:"code"`
	Name                 string               `json:"name"`
	LegacyViewType       string               `json:"legacyViewType"`
	RegistrationFormCode string               `json:"registrationFormCode"`
	RegistryGroup        string               `json:"registryGroup"`
	SupportedActions     []DocumentKindAction `json:"supportedActions"`
}

var documentKindSpecs = []DocumentKindSpec{
	{
		Code:                 DocumentKindIncomingLetter,
		Name:                 "Входящее письмо",
		LegacyViewType:       "incoming",
		RegistrationFormCode: "incoming_letter_form",
		RegistryGroup:        "letters",
		SupportedActions: []DocumentKindAction{
			DocumentActionCreate,
			DocumentActionRead,
			DocumentActionUpdate,
			DocumentActionDelete,
			DocumentActionAssign,
			DocumentActionAcknowledge,
			DocumentActionUpload,
			DocumentActionLink,
			DocumentActionViewJournal,
		},
	},
	{
		Code:                 DocumentKindOutgoingLetter,
		Name:                 "Исходящее письмо",
		LegacyViewType:       "outgoing",
		RegistrationFormCode: "outgoing_letter_form",
		RegistryGroup:        "letters",
		SupportedActions: []DocumentKindAction{
			DocumentActionCreate,
			DocumentActionRead,
			DocumentActionUpdate,
			DocumentActionDelete,
			DocumentActionAssign,
			DocumentActionAcknowledge,
			DocumentActionUpload,
			DocumentActionLink,
			DocumentActionViewJournal,
		},
	},
}

// AllDocumentKindSpecs возвращает все системные виды документов.
func AllDocumentKindSpecs() []DocumentKindSpec {
	specs := make([]DocumentKindSpec, len(documentKindSpecs))
	copy(specs, documentKindSpecs)
	return specs
}

// GetDocumentKindSpec возвращает метаданные системного вида документа.
func GetDocumentKindSpec(kind DocumentKind) (DocumentKindSpec, bool) {
	for _, spec := range documentKindSpecs {
		if spec.Code == kind {
			return spec, true
		}
	}

	return DocumentKindSpec{}, false
}

// NormalizeDocumentKind приводит legacy- и системные коды к системному виду документа.
func NormalizeDocumentKind(kind string) DocumentKind {
	switch kind {
	case "incoming", string(DocumentKindIncomingLetter):
		return DocumentKindIncomingLetter
	case "outgoing", string(DocumentKindOutgoingLetter):
		return DocumentKindOutgoingLetter
	default:
		return DocumentKind(kind)
	}
}

// Label возвращает человекочитаемое имя вида документа.
func (k DocumentKind) Label() string {
	spec, ok := GetDocumentKindSpec(k)
	if !ok {
		return string(k)
	}

	return spec.Name
}
