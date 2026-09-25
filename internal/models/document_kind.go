package models

// DocumentKindAction описывает системное действие над документом.
type DocumentKindAction string

const (
	DocumentActionCreate      DocumentKindAction = "create"
	DocumentActionRead        DocumentKindAction = "read"
	DocumentActionUpdate      DocumentKindAction = "update"
	DocumentActionAssign      DocumentKindAction = "assign"
	DocumentActionAcknowledge DocumentKindAction = "acknowledge"
	DocumentActionUpload      DocumentKindAction = "upload"
	DocumentActionLink        DocumentKindAction = "link"
	DocumentActionViewJournal DocumentKindAction = "view_journal"
)

// DocumentKindSpec описывает системный вид документа и его метаданные.
type DocumentKindSpec struct {
	Code             DocumentKind         `json:"code"`
	Name             string               `json:"name"`
	SupportedActions []DocumentKindAction `json:"supportedActions"`
}

var documentKindSpecs = []DocumentKindSpec{
	{
		Code: DocumentKindIncomingLetter,
		Name: "Входящее письмо",
		SupportedActions: []DocumentKindAction{
			DocumentActionCreate,
			DocumentActionRead,
			DocumentActionUpdate,
			DocumentActionAssign,
			DocumentActionAcknowledge,
			DocumentActionUpload,
			DocumentActionLink,
			DocumentActionViewJournal,
		},
	},
	{
		Code: DocumentKindOutgoingLetter,
		Name: "Исходящее письмо",
		SupportedActions: []DocumentKindAction{
			DocumentActionCreate,
			DocumentActionRead,
			DocumentActionUpdate,
			DocumentActionAssign,
			DocumentActionAcknowledge,
			DocumentActionUpload,
			DocumentActionLink,
			DocumentActionViewJournal,
		},
	},
	{
		Code: DocumentKindCitizenAppeal,
		Name: "Обращения граждан",
		SupportedActions: []DocumentKindAction{
			DocumentActionCreate,
			DocumentActionRead,
			DocumentActionUpdate,
			DocumentActionAssign,
			DocumentActionAcknowledge,
			DocumentActionUpload,
			DocumentActionLink,
			DocumentActionViewJournal,
		},
	},
	{
		Code: DocumentKindAdministrativeOrder,
		Name: "Приказы",
		SupportedActions: []DocumentKindAction{
			DocumentActionCreate,
			DocumentActionRead,
			DocumentActionUpdate,
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

// SupportsAction проверяет, поддерживает ли вид документа указанное действие.
func (k DocumentKind) SupportsAction(action string) bool {
	spec, ok := GetDocumentKindSpec(k)
	if !ok {
		return false
	}

	for _, supportedAction := range spec.SupportedActions {
		if string(supportedAction) == action {
			return true
		}
	}

	return false
}

// Label возвращает человекочитаемое имя вида документа.
func (k DocumentKind) Label() string {
	spec, ok := GetDocumentKindSpec(k)
	if !ok {
		return string(k)
	}

	return spec.Name
}
