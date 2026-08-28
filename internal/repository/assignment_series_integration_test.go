package repository

import (
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
	"github.com/google/uuid"
)

func TestAssignmentSeriesLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	actorID, documentID := seedIntegrationDocument(t, sqlDB)
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

	visible, err := repo.GetList(models.AssignmentFilter{DocumentID: documentID.String(), Page: 1, PageSize: 20, ShowFinished: true})
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
	if _, err = repo.FinishSeriesIterationWithNext(firstID, seriesID, secondID, series.UpdatedAt, "готово", &completedAt, secondDeadline, 2, actorID, "Квартальный отчёт", nil, nil, nil); err == nil {
		t.Fatal("stale series revision was accepted")
	}
	series, err = repo.GetAssignmentSeries(seriesID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.FinishSeriesIterationWithNext(firstID, seriesID, secondID, series.UpdatedAt, "готово", &completedAt, secondDeadline, 2, actorID, "Квартальный отчёт", nil, nil, nil); err != nil {
		t.Fatalf("advance series: %v", err)
	}

	visible, err = repo.GetList(models.AssignmentFilter{DocumentID: documentID.String(), Page: 1, PageSize: 20, ShowFinished: true})
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
	files, err := NewAttachmentRepository(db).GetByAssignmentID(firstID)
	if err != nil || len(files) != 1 || files[0].AssignmentID == nil || *files[0].AssignmentID != firstID {
		t.Fatalf("iteration files=%+v err=%v", files, err)
	}

	advancedSeries, getErr := repo.GetAssignmentSeries(seriesID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if _, err = repo.FinishSeriesIterationWithNext(firstID, seriesID, uuid.New(), advancedSeries.UpdatedAt, "повтор", &completedAt, secondDeadline, 2, actorID, "Квартальный отчёт", nil, nil, nil); err == nil {
		t.Fatal("concurrent retry advanced a non-current iteration")
	}
}
