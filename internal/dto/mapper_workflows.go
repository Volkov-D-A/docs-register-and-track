package dto

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

func MapDocumentLink(m *models.DocumentLink) *DocumentLink {
	if m == nil {
		return nil
	}
	return &DocumentLink{ID: m.ID.String(), SourceKind: string(m.SourceKind), SourceID: m.SourceID.String(), TargetKind: string(m.TargetKind), TargetID: m.TargetID.String(), LinkType: m.LinkType, CreatedBy: m.CreatedBy.String(), CreatedAt: m.CreatedAt, SourceNumber: m.SourceNumber, TargetNumber: m.TargetNumber, TargetSubject: m.TargetSubject}
}

func MapAttachment(m *models.Attachment) *Attachment {
	if m == nil {
		return nil
	}
	return &Attachment{ID: m.ID.String(), DocumentID: m.DocumentID.String(), Filename: m.Filename, Filepath: m.Filepath, FileSize: m.FileSize, ContentType: m.ContentType, UploadedBy: m.UploadedBy.String(), UploadedByName: m.UploadedByName, UploadedAt: m.UploadedAt}
}

func MapAssignment(m *models.Assignment) *Assignment {
	if m == nil {
		return nil
	}
	var coExecutors []User
	if m.CoExecutors != nil {
		coExecutors = make([]User, len(m.CoExecutors))
		for i, executor := range m.CoExecutors {
			if mapped := MapUser(&executor); mapped != nil {
				coExecutors[i] = *mapped
			}
		}
	}
	return &Assignment{ID: m.ID.String(), DocumentID: m.DocumentID.String(), DocumentKind: m.DocumentKind, ExecutorID: m.ExecutorID.String(), ExecutorName: m.ExecutorName, Content: m.Content, Deadline: m.Deadline, Status: m.Status, Report: m.Report, CompletedAt: m.CompletedAt, DocumentNumber: m.DocumentNumber, DocumentSubject: m.DocumentSubject, CoExecutors: coExecutors, CoExecutorIDs: m.CoExecutorIDs, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}

func MapAcknowledgment(m *models.Acknowledgment) *Acknowledgment {
	if m == nil {
		return nil
	}
	var users []AcknowledgmentUser
	if m.Users != nil {
		users = make([]AcknowledgmentUser, len(m.Users))
		for i, user := range m.Users {
			if mapped := MapAcknowledgmentUser(&user); mapped != nil {
				users[i] = *mapped
			}
		}
	}
	return &Acknowledgment{ID: m.ID.String(), DocumentID: m.DocumentID.String(), DocumentKind: m.DocumentKind, DocumentNumber: m.DocumentNumber, CreatorID: m.CreatorID.String(), CreatorName: m.CreatorName, Content: m.Content, CreatedAt: m.CreatedAt, CompletedAt: m.CompletedAt, Users: users, UserIDs: m.UserIDs}
}

func MapAcknowledgmentUser(m *models.AcknowledgmentUser) *AcknowledgmentUser {
	if m == nil {
		return nil
	}
	return &AcknowledgmentUser{ID: m.ID.String(), UserID: m.UserID.String(), UserName: m.UserName, ViewedAt: m.ViewedAt, ConfirmedAt: m.ConfirmedAt, CreatedAt: m.CreatedAt}
}

func MapDocumentLinks(m []models.DocumentLink) []DocumentLink {
	if m == nil {
		return nil
	}
	res := make([]DocumentLink, len(m))
	for i, item := range m {
		if mapped := MapDocumentLink(&item); mapped != nil {
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
	for i, item := range m {
		if mapped := MapAttachment(&item); mapped != nil {
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
	for i, item := range m {
		if mapped := MapAssignment(&item); mapped != nil {
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
	for i, item := range m {
		if mapped := MapAcknowledgment(&item); mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}
