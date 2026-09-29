package repository

import (
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	servereffects "github.com/Volkov-D-A/docs-register-and-track/internal/server/effects"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
)

func TestAssignmentSeriesLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	actorID, documentID := seedIntegrationDocument(t, sqlDB)
	if _, err := db.Exec(`UPDATE users SET is_document_participant=TRUE WHERE id=$1`, actorID); err != nil {
		t.Fatal(err)
	}
	repo := NewAssignmentRepository(db)
	repo.SetOutbox(NewOutboxRepository(db))
	seriesID, firstID := uuid.New(), uuid.New()
	firstDeadline := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	series, err := repo.CreateSeriesWithFirstAssignment(seriesID, firstID, documentID, actorID, actorID, "Квартальный отчёт", firstDeadline, "month", 3, "last_day", 0, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if series.CurrentAssignmentID == nil || *series.CurrentAssignmentID != firstID || series.CurrentIteration != 1 {
		t.Fatalf("unexpected initial series: %+v", series)
	}

	visible, err := repo.GetList(models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, DocumentID: documentID.String(), Page: 1, PageSize: 20, ShowFinished: true})
	if err != nil || len(visible.Items) != 1 || visible.Items[0].ID != firstID {
		t.Fatalf("initial visible assignments=%+v err=%v", visible, err)
	}

	completedAt := time.Now().UTC()
	if _, err = sqlDB.Exec(`UPDATE assignments SET status='completed',report='готово',completed_at=$1 WHERE id=$2`, completedAt, firstID); err != nil {
		t.Fatal(err)
	}
	secondID := uuid.New()
	secondDeadline := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	if _, err = sqlDB.Exec(`UPDATE assignment_series SET updated_at=updated_at+INTERVAL '1 second' WHERE id=$1`, seriesID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.FinishSeriesIterationWithNext(firstID, seriesID, secondID, actorID, series.UpdatedAt, "готово", &completedAt, secondDeadline, 2, actorID, "Квартальный отчёт", nil, nil, nil, models.OutboxEvent{}); err == nil {
		t.Fatal("stale series revision was accepted")
	}
	series, err = repo.GetAssignmentSeries(seriesID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.FinishSeriesIterationWithNext(firstID, seriesID, secondID, actorID, series.UpdatedAt, "готово", &completedAt, secondDeadline, 2, actorID, "Квартальный отчёт", nil, nil, nil, models.OutboxEvent{}); err != nil {
		t.Fatalf("advance series: %v", err)
	}

	visible, err = repo.GetList(models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, DocumentID: documentID.String(), Page: 1, PageSize: 20, ShowFinished: true})
	if err != nil || len(visible.Items) != 1 || visible.Items[0].ID != secondID || visible.Items[0].IterationNumber != 2 {
		t.Fatalf("advanced visible assignments=%+v err=%v", visible, err)
	}
	history, err := repo.GetAssignmentSeriesHistory(seriesID)
	if err != nil || len(history) != 2 || history[0].ID != secondID || history[1].Status != "finished" {
		t.Fatalf("series history=%+v err=%v", history, err)
	}
	if _, err = sqlDB.Exec(`INSERT INTO attachments(document_id,assignment_id,filename,storage_path,file_size,content_type,uploaded_by) VALUES($1,$2,'result.pdf','series/result.pdf',10,'application/pdf',$3)`, documentID, firstID, actorID); err != nil {
		t.Fatalf("insert linked attachment: %v", err)
	}
	attachmentRepo := NewAttachmentRepository(db)
	files, err := attachmentRepo.GetByAssignmentID(firstID)
	if err != nil || len(files) != 1 || files[0].Filename != "result.pdf" || files[0].FileSize != 10 {
		t.Fatalf("iteration files=%+v err=%v", files, err)
	}
	files, err = attachmentRepo.GetByDocumentID(documentID)
	if err != nil || len(files) != 1 || files[0].Filename != "result.pdf" || files[0].FileSize != 10 {
		t.Fatalf("document files=%+v err=%v", files, err)
	}

	advancedSeries, getErr := repo.GetAssignmentSeries(seriesID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if _, err = repo.FinishSeriesIterationWithNext(firstID, seriesID, uuid.New(), actorID, advancedSeries.UpdatedAt, "повтор", &completedAt, secondDeadline, 2, actorID, "Квартальный отчёт", nil, nil, nil, models.OutboxEvent{}); err == nil {
		t.Fatal("concurrent retry advanced a non-current iteration")
	}
}

func TestAssignmentSeriesFinishesWithoutNextWhenRecipientsBecomeIneligibleIntegration(t *testing.T) {
	cases := []struct {
		name          string
		mainActive    bool
		coParticipant bool
	}{
		{"inactive main", false, true},
		{"nonparticipant coexecutor", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := integrationdb.Open(t)
			db := database.Wrap(sqlDB)
			actorID, documentID := seedIntegrationDocument(t, sqlDB)
			if _, err := db.Exec(`INSERT INTO user_system_permissions(user_id, permission, is_allowed) VALUES($1, 'admin', TRUE)`, actorID); err != nil {
				t.Fatal(err)
			}
			repo := NewAssignmentRepository(db)
			repo.SetOutbox(NewOutboxRepository(db))
			executorID, coExecutorID := uuid.New(), uuid.New()
			_, err := db.Exec(`INSERT INTO users(id, login, password_hash, last_name, first_name, no_patronymic, is_active, is_document_participant) VALUES ($1, $2, 'hash', 'Main', 'Executor', TRUE, TRUE, TRUE),($3, $4, 'hash', 'Coexecutor', 'User', TRUE, TRUE, TRUE)`, executorID, "series-main-"+executorID.String(), coExecutorID, "series-co-"+coExecutorID.String())
			if err != nil {
				t.Fatal(err)
			}
			seriesID, firstID, nextID := uuid.New(), uuid.New(), uuid.New()
			deadline := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
			series, err := repo.CreateSeriesWithFirstAssignment(seriesID, firstID, documentID, executorID, actorID, "Отчёт", deadline, "month", 3, "last_day", 0, []string{coExecutorID.String()}, nil)
			if err != nil {
				t.Fatal(err)
			}
			completedAt := time.Now().UTC()
			_, err = db.Exec(`UPDATE assignments SET status='completed',report='готово',completed_at=$1 WHERE id=$2`, completedAt, firstID)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.mainActive {
				_, err = db.Exec(`UPDATE users SET is_active=FALSE WHERE id=$1`, executorID)
			} else if !tc.coParticipant {
				_, err = db.Exec(`UPDATE users SET is_document_participant=FALSE WHERE id=$1`, coExecutorID)
			}
			if err != nil {
				t.Fatal(err)
			}
			cancelKey := "series-auto-cancel-" + seriesID.String()
			cancelEffect, err := servereffects.NewJournalOutboxEvent(cancelKey, models.CreateJournalEntryRequest{DocumentID: documentID, UserID: actorID, Action: "ASSIGNMENT_SERIES_CANCEL", Details: "Исполнитель недоступен"})
			if err != nil {
				t.Fatal(err)
			}
			nextKey := "series-next-" + seriesID.String()
			nextEffect, err := servereffects.NewJournalOutboxEvent(nextKey, models.CreateJournalEntryRequest{DocumentID: documentID, UserID: actorID, Action: "ASSIGNMENT_SERIES_ITERATION_CREATE", Details: "Следующая итерация"})
			if err != nil {
				t.Fatal(err)
			}
			result, err := repo.FinishSeriesIterationWithNext(firstID, seriesID, nextID, actorID, series.UpdatedAt, "готово", &completedAt, deadline.AddDate(0, 3, 0), 2, executorID, "Отчёт", []string{coExecutorID.String()}, nil, []models.OutboxEvent{nextEffect}, cancelEffect)
			if err != nil {
				t.Fatal(err)
			}
			if result == nil || result.Status != "finished" {
				t.Fatalf("current assignment was not accepted: %+v", result)
			}
			var active bool
			var cancelledBy uuid.UUID
			var cancelledAt time.Time
			var iteration int
			err = db.QueryRow(`SELECT active,cancelled_by,cancelled_at,current_iteration FROM assignment_series WHERE id=$1`, seriesID).Scan(&active, &cancelledBy, &cancelledAt, &iteration)
			if err != nil || active || cancelledBy != actorID || cancelledAt.IsZero() || iteration != 1 {
				t.Fatalf("unexpected series state: active=%v actor=%v cancelled=%v iteration=%d err=%v", active, cancelledBy, cancelledAt, iteration, err)
			}
			var nextCount, cancelEvents, nextEvents int
			if err = db.QueryRow(`SELECT COUNT(*) FROM assignments WHERE id=$1`, nextID).Scan(&nextCount); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE deduplication_key=$1`, cancelKey).Scan(&cancelEvents); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(`SELECT COUNT(*) FROM event_outbox WHERE deduplication_key=$1`, nextKey).Scan(&nextEvents); err != nil {
				t.Fatal(err)
			}
			if nextCount != 0 || cancelEvents != 1 || nextEvents != 0 {
				t.Fatalf("unexpected next assignment or effects: next=%d cancel=%d nextEvent=%d", nextCount, cancelEvents, nextEvents)
			}
		})
	}
}
