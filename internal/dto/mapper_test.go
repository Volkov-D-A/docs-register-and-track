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

func TestMapDashboardStats(t *testing.T) {
	// Тестирование маппинга агрегированной статистики для дашборда
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapDashboardStats(nil))
	})

	t.Run("success", func(t *testing.T) {
		m := &models.DashboardStats{Role: "admin", UserCount: 5, TotalDocuments: 100, DBSize: "10MB"}
		d := MapDashboardStats(m)
		assert.Equal(t, "admin", d.Role)
		assert.Equal(t, 5, d.UserCount)
		assert.Equal(t, 100, d.TotalDocuments)
	})
}
