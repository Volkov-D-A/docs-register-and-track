package dto

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

// MapUserEvent преобразует событие пользователя в DTO.
func MapUserEvent(m *models.UserEvent) *UserEvent {
	if m == nil {
		return nil
	}
	actorUserID := ""
	if m.ActorUserID != nil {
		actorUserID = m.ActorUserID.String()
	}
	return &UserEvent{ID: m.ID.String(), ActorUserID: actorUserID, ActorUserName: m.ActorUserName, DocumentID: m.DocumentID.String(), DocumentKind: m.DocumentKind, DocumentNumber: m.DocumentNumber, EntityType: m.EntityType, EntityID: m.EntityID.String(), EventType: m.EventType, Title: m.Title, Message: m.Message, Metadata: m.Metadata, CreatedAt: m.CreatedAt, ReadAt: m.ReadAt}
}

func MapUserEvents(m []models.UserEvent) []UserEvent {
	if m == nil {
		return nil
	}
	res := make([]UserEvent, len(m))
	for i, v := range m {
		if mapped := MapUserEvent(&v); mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapJournalEntry(m *models.JournalEntry) *JournalEntry {
	if m == nil {
		return nil
	}
	return &JournalEntry{ID: m.ID.String(), DocumentID: m.DocumentID.String(), UserName: m.UserName, Action: m.Action, Details: m.Details, CreatedAt: m.CreatedAt}
}

func MapJournalEntries(m []models.JournalEntry) []JournalEntry {
	if m == nil {
		return []JournalEntry{}
	}
	res := make([]JournalEntry, len(m))
	for i, v := range m {
		if mapped := MapJournalEntry(&v); mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}

func MapAdminAuditLog(m *models.AdminAuditLog) *AdminAuditLog {
	if m == nil {
		return nil
	}
	return &AdminAuditLog{ID: m.ID.String(), UserName: m.UserName, Action: m.Action, Details: m.Details, CreatedAt: m.CreatedAt}
}

func MapAdminAuditLogs(m []models.AdminAuditLog) []AdminAuditLog {
	if m == nil {
		return []AdminAuditLog{}
	}
	res := make([]AdminAuditLog, len(m))
	for i, v := range m {
		if mapped := MapAdminAuditLog(&v); mapped != nil {
			res[i] = *mapped
		}
	}
	return res
}
