package repository

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestUserEventOutboxInsertIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	userID, documentID := seedIntegrationDocument(t, sqlDB)
	repo := NewUserEventRepository(&database.DB{DB: sqlDB})
	request := models.CreateUserEventRequest{
		RecipientUserID: userID,
		DocumentID:      documentID,
		DocumentKind:    string(models.DocumentKindIncomingLetter),
		EntityType:      models.UserEventEntityAssignment,
		EventType:       models.UserEventAssignmentCreated,
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

	// Rollback must restore the old shape for existing rows; reapplying the
	// migration must preserve the event while removing the unused columns.
	if err := (&database.DB{DB: sqlDB}).RollbackMigration(database.DefaultMigrationsPath); err != nil {
		t.Fatal(err)
	}
	var entityID string
	if err := sqlDB.QueryRow(`SELECT entity_id FROM user_events WHERE outbox_deduplication_key = $1`, key).Scan(&entityID); err != nil {
		t.Fatal(err)
	}
	if entityID != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("rollback placeholder entity_id = %s", entityID)
	}
	if err := (&database.DB{DB: sqlDB}).RunMigrations(database.DefaultMigrationsPath); err != nil {
		t.Fatal(err)
	}
	var unusedColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'user_events' AND column_name IN ('actor_user_id', 'entity_id', 'metadata')`).Scan(&unusedColumns); err != nil {
		t.Fatal(err)
	}
	if unusedColumns != 0 {
		t.Fatalf("migration left %d unused user_events columns", unusedColumns)
	}
}
