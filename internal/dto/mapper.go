package dto

import (
	"docflow/internal/models"
)

func MapUser(m *models.User) *User {
	if m == nil {
		return nil
	}
	var dept *Department
	if m.Department != nil {
		dept = MapDepartment(m.Department)
	}
	return &User{
		ID:         m.ID.String(),
		Login:      m.Login,
		FullName:   m.FullName,
		IsActive:   m.IsActive,
		Roles:      m.Roles,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		Department: dept,
	}
}

func MapDepartment(m *models.Department) *Department {
	if m == nil {
		return nil
	}
	var noms []Nomenclature
	if m.Nomenclature != nil {
		noms = make([]Nomenclature, len(m.Nomenclature))
		for i, n := range m.Nomenclature {
			noms[i] = *MapNomenclature(&n)
		}
	}
	return &Department{
		ID:              m.ID.String(),
		Name:            m.Name,
		NomenclatureIDs: m.NomenclatureIDs,
		Nomenclature:    noms,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func MapNomenclature(m *models.Nomenclature) *Nomenclature {
	if m == nil {
		return nil
	}
	return &Nomenclature{
		ID:         m.ID.String(),
		Name:       m.Name,
		Index:      m.Index,
		Year:       m.Year,
		Direction:  m.Direction,
		NextNumber: m.NextNumber,
		IsActive:   m.IsActive,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func MapOrganization(m *models.Organization) *Organization {
	if m == nil {
		return nil
	}
	return &Organization{
		ID:        m.ID.String(),
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
	}
}

func MapDocumentType(m *models.DocumentType) *DocumentType {
	if m == nil {
		return nil
	}
	return &DocumentType{
		ID:        m.ID.String(),
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
	}
}

func MapIncomingDocument(m *models.IncomingDocument) *IncomingDocument {
	if m == nil {
		return nil
	}
	return &IncomingDocument{
		ID:                   m.ID.String(),
		NomenclatureID:       m.NomenclatureID.String(),
		NomenclatureName:     m.NomenclatureName,
		IncomingNumber:       m.IncomingNumber,
		IncomingDate:         m.IncomingDate,
		OutgoingNumberSender: m.OutgoingNumberSender,
		OutgoingDateSender:   m.OutgoingDateSender,
		IntermediateNumber:   m.IntermediateNumber,
		IntermediateDate:     m.IntermediateDate,
		DocumentTypeID:       m.DocumentTypeID.String(),
		DocumentTypeName:     m.DocumentTypeName,
		Subject:              m.Subject,
		PagesCount:           m.PagesCount,
		Content:              m.Content,
		SenderOrgID:          m.SenderOrgID.String(),
		SenderOrgName:        m.SenderOrgName,
		SenderSignatory:      m.SenderSignatory,
		SenderExecutor:       m.SenderExecutor,
		RecipientOrgID:       m.RecipientOrgID.String(),
		RecipientOrgName:     m.RecipientOrgName,
		Addressee:            m.Addressee,
		Resolution:           m.Resolution,
		CreatedBy:            m.CreatedBy.String(),
		CreatedByName:        m.CreatedByName,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
		AttachmentsCount:     m.AttachmentsCount,
		AssignmentsCount:     m.AssignmentsCount,
	}
}

func MapOutgoingDocument(m *models.OutgoingDocument) *OutgoingDocument {
	if m == nil {
		return nil
	}
	return &OutgoingDocument{
		ID:               m.ID.String(),
		NomenclatureID:   m.NomenclatureID.String(),
		NomenclatureName: m.NomenclatureName,
		OutgoingNumber:   m.OutgoingNumber,
		OutgoingDate:     m.OutgoingDate,
		DocumentTypeID:   m.DocumentTypeID.String(),
		DocumentTypeName: m.DocumentTypeName,
		Subject:          m.Subject,
		PagesCount:       m.PagesCount,
		Content:          m.Content,
		SenderOrgID:      m.SenderOrgID.String(),
		SenderOrgName:    m.SenderOrgName,
		SenderSignatory:  m.SenderSignatory,
		SenderExecutor:   m.SenderExecutor,
		RecipientOrgID:   m.RecipientOrgID.String(),
		RecipientOrgName: m.RecipientOrgName,
		Addressee:        m.Addressee,
		CreatedBy:        m.CreatedBy.String(),
		CreatedByName:    m.CreatedByName,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		AttachmentsCount: m.AttachmentsCount,
	}
}

func MapDocumentLink(m *models.DocumentLink) *DocumentLink {
	if m == nil {
		return nil
	}
	return &DocumentLink{
		ID:            m.ID.String(),
		SourceType:    m.SourceType,
		SourceID:      m.SourceID.String(),
		TargetType:    m.TargetType,
		TargetID:      m.TargetID.String(),
		LinkType:      m.LinkType,
		CreatedBy:     m.CreatedBy.String(),
		CreatedAt:     m.CreatedAt,
		SourceNumber:  m.SourceNumber,
		TargetNumber:  m.TargetNumber,
		TargetSubject: m.TargetSubject,
	}
}

func MapAttachment(m *models.Attachment) *Attachment {
	if m == nil {
		return nil
	}
	return &Attachment{
		ID:             m.ID.String(),
		DocumentID:     m.DocumentID.String(),
		DocumentType:   m.DocumentType,
		Filename:       m.Filename,
		Filepath:       m.Filepath,
		FileSize:       m.FileSize,
		ContentType:    m.ContentType,
		UploadedBy:     m.UploadedBy.String(),
		UploadedByName: m.UploadedByName,
		UploadedAt:     m.UploadedAt,
	}
}

func MapAssignment(m *models.Assignment) *Assignment {
	if m == nil {
		return nil
	}
	var coExecutors []User
	if m.CoExecutors != nil {
		coExecutors = make([]User, len(m.CoExecutors))
		for i, c := range m.CoExecutors {
			mapped := MapUser(&c)
			if mapped != nil {
				coExecutors[i] = *mapped
			}
		}
	}
	return &Assignment{
		ID:              m.ID.String(),
		DocumentID:      m.DocumentID.String(),
		DocumentType:    m.DocumentType,
		ExecutorID:      m.ExecutorID.String(),
		ExecutorName:    m.ExecutorName,
		Content:         m.Content,
		Deadline:        m.Deadline,
		Status:          m.Status,
		Report:          m.Report,
		CompletedAt:     m.CompletedAt,
		DocumentNumber:  m.DocumentNumber,
		DocumentSubject: m.DocumentSubject,
		CoExecutors:     coExecutors,
		CoExecutorIDs:   m.CoExecutorIDs,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func MapAcknowledgment(m *models.Acknowledgment) *Acknowledgment {
	if m == nil {
		return nil
	}
	var users []AcknowledgmentUser
	if m.Users != nil {
		users = make([]AcknowledgmentUser, len(m.Users))
		for i, u := range m.Users {
			mapped := MapAcknowledgmentUser(&u)
			if mapped != nil {
				users[i] = *mapped
			}
		}
	}
	return &Acknowledgment{
		ID:             m.ID.String(),
		DocumentID:     m.DocumentID.String(),
		DocumentType:   m.DocumentType,
		DocumentNumber: m.DocumentNumber,
		CreatorID:      m.CreatorID.String(),
		CreatorName:    m.CreatorName,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt,
		CompletedAt:    m.CompletedAt,
		Users:          users,
		UserIDs:        m.UserIDs,
	}
}

func MapAcknowledgmentUser(m *models.AcknowledgmentUser) *AcknowledgmentUser {
	if m == nil {
		return nil
	}
	return &AcknowledgmentUser{
		ID:          m.ID.String(),
		UserID:      m.UserID.String(),
		UserName:    m.UserName,
		ViewedAt:    m.ViewedAt,
		ConfirmedAt: m.ConfirmedAt,
		CreatedAt:   m.CreatedAt,
	}
}

// Функции-мапперы для слайсов

func MapUsers(m []models.User) []User {
	if m == nil {
		return nil
	}
	res := make([]User, len(m))
	for i, v := range m {
		mapped := MapUser(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapDepartments(m []models.Department) []Department {
	if m == nil {
		return nil
	}
	res := make([]Department, len(m))
	for i, v := range m {
		mapped := MapDepartment(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapNomenclatures(m []models.Nomenclature) []Nomenclature {
	if m == nil {
		return nil
	}
	res := make([]Nomenclature, len(m))
	for i, v := range m {
		mapped := MapNomenclature(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapOrganizations(m []models.Organization) []Organization {
	if m == nil {
		return nil
	}
	res := make([]Organization, len(m))
	for i, v := range m {
		mapped := MapOrganization(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapDocumentTypes(m []models.DocumentType) []DocumentType {
	if m == nil {
		return nil
	}
	res := make([]DocumentType, len(m))
	for i, v := range m {
		mapped := MapDocumentType(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapIncomingDocuments(m []models.IncomingDocument) []IncomingDocument {
	if m == nil {
		return nil
	}
	res := make([]IncomingDocument, len(m))
	for i, v := range m {
		mapped := MapIncomingDocument(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapOutgoingDocuments(m []models.OutgoingDocument) []OutgoingDocument {
	if m == nil {
		return nil
	}
	res := make([]OutgoingDocument, len(m))
	for i, v := range m {
		mapped := MapOutgoingDocument(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapDocumentLinks(m []models.DocumentLink) []DocumentLink {
	if m == nil {
		return nil
	}
	res := make([]DocumentLink, len(m))
	for i, v := range m {
		mapped := MapDocumentLink(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapAttachments(m []models.Attachment) []Attachment {
	if m == nil {
		return nil
	}
	res := make([]Attachment, len(m))
	for i, v := range m {
		mapped := MapAttachment(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapAssignments(m []models.Assignment) []Assignment {
	if m == nil {
		return nil
	}
	res := make([]Assignment, len(m))
	for i, v := range m {
		mapped := MapAssignment(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapAcknowledgments(m []models.Acknowledgment) []Acknowledgment {
	if m == nil {
		return nil
	}
	res := make([]Acknowledgment, len(m))
	for i, v := range m {
		mapped := MapAcknowledgment(&v)
		if mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapDashboardStats(m *models.DashboardStats) *DashboardStats {
	if m == nil {
		return nil
	}
	return &DashboardStats{
		Role:                       m.Role,
		MyAssignmentsNew:           m.MyAssignmentsNew,
		MyAssignmentsInProgress:    m.MyAssignmentsInProgress,
		MyAssignmentsOverdue:       m.MyAssignmentsOverdue,
		MyAssignmentsFinished:      m.MyAssignmentsFinished,
		MyAssignmentsFinishedLate:  m.MyAssignmentsFinishedLate,
		IncomingCount:              m.IncomingCount,
		OutgoingCount:              m.OutgoingCount,
		AllAssignmentsOverdue:      m.AllAssignmentsOverdue,
		AllAssignmentsFinished:     m.AllAssignmentsFinished,
		AllAssignmentsFinishedLate: m.AllAssignmentsFinishedLate,
		UserCount:                  m.UserCount,
		TotalDocuments:             m.TotalDocuments,
		DBSize:                     m.DBSize,
		ExpiringAssignments:        MapAssignments(m.ExpiringAssignments),
	}
}
