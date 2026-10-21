package repository

import (
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestUserEventOutboxInsertIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	userID, documentID := seedIntegrationDocument(t, sqlDB)
	execSQL(t, sqlDB, `UPDATE documents SET registration_date = '2026-09-24' WHERE id = $1`, documentID)
	execSQL(t, sqlDB, `INSERT INTO document_permissions(kind_code,subject_type,subject_key,action) VALUES('outgoing_letter','user',$1,'register')`, userID.String())
	repo := NewUserEventRepository(database.Wrap(sqlDB))
	request := models.CreateUserEventRequest{
		RecipientUserID: userID,
		DocumentID:      documentID,
		DocumentKind:    string(models.DocumentKindIncomingLetter),
		EntityType:      "deadline_control",
		EventType:       "deadline_control_create",
		Title:           "Новое поручение",
		Message:         "Назначено поручение",
	}
	const key = "user-event:outbox-integration"
	for range 2 {
		if err := repo.CreateFromOutbox(request, key); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM user_events WHERE outbox_deduplication_key = $1`, key).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("outbox delivery created %d events, want 1", count)
	}
	listed, err := repo.GetList(userID, models.UserEventFilter{Page: 1, PageSize: 20})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].DocumentID != documentID || listed.Items[0].Title != request.Title {
		t.Fatalf("visible events=%+v err=%v", listed, err)
	}

	if listed.Items[0].DocumentDate == nil || listed.Items[0].DocumentDate.Format("2006-01-02") != "2026-09-24" {
		t.Fatalf("document date=%v, want 2026-09-24", listed.Items[0].DocumentDate)
	}
	if err := repo.MarkAllRead(userID, time.Now()); err != nil {
		t.Fatal(err)
	}
	listed, err = repo.GetList(userID, models.UserEventFilter{Page: 1, PageSize: 20})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ReadAt == nil {
		t.Fatalf("read event disappeared from the list: events=%+v err=%v", listed, err)
	}
	unread, err := repo.CountUnread(userID)
	if err != nil || unread != 0 {
		t.Fatalf("unread count=%d err=%v, want 0", unread, err)
	}

	var unusedColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'user_events' AND column_name IN ('actor_user_id', 'entity_id', 'metadata')`).Scan(&unusedColumns); err != nil {
		t.Fatal(err)
	}
	if unusedColumns != 0 {
		t.Fatalf("fresh schema contains %d unused user_events columns", unusedColumns)
	}
}
