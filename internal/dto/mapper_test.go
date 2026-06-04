package dto

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapUser(t *testing.T) {
	// Тестирование маппинга модели пользователя в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapUser(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.User{ID: id, Login: "test", FullName: "Test User", IsActive: true, SystemPermissions: []string{"admin"}}
		d := MapUser(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "test", d.Login)
		assert.Equal(t, "Test User", d.FullName)
		assert.True(t, d.IsActive)
	})
}

func TestMapDepartment(t *testing.T) {
	// Тестирование маппинга подразделения в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapDepartment(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		nomID := uuid.New()
		now := time.Now()
		m := &models.Department{
			ID:              id,
			Name:            "IT",
			NomenclatureIDs: []string{nomID.String()},
			Nomenclature: []models.Nomenclature{
				{ID: nomID, Name: "Cases", Index: "01-01"},
			},
			CreatedAt: now,
		}
		d := MapDepartment(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "IT", d.Name)
		assert.Equal(t, []string{nomID.String()}, d.NomenclatureIDs)
		assert.Len(t, d.Nomenclature, 1)
		assert.Equal(t, "Cases", d.Nomenclature[0].Name)
	})
}

func TestMapNomenclature(t *testing.T) {
	// Тестирование маппинга номенклатуры дел в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapNomenclature(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.Nomenclature{ID: id, Name: "N", Index: "1", Year: 2024, KindCode: "incoming_letter", Separator: "/", NumberingMode: "index_and_number", NextNumber: 5, IsActive: true}
		d := MapNomenclature(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "N", d.Name)
		assert.Equal(t, 2024, d.Year)
		assert.Equal(t, 5, d.NextNumber)
	})
}

func TestMapOrganization(t *testing.T) {
	// Тестирование маппинга организации (корреспондента) в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapOrganization(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.Organization{ID: id, Name: "Org"}
		d := MapOrganization(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "Org", d.Name)
	})
}

func TestMapDocumentType(t *testing.T) {
	// Тестирование маппинга типа документа в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapDocumentType(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.DocumentType{ID: id, Name: "Type"}
		d := MapDocumentType(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "Type", d.Name)
	})
}

func TestMapDocumentLink(t *testing.T) {
	// Тестирование маппинга связи между документами в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapDocumentLink(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.DocumentLink{ID: id, LinkType: "reply"}
		d := MapDocumentLink(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "reply", d.LinkType)
	})
}

func TestMapAttachment(t *testing.T) {
	// Тестирование маппинга вложения файла в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAttachment(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.Attachment{ID: id, Filename: "f.txt"}
		d := MapAttachment(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "f.txt", d.Filename)
	})
}

func TestMapAcknowledgmentUser(t *testing.T) {
	// Тестирование маппинга записи ознакомления конкретного пользователя в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAcknowledgmentUser(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.AcknowledgmentUser{ID: id, UserName: "U"}
		d := MapAcknowledgmentUser(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "U", d.UserName)
	})
}

func TestMapAssignment(t *testing.T) {
	// Тестирование маппинга поручения (резолюции) в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAssignment(nil))
	})

	t.Run("success with co-executors", func(t *testing.T) {
		id := uuid.New()
		coExec := models.User{ID: uuid.New(), Login: "co", FullName: "CoExec"}
		m := &models.Assignment{
			ID:          id,
			DocumentID:  uuid.New(),
			ExecutorID:  uuid.New(),
			Content:     "Do it",
			Status:      "new",
			CoExecutors: []models.User{coExec},
		}
		d := MapAssignment(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Len(t, d.CoExecutors, 1)
		assert.Equal(t, "CoExec", d.CoExecutors[0].FullName)
	})
}

func TestMapAcknowledgment(t *testing.T) {
	// Тестирование маппинга листа ознакомления в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAcknowledgment(nil))
	})

	t.Run("success with users", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()
		u := models.AcknowledgmentUser{ID: uuid.New(), UserID: uuid.New(), UserName: "Tester", CreatedAt: now}
		m := &models.Acknowledgment{ID: id, DocumentID: uuid.New(), CreatorID: uuid.New(), Users: []models.AcknowledgmentUser{u}}
		d := MapAcknowledgment(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Len(t, d.Users, 1)
		assert.Equal(t, "Tester", d.Users[0].UserName)
	})
}

func TestMapIncomingDocument(t *testing.T) {
	// Тестирование маппинга входящего документа в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapIncomingDocument(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.IncomingDocument{ID: id, IncomingNumber: "01-01/1", Content: "Test"}
		d := MapIncomingDocument(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "01-01/1", d.IncomingNumber)
	})
}

func TestMapOutgoingDocument(t *testing.T) {
	// Тестирование маппинга исходящего документа в DTO
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapOutgoingDocument(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.OutgoingDocument{ID: id, OutgoingNumber: "02-01/1", Content: "Test"}
		d := MapOutgoingDocument(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "02-01/1", d.OutgoingNumber)
		assert.Equal(t, "Test", d.Content)
	})
}

func TestMapDocumentListItemsPagesCount(t *testing.T) {
	t.Run("incoming", func(t *testing.T) {
		assert.Nil(t, MapIncomingDocumentListItem(nil))

		incomingDate := time.Now()
		correspondentID := uuid.New()
		resolution := "Исполнить"
		resolutionAuthor := "Руководитель"
		resolutionExecutors := "Исполнитель"
		item := MapIncomingDocumentListItem(&models.IncomingDocument{
			ID:                  uuid.New(),
			NomenclatureID:      uuid.New(),
			NomenclatureName:    "01-01",
			IncomingNumber:      "IN-12",
			IncomingDate:        incomingDate,
			DocumentTypeID:      models.DocumentTypeLetter,
			DocumentTypeName:    "Письмо",
			Content:             "Входящее",
			CreatedBy:           uuid.New(),
			CreatedByName:       "Регистратор",
			PagesCount:          12,
			Correspondents:      []models.DocumentCorrespondentRegistration{{CorrespondentOrgID: correspondentID, CorrespondentName: "Организация"}},
			SenderSignatory:     "Подписант",
			Resolution:          &resolution,
			ResolutionAuthor:    &resolutionAuthor,
			ResolutionExecutors: &resolutionExecutors,
		})

		require.NotNil(t, item)
		assert.Equal(t, 12, item.PagesCount)
		assert.Equal(t, "IN-12", item.RegistrationNumber)
		require.NotNil(t, item.IncomingDate)
		assert.Equal(t, incomingDate, *item.IncomingDate)
		require.Len(t, item.Correspondents, 1)
		assert.Equal(t, correspondentID.String(), item.Correspondents[0].CorrespondentOrgID)
		assert.Equal(t, "Подписант", item.SenderSignatory)
		require.NotNil(t, item.Resolution)
		assert.Equal(t, "Исполнить", *item.Resolution)
		require.NotNil(t, item.ResolutionAuthor)
		assert.Equal(t, "Руководитель", *item.ResolutionAuthor)
		require.NotNil(t, item.ResolutionExecutors)
		assert.Equal(t, "Исполнитель", *item.ResolutionExecutors)
	})

	t.Run("outgoing", func(t *testing.T) {
		assert.Nil(t, MapOutgoingDocumentListItem(nil))

		outgoingDate := time.Now()
		item := MapOutgoingDocumentListItem(&models.OutgoingDocument{
			ID:               uuid.New(),
			NomenclatureID:   uuid.New(),
			OutgoingNumber:   "OUT-7",
			OutgoingDate:     outgoingDate,
			DocumentTypeID:   models.DocumentTypeLetter,
			Content:          "Исходящее",
			CreatedBy:        uuid.New(),
			PagesCount:       7,
			RecipientOrgName: "Получатель",
			Addressee:        "Адресат",
			SenderSignatory:  "Подписант",
			SenderExecutor:   "Исполнитель",
		})

		require.NotNil(t, item)
		assert.Equal(t, 7, item.PagesCount)
		assert.Equal(t, "OUT-7", item.RegistrationNumber)
		require.NotNil(t, item.OutgoingDate)
		assert.Equal(t, outgoingDate, *item.OutgoingDate)
		assert.Equal(t, "Получатель", item.RecipientOrgName)
		assert.Equal(t, "Адресат", item.Addressee)
		assert.Equal(t, "Подписант", item.SenderSignatory)
		assert.Equal(t, "Исполнитель", item.SenderExecutor)
	})
}

func TestMapDocumentKindSpec(t *testing.T) {
	spec, ok := models.GetDocumentKindSpec(models.DocumentKindIncomingLetter)
	require.True(t, ok)

	dto := MapDocumentKindSpec(spec)

	require.NotNil(t, dto)
	assert.Equal(t, string(models.DocumentKindIncomingLetter), dto.Code)
	assert.Equal(t, spec.Name, dto.Name)
	assert.Equal(t, spec.RegistrationFormCode, dto.RegistrationFormCode)
	assert.Equal(t, spec.RegistryGroup, dto.RegistryGroup)
	assert.Contains(t, dto.SupportedActions, string(models.DocumentActionRead))
}

func TestMapIncomingAndOutgoingDocumentCards(t *testing.T) {
	now := time.Now()

	t.Run("incoming", func(t *testing.T) {
		model := &models.IncomingDocument{
			ID:               uuid.New(),
			NomenclatureID:   uuid.New(),
			NomenclatureName: "01-01",
			IncomingNumber:   "IN-1",
			IncomingDate:     now,
			DocumentTypeID:   models.DocumentTypeLetter,
			DocumentTypeName: models.DocumentTypeLetter,
			Content:          "Входящее",
			CreatedBy:        uuid.New(),
			CreatedByName:    "Регистратор",
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		card := MapIncomingDocumentCard(model)

		require.NotNil(t, card)
		assert.Equal(t, model.ID.String(), card.ID)
		assert.Equal(t, string(models.DocumentKindIncomingLetter), card.KindCode)
		assert.Equal(t, model.IncomingNumber, card.RegistrationNumber)
		assert.NotNil(t, card.IncomingLetter)
		assert.Nil(t, MapIncomingDocumentCard(nil))
	})

	t.Run("outgoing", func(t *testing.T) {
		model := &models.OutgoingDocument{
			ID:               uuid.New(),
			NomenclatureID:   uuid.New(),
			NomenclatureName: "02-01",
			OutgoingNumber:   "OUT-1",
			OutgoingDate:     now,
			DocumentTypeID:   models.DocumentTypeLetter,
			DocumentTypeName: models.DocumentTypeLetter,
			Content:          "Исходящее",
			CreatedBy:        uuid.New(),
			CreatedByName:    "Регистратор",
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		card := MapOutgoingDocumentCard(model)

		require.NotNil(t, card)
		assert.Equal(t, model.ID.String(), card.ID)
		assert.Equal(t, string(models.DocumentKindOutgoingLetter), card.KindCode)
		assert.Equal(t, model.OutgoingNumber, card.RegistrationNumber)
		assert.NotNil(t, card.OutgoingLetter)
		assert.Nil(t, MapOutgoingDocumentCard(nil))
	})
}

func TestMapIncomingAndOutgoingDocumentListSlices(t *testing.T) {
	t.Run("incoming", func(t *testing.T) {
		assert.Nil(t, MapDocumentListItemsFromIncoming(nil))

		id := uuid.New()
		items := MapDocumentListItemsFromIncoming([]models.IncomingDocument{{
			ID:             id,
			NomenclatureID: uuid.New(),
			IncomingNumber: "IN-1",
			IncomingDate:   time.Now(),
			CreatedBy:      uuid.New(),
		}})

		require.Len(t, items, 1)
		assert.Equal(t, id.String(), items[0].ID)
		assert.Equal(t, string(models.DocumentKindIncomingLetter), items[0].KindCode)
	})

	t.Run("outgoing", func(t *testing.T) {
		assert.Nil(t, MapDocumentListItemsFromOutgoing(nil))

		id := uuid.New()
		items := MapDocumentListItemsFromOutgoing([]models.OutgoingDocument{{
			ID:             id,
			NomenclatureID: uuid.New(),
			OutgoingNumber: "OUT-1",
			OutgoingDate:   time.Now(),
			CreatedBy:      uuid.New(),
		}})

		require.Len(t, items, 1)
		assert.Equal(t, id.String(), items[0].ID)
		assert.Equal(t, string(models.DocumentKindOutgoingLetter), items[0].KindCode)
	})
}

func TestMapJournalAndAdminAuditLogs(t *testing.T) {
	now := time.Now()

	t.Run("journal", func(t *testing.T) {
		entry := models.JournalEntry{
			ID:         uuid.New(),
			DocumentID: uuid.New(),
			UserName:   "Регистратор",
			Action:     "CREATE",
			Details:    "Создан документ",
			CreatedAt:  now,
		}

		mapped := MapJournalEntry(&entry)

		require.NotNil(t, mapped)
		assert.Equal(t, entry.ID.String(), mapped.ID)
		assert.Equal(t, entry.DocumentID.String(), mapped.DocumentID)
		assert.Equal(t, entry.UserName, mapped.UserName)
		assert.Nil(t, MapJournalEntry(nil))
		assert.Empty(t, MapJournalEntries(nil))

		items := MapJournalEntries([]models.JournalEntry{entry})
		require.Len(t, items, 1)
		assert.Equal(t, entry.Action, items[0].Action)
	})

	t.Run("admin audit", func(t *testing.T) {
		entry := models.AdminAuditLog{
			ID:        uuid.New(),
			UserName:  "Админ",
			Action:    "UPDATE_USER",
			Details:   "Изменен пользователь",
			CreatedAt: now,
		}

		mapped := MapAdminAuditLog(&entry)

		require.NotNil(t, mapped)
		assert.Equal(t, entry.ID.String(), mapped.ID)
		assert.Equal(t, entry.UserName, mapped.UserName)
		assert.Equal(t, entry.Action, mapped.Action)
		assert.Nil(t, MapAdminAuditLog(nil))
		assert.Empty(t, MapAdminAuditLogs(nil))

		items := MapAdminAuditLogs([]models.AdminAuditLog{entry})
		require.Len(t, items, 1)
		assert.Equal(t, entry.Details, items[0].Details)
	})
}

func TestMapResolutionExecutor(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	model := &models.ResolutionExecutor{
		ID:        id,
		Name:      "Исполнитель резолюции",
		CreatedAt: now,
	}

	mapped := MapResolutionExecutor(model)

	require.NotNil(t, mapped)
	assert.Equal(t, id.String(), mapped.ID)
	assert.Equal(t, model.Name, mapped.Name)
	assert.Equal(t, now, mapped.CreatedAt)
	assert.Nil(t, MapResolutionExecutor(nil))
	assert.Nil(t, MapResolutionExecutors(nil))

	items := MapResolutionExecutors([]models.ResolutionExecutor{*model})
	require.Len(t, items, 1)
	assert.Equal(t, model.Name, items[0].Name)
}

func TestMapSlices(t *testing.T) {
	// Тестирование групповых функций маппинга массивов (слайсов)
	t.Run("MapUsers nil", func(t *testing.T) {
		assert.Nil(t, MapUsers(nil))
	})
	t.Run("MapUsers success", func(t *testing.T) {
		res := MapUsers([]models.User{{ID: uuid.New(), Login: "u"}})
		require.Len(t, res, 1)
		assert.Equal(t, "u", res[0].Login)
	})

	t.Run("MapDepartments nil", func(t *testing.T) {
		assert.Nil(t, MapDepartments(nil))
	})
	t.Run("MapDepartments success", func(t *testing.T) {
		res := MapDepartments([]models.Department{{ID: uuid.New(), Name: "D1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "D1", res[0].Name)
	})

	t.Run("MapNomenclatures nil", func(t *testing.T) {
		assert.Nil(t, MapNomenclatures(nil))
	})
	t.Run("MapNomenclatures success", func(t *testing.T) {
		res := MapNomenclatures([]models.Nomenclature{{ID: uuid.New(), Name: "N1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "N1", res[0].Name)
	})

	t.Run("MapOrganizations nil", func(t *testing.T) {
		assert.Nil(t, MapOrganizations(nil))
	})
	t.Run("MapOrganizations success", func(t *testing.T) {
		res := MapOrganizations([]models.Organization{{ID: uuid.New(), Name: "O1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "O1", res[0].Name)
	})

	t.Run("MapDocumentTypes nil", func(t *testing.T) {
		assert.Nil(t, MapDocumentTypes(nil))
	})
	t.Run("MapDocumentTypes success", func(t *testing.T) {
		res := MapDocumentTypes([]models.DocumentType{{ID: uuid.New(), Name: "T1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "T1", res[0].Name)
	})

	t.Run("MapIncomingDocuments nil", func(t *testing.T) {
		assert.Nil(t, MapIncomingDocuments(nil))
	})
	t.Run("MapIncomingDocuments success", func(t *testing.T) {
		res := MapIncomingDocuments([]models.IncomingDocument{{ID: uuid.New(), IncomingNumber: "IN1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "IN1", res[0].IncomingNumber)
	})

	t.Run("MapOutgoingDocuments nil", func(t *testing.T) {
		assert.Nil(t, MapOutgoingDocuments(nil))
	})
	t.Run("MapOutgoingDocuments success", func(t *testing.T) {
		res := MapOutgoingDocuments([]models.OutgoingDocument{{ID: uuid.New(), OutgoingNumber: "OUT1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "OUT1", res[0].OutgoingNumber)
	})

	t.Run("MapDocumentLinks nil", func(t *testing.T) {
		assert.Nil(t, MapDocumentLinks(nil))
	})
	t.Run("MapDocumentLinks success", func(t *testing.T) {
		res := MapDocumentLinks([]models.DocumentLink{{ID: uuid.New(), LinkType: "reply"}})
		require.Len(t, res, 1)
		assert.Equal(t, "reply", res[0].LinkType)
	})

	t.Run("MapAttachments nil", func(t *testing.T) {
		assert.Nil(t, MapAttachments(nil))
	})
	t.Run("MapAttachments success", func(t *testing.T) {
		res := MapAttachments([]models.Attachment{{ID: uuid.New(), Filename: "F1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "F1", res[0].Filename)
	})

	t.Run("MapAssignments nil", func(t *testing.T) {
		assert.Nil(t, MapAssignments(nil))
	})
	t.Run("MapAssignments success", func(t *testing.T) {
		items := []models.Assignment{{ID: uuid.New(), Status: "new"}, {ID: uuid.New(), Status: "done"}}
		result := MapAssignments(items)
		assert.Len(t, result, 2)
	})

	t.Run("MapAcknowledgments nil", func(t *testing.T) {
		assert.Nil(t, MapAcknowledgments(nil))
	})
	t.Run("MapAcknowledgments success", func(t *testing.T) {
		res := MapAcknowledgments([]models.Acknowledgment{{ID: uuid.New(), Content: "A1"}})
		require.Len(t, res, 1)
		assert.Equal(t, "A1", res[0].Content)
	})
}

func TestMapCitizenAppealDocument(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapCitizenAppealDocument(nil))
	})

	t.Run("success with correspondents and resolutions", func(t *testing.T) {
		id := uuid.New()
		nomID := uuid.New()
		correspondentID := uuid.New()
		createdBy := uuid.New()
		regDate := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
		appealDate := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
		resolutionText := "Подготовить ответ"
		resolutionAuthor := "Руководитель"
		resolutionExecutors := "Исполнитель"

		dto := MapCitizenAppealDocument(&models.CitizenAppealDocument{
			ID:                   id,
			NomenclatureID:       nomID,
			NomenclatureName:     "Appeals",
			RegistrationNumber:   "CA-1",
			RegistrationDate:     regDate,
			AppealDate:           appealDate,
			DocumentTypeID:       models.DocumentTypeCitizenAppeal,
			DocumentTypeName:     models.DocumentTypeCitizenAppeal,
			Content:              "Content",
			PagesCount:           5,
			ApplicantFullName:    "Иван Иванов",
			RegistrationAddress:  "Address",
			AppealType:           "жалоба",
			ApplicantCategory:    "гражданин",
			AppealPagesCount:     2,
			AttachmentPagesCount: 3,
			HasEnvelope:          true,
			ReceivedFromPOS:      true,
			Correspondents: []models.DocumentCorrespondentRegistration{
				{
					ID:                 uuid.New(),
					RegistrationNumber: "EXT-1",
					RegistrationDate:   appealDate,
					CorrespondentOrgID: correspondentID,
					CorrespondentName:  "Администрация",
					Position:           1,
				},
			},
			Resolutions: []models.DocumentResolution{
				{
					ID:                  uuid.New(),
					Resolution:          &resolutionText,
					ResolutionAuthor:    &resolutionAuthor,
					ResolutionExecutors: &resolutionExecutors,
					Position:            1,
				},
			},
			CreatedBy:        createdBy,
			CreatedByName:    "Registrar",
			CreatedAt:        regDate,
			UpdatedAt:        appealDate,
			AttachmentsCount: 1,
			AssignmentsCount: 2,
		})

		require.NotNil(t, dto)
		assert.Equal(t, id.String(), dto.ID)
		assert.Equal(t, nomID.String(), dto.NomenclatureID)
		assert.Equal(t, "CA-1", dto.RegistrationNumber)
		assert.Equal(t, regDate, dto.RegistrationDate)
		assert.Equal(t, appealDate, dto.AppealDate)
		assert.Equal(t, "Иван Иванов", dto.ApplicantFullName)
		assert.Equal(t, "жалоба", dto.AppealType)
		assert.True(t, dto.HasEnvelope)
		assert.True(t, dto.ReceivedFromPOS)
		assert.Len(t, dto.Correspondents, 1)
		assert.Equal(t, correspondentID.String(), dto.Correspondents[0].CorrespondentOrgID)
		assert.Len(t, dto.Resolutions, 1)
		assert.Equal(t, resolutionText, *dto.Resolutions[0].Resolution)
		assert.Equal(t, createdBy.String(), dto.CreatedBy)
		assert.Equal(t, 1, dto.AttachmentsCount)
		assert.Equal(t, 2, dto.AssignmentsCount)
	})
}

func TestMapCitizenAppealDocumentCardAndListItem(t *testing.T) {
	resolutionText := "Резолюция"
	resolutionAuthor := "Автор"
	resolutionExecutors := "Исполнитель"
	model := &models.CitizenAppealDocument{
		ID:                 uuid.New(),
		NomenclatureID:     uuid.New(),
		NomenclatureName:   "Appeals",
		RegistrationNumber: "CA-2",
		RegistrationDate:   time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
		AppealDate:         time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
		DocumentTypeID:     models.DocumentTypeCitizenAppeal,
		DocumentTypeName:   models.DocumentTypeCitizenAppeal,
		Content:            "Appeal content",
		PagesCount:         4,
		ApplicantFullName:  "Петр Петров",
		AppealType:         "заявление",
		Resolutions: []models.DocumentResolution{
			{
				Resolution:          &resolutionText,
				ResolutionAuthor:    &resolutionAuthor,
				ResolutionExecutors: &resolutionExecutors,
			},
		},
		CreatedBy: uuid.New(),
	}

	card := MapCitizenAppealDocumentCard(model)
	require.NotNil(t, card)
	assert.Equal(t, string(models.DocumentKindCitizenAppeal), card.KindCode)
	assert.Equal(t, "CA-2", card.RegistrationNumber)
	require.NotNil(t, card.CitizenAppeal)
	assert.Equal(t, model.ID.String(), card.CitizenAppeal.ID)

	item := MapCitizenAppealDocumentListItem(model)
	require.NotNil(t, item)
	assert.Equal(t, string(models.DocumentKindCitizenAppeal), item.KindCode)
	assert.Equal(t, "CA-2", item.RegistrationNumber)
	require.NotNil(t, item.AppealDate)
	assert.Equal(t, model.AppealDate, *item.AppealDate)
	assert.Equal(t, "Петр Петров", item.ApplicantFullName)
	assert.Equal(t, resolutionText, *item.Resolution)
	assert.Equal(t, resolutionAuthor, *item.ResolutionAuthor)
	assert.Equal(t, resolutionExecutors, *item.ResolutionExecutors)

	assert.Nil(t, MapCitizenAppealDocumentCard(nil))
	assert.Nil(t, MapCitizenAppealDocumentListItem(nil))
}

func TestMapAdministrativeOrderDocument(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAdministrativeOrderDocument(nil))
	})

	t.Run("success with acknowledgment people", func(t *testing.T) {
		id := uuid.New()
		nomID := uuid.New()
		createdBy := uuid.New()
		ackID := uuid.New()
		ackBy := uuid.New()
		orderDate := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
		deadline := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
		ackTime := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)

		dto := MapAdministrativeOrderDocument(&models.AdministrativeOrderDocument{
			ID:                  id,
			NomenclatureID:      nomID,
			NomenclatureName:    "Orders",
			OrderNumber:         "ORD-1",
			OrderDate:           orderDate,
			Title:               "Order title",
			ExecutionController: "Controller",
			ExecutionDeadline:   &deadline,
			IsActive:            true,
			AcknowledgmentPeople: []models.AdministrativeOrderAcknowledgmentPerson{
				{
					ID:                 ackID,
					FullName:           "Иван Иванов",
					AcknowledgedAt:     &ackTime,
					AcknowledgedBy:     &ackBy,
					AcknowledgedByName: "Registrar",
					Position:           1,
					CreatedAt:          orderDate,
				},
			},
			CreatedBy:        createdBy,
			CreatedByName:    "Creator",
			CreatedAt:        orderDate,
			UpdatedAt:        ackTime,
			AttachmentsCount: 1,
			AssignmentsCount: 2,
		})

		require.NotNil(t, dto)
		assert.Equal(t, id.String(), dto.ID)
		assert.Equal(t, nomID.String(), dto.NomenclatureID)
		assert.Equal(t, "ORD-1", dto.OrderNumber)
		assert.Equal(t, models.DocumentTypeAdministrativeOrder, dto.DocumentTypeID)
		assert.Equal(t, "Order title", dto.Content)
		assert.Equal(t, 1, dto.PagesCount)
		assert.Equal(t, "Controller", dto.ExecutionController)
		require.NotNil(t, dto.ExecutionDeadline)
		assert.Equal(t, deadline, *dto.ExecutionDeadline)
		assert.True(t, dto.IsActive)
		assert.Len(t, dto.AcknowledgmentPeople, 1)
		assert.Equal(t, ackID.String(), dto.AcknowledgmentPeople[0].ID)
		assert.Equal(t, ackBy.String(), dto.AcknowledgmentPeople[0].AcknowledgedBy)
		assert.Equal(t, createdBy.String(), dto.CreatedBy)
		assert.Equal(t, 1, dto.AttachmentsCount)
		assert.Equal(t, 2, dto.AssignmentsCount)
	})
}

func TestMapAdministrativeOrderDocumentCardAndListItem(t *testing.T) {
	cancelledAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	model := &models.AdministrativeOrderDocument{
		ID:                  uuid.New(),
		NomenclatureID:      uuid.New(),
		NomenclatureName:    "Orders",
		OrderNumber:         "ORD-2",
		OrderDate:           time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		Title:               "Inactive order",
		ExecutionController: "Controller",
		IsActive:            false,
		CancelledAt:         &cancelledAt,
		AcknowledgmentPeople: []models.AdministrativeOrderAcknowledgmentPerson{
			{ID: uuid.New(), FullName: "Acknowledged", AcknowledgedAt: &cancelledAt},
			{ID: uuid.New(), FullName: "Pending"},
		},
		CreatedBy: uuid.New(),
	}

	card := MapAdministrativeOrderDocumentCard(model)
	require.NotNil(t, card)
	assert.Equal(t, string(models.DocumentKindAdministrativeOrder), card.KindCode)
	assert.Equal(t, "ORD-2", card.RegistrationNumber)
	assert.Equal(t, models.DocumentTypeAdministrativeOrder, card.DocumentTypeID)
	require.NotNil(t, card.AdministrativeOrder)
	assert.Equal(t, model.ID.String(), card.AdministrativeOrder.ID)

	item := MapAdministrativeOrderDocumentListItem(model)
	require.NotNil(t, item)
	assert.Equal(t, string(models.DocumentKindAdministrativeOrder), item.KindCode)
	assert.Equal(t, "ORD-2", item.OrderNumber)
	require.NotNil(t, item.OrderDate)
	assert.Equal(t, model.OrderDate, *item.OrderDate)
	assert.False(t, item.IsActive)
	require.NotNil(t, item.CancelledAt)
	assert.Equal(t, cancelledAt, *item.CancelledAt)
	assert.Equal(t, 1, item.PendingAcknowledgmentsCount)
	assert.Len(t, item.AcknowledgmentPeople, 2)

	assert.Nil(t, MapAdministrativeOrderDocumentCard(nil))
	assert.Nil(t, MapAdministrativeOrderDocumentListItem(nil))
}

func TestMapCitizenAppealAndAdministrativeOrderSlices(t *testing.T) {
	t.Run("citizen appeals nil", func(t *testing.T) {
		assert.Nil(t, MapCitizenAppealDocuments(nil))
		assert.Nil(t, MapDocumentListItemsFromCitizenAppeals(nil))
	})

	t.Run("citizen appeals success", func(t *testing.T) {
		items := []models.CitizenAppealDocument{{ID: uuid.New(), RegistrationNumber: "CA-1"}}
		docs := MapCitizenAppealDocuments(items)
		require.Len(t, docs, 1)
		assert.Equal(t, "CA-1", docs[0].RegistrationNumber)

		listItems := MapDocumentListItemsFromCitizenAppeals(items)
		require.Len(t, listItems, 1)
		assert.Equal(t, "CA-1", listItems[0].RegistrationNumber)
	})

	t.Run("administrative orders nil", func(t *testing.T) {
		assert.Nil(t, MapDocumentListItemsFromAdministrativeOrders(nil))
	})

	t.Run("administrative orders success", func(t *testing.T) {
		items := []models.AdministrativeOrderDocument{{ID: uuid.New(), OrderNumber: "ORD-1"}}
		listItems := MapDocumentListItemsFromAdministrativeOrders(items)
		require.Len(t, listItems, 1)
		assert.Equal(t, "ORD-1", listItems[0].OrderNumber)
	})
}
