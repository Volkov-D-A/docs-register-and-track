package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	serverservices "github.com/Volkov-D-A/docs-register-and-track/internal/server/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type deadlineFixture struct {
	db       *sql.DB
	repo     *repository.DeadlineControlRepository
	users    [5]uuid.UUID
	docs     [4]uuid.UUID
	external string
}

func newDeadlineFixture(t *testing.T) *deadlineFixture {
	t.Helper()
	db := integrationdb.Open(t)
	wrapped := database.Wrap(db)
	f := &deadlineFixture{db: db, repo: repository.NewDeadlineControlRepository(wrapped, repository.NewOutboxRepository(wrapped))}
	for i := range f.users {
		f.users[i] = uuid.New()
		_, err := db.Exec(`INSERT INTO users(id,login,password_hash,last_name,first_name,no_patronymic,is_document_participant,password_change_required) VALUES($1,$2,'test','Test','User',true,true,false)`, f.users[i], fmt.Sprintf("deadline-%d", i))
		require.NoError(t, err)
	}
	_, err := db.Exec(`INSERT INTO user_system_permissions(user_id,permission) VALUES($1,'admin')`, f.users[0])
	require.NoError(t, err)
	for _, id := range f.users[:2] {
		_, err = db.Exec(`INSERT INTO document_permissions(kind_code,subject_type,subject_key,action) VALUES('incoming_letter','user',$1,'control_deadlines')`, id.String())
		require.NoError(t, err)
	}
	n := uuid.New()
	_, err = db.Exec(`INSERT INTO nomenclature(id,name,index,year,kind_code) VALUES($1,'Test','deadline',2026,'incoming_letter')`, n)
	require.NoError(t, err)
	for i := range f.docs {
		f.docs[i] = uuid.New()
		kind := "incoming_letter"
		if i > 0 {
			kind = "outgoing_letter"
		}
		_, err = db.Exec(`INSERT INTO documents(id,kind,nomenclature_id,registration_number,registration_date,document_type,content,created_by) VALUES($1,$2,$3,$4,'2026-10-01','Письмо','Test',$5)`, f.docs[i], kind, n, fmt.Sprint(i), f.users[0])
		require.NoError(t, err)
	}
	external, err := f.service(0).SaveExternalExecutor(context.Background(), "", models.ExternalDeadlineExecutorRequest{Name: "Внешний исполнитель", Institution: "Учреждение", IsActive: true})
	require.NoError(t, err)
	f.external = external.ID
	return f
}
func (f *deadlineFixture) service(index int) *serverservices.DeadlineControlService {
	return serverservices.NewDeadlineControlService(f.repo, requestDocumentPrincipal{user: &models.User{ID: f.users[index], IsActive: true, IsDocumentParticipant: true}})
}
func (f *deadlineFixture) request() models.DeadlineControlRequest {
	return models.DeadlineControlRequest{DocumentID: f.docs[0].String(), ExecutorID: f.users[2].String(), CoExecutorIDs: []string{f.external}, Subject: "Проверить исполнение", DueDate: "2026-10-10"}
}

func TestDeadlineControlsAuthorizationAndHistoryIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	creator, controller, responsible := f.service(0), f.service(1), f.service(2)
	c, err := creator.Create(ctx, f.request())
	require.NoError(t, err)
	require.Equal(t, f.users[0].String(), c.CreatedBy)
	visible, err := responsible.Get(ctx, c.ID)
	require.NoError(t, err)
	require.False(t, visible.CanManage)
	_, err = responsible.Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: "2026-10-10"})
	require.ErrorIs(t, err, models.ErrForbidden)
	_, err = f.service(3).Get(ctx, c.ID)
	require.ErrorIs(t, err, models.ErrForbidden)
	update := f.request()
	update.Version = c.Version
	update.DueDate = "2026-10-11"
	update.ChangeReason = "Продление"
	c, err = controller.Update(ctx, c.ID, update)
	require.NoError(t, err)
	require.Equal(t, f.users[0].String(), c.CreatedBy)
	beforeEvents := 0
	require.NoError(t, f.db.QueryRow(`SELECT count(*) FROM event_outbox`).Scan(&beforeEvents))
	c, err = controller.Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: "2026-10-10", Comment: "Исполнено"})
	require.NoError(t, err)
	require.Equal(t, f.users[1].String(), c.CompletedBy)
	require.Equal(t, "2026-10-10", c.ExecutedOn)
	require.False(t, c.WasExecutedLate())
	_, err = responsible.Get(ctx, c.ID)
	require.NoError(t, err)
	eventCount := 0
	require.NoError(t, f.db.QueryRow(`SELECT count(*) FROM event_outbox`).Scan(&eventCount))
	require.Greater(t, eventCount, beforeEvents)
	_, err = controller.Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: "2026-10-10"})
	require.True(t, isDeadlineConflict(err), "%v", err)
	var afterRepeat int
	require.NoError(t, f.db.QueryRow(`SELECT count(*) FROM event_outbox`).Scan(&afterRepeat))
	require.Equal(t, eventCount, afterRepeat)
	var actors int
	require.NoError(t, f.db.QueryRow(`SELECT count(DISTINCT payload->>'UserID') FROM event_outbox WHERE event_type='journal_entry' AND payload->>'DocumentID'=$1`, f.docs[0].String()).Scan(&actors))
	require.Equal(t, 2, actors)
}

func TestDeadlineControlsVisibilityClosureAndRevocationIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	c, err := f.service(0).Create(ctx, f.request())
	require.NoError(t, err)
	for _, edge := range [][2]int{{0, 1}, {1, 2}, {2, 0}} {
		_, err = f.db.Exec(`INSERT INTO document_links(source_document_id,target_document_id,link_type,created_by) VALUES($1,$2,'related',$3)`, f.docs[edge[0]], f.docs[edge[1]], f.users[0])
		require.NoError(t, err)
	}
	for _, i := range []int{0, 1, 2} {
		allowed, err := f.repo.CanRead(ctx, f.users[2], f.docs[i].String())
		require.NoError(t, err)
		require.True(t, allowed)
	}
	allowed, err := f.repo.CanRead(ctx, f.users[2], f.docs[3].String())
	require.NoError(t, err)
	require.False(t, allowed)
	second := f.request()
	second.ExecutorID = f.users[2].String()
	second.Subject = "Второй контроль"
	_, err = f.service(1).Create(ctx, second)
	require.NoError(t, err)
	list, err := f.service(2).List(ctx, models.DeadlineControlFilter{MineOnly: true, DueFrom: "2026-10-01", DueTo: "2026-10-31", PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.Len(t, list.Items, 1)
	empty, err := f.service(4).List(ctx, models.DeadlineControlFilter{})
	require.NoError(t, err)
	require.Zero(t, empty.Total)
	require.Empty(t, empty.Items)
	_, err = f.service(1).Cancel(ctx, c.ID, models.CancelDeadlineRequest{Version: c.Version, Reason: "Отмена"})
	require.NoError(t, err)
	// Another active control still grants access. Remove that assignment explicitly.
	remaining, err := f.service(2).List(ctx, models.DeadlineControlFilter{Status: models.DeadlineActive})
	require.NoError(t, err)
	require.Len(t, remaining.Items, 1)
	update := f.request()
	update.Version = remaining.Items[0].Version
	update.ExecutorID = f.users[3].String()
	update.CoExecutorIDs = []string{}
	_, err = f.service(1).Update(ctx, remaining.Items[0].ID, update)
	require.NoError(t, err)
	allowed, err = f.repo.CanRead(ctx, f.users[2], f.docs[2].String())
	require.NoError(t, err)
	require.False(t, allowed)
	// Department-nomenclature access is independent of the executor eligibility flag.
	dep := uuid.New()
	_, err = f.db.Exec(`INSERT INTO departments(id,name) VALUES($1,'Test department')`, dep)
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE users SET department_id=$1,is_document_participant=false WHERE id=$2`, dep, f.users[4])
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO department_nomenclature(department_id,nomenclature_id) SELECT $1,nomenclature_id FROM documents WHERE id=$2`, dep, f.docs[0])
	require.NoError(t, err)
	allowed, err = f.repo.CanRead(ctx, f.users[4], f.docs[2].String())
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestDeadlineSeriesConcurrencyCalendarAndResumeIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	req := f.request()
	req.DueDate = "2026-01-31"
	series, err := f.service(0).CreateSeries(ctx, models.DeadlineSeriesRequest{Control: req, Recurrence: models.DeadlineRecurrence{IntervalUnit: "month", IntervalValue: 1, DayRule: "fixed", DayOfMonth: 31}})
	require.NoError(t, err)
	original, err := f.service(1).Get(ctx, series.CurrentControlID)
	require.NoError(t, err)
	update := req
	update.Version = original.Version
	update.DueDate = "2026-02-10"
	update.ChangeReason = "Перенос рабочего срока"
	current, err := f.service(1).Update(ctx, original.ID, update)
	require.NoError(t, err)
	errorsChannel := make(chan error, 2)
	var wg sync.WaitGroup
	for _, index := range []int{0, 1} {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := f.service(index).Complete(ctx, current.ID, models.CompleteDeadlineRequest{Version: current.Version, ExecutedOn: "2026-02-05"})
			errorsChannel <- err
		}(index)
	}
	wg.Wait()
	close(errorsChannel)
	successes, conflicts := 0, 0
	for err := range errorsChannel {
		if err == nil {
			successes++
		} else if isDeadlineConflict(err) {
			conflicts++
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	series, err = f.service(1).GetSeries(ctx, series.ID)
	require.NoError(t, err)
	require.Equal(t, 2, series.CurrentIteration)
	next, err := f.service(1).Get(ctx, series.CurrentControlID)
	require.NoError(t, err)
	require.Equal(t, "2026-02-28", next.DueDate)
	require.Equal(t, "2026-02-28", next.PlannedDueDate)
	series, err = f.service(1).StopSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: series.Version, Reason: "Пауза"})
	require.NoError(t, err)
	next, err = f.service(0).Complete(ctx, next.ID, models.CompleteDeadlineRequest{Version: next.Version, ExecutedOn: "2026-02-28"})
	require.NoError(t, err)
	series, err = f.service(1).ResumeSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: series.Version, ExpectedNextDueDate: "2026-03-31"})
	require.NoError(t, err)
	require.Equal(t, 3, series.CurrentIteration)
	next, err = f.service(1).Get(ctx, series.CurrentControlID)
	require.NoError(t, err)
	require.Equal(t, "2026-03-31", next.DueDate)
	_, err = f.service(0).Cancel(ctx, next.ID, models.CancelDeadlineRequest{Version: next.Version, Reason: "Отмена итерации"})
	require.NoError(t, err)
	series, err = f.service(0).GetSeries(ctx, series.ID)
	require.NoError(t, err)
	require.False(t, series.Active)
	_, err = f.service(1).ResumeSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: series.Version, ExpectedNextDueDate: "2026-04-30"})
	require.NoError(t, err)
	var count int
	require.NoError(t, f.db.QueryRow(`SELECT count(*) FROM document_deadline_controls WHERE series_id=$1`, series.ID).Scan(&count))
	require.Equal(t, 4, count)
}
func TestDeadlineSeriesStopsForUnavailableCoExecutorIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	series, err := f.service(0).CreateSeries(ctx, models.DeadlineSeriesRequest{Control: f.request(), Recurrence: models.DeadlineRecurrence{IntervalUnit: "day", IntervalValue: 1, DayRule: "same_day"}})
	require.NoError(t, err)
	executors, err := f.service(0).ListExecutors(ctx, true)
	require.NoError(t, err)
	var external models.DeadlineExecutor
	for _, e := range executors {
		if e.ID == f.external {
			external = e
		}
	}
	_, err = f.service(0).SaveExternalExecutor(ctx, f.external, models.ExternalDeadlineExecutorRequest{Name: external.Name, Institution: external.Institution, Version: external.Version, IsActive: false})
	require.NoError(t, err)
	current, err := f.service(1).Get(ctx, series.CurrentControlID)
	require.NoError(t, err)
	_, err = f.service(1).Complete(ctx, current.ID, models.CompleteDeadlineRequest{Version: current.Version, ExecutedOn: "2026-10-10"})
	require.NoError(t, err)
	series, err = f.service(1).GetSeries(ctx, series.ID)
	require.NoError(t, err)
	require.False(t, series.Active)
	require.NotEmpty(t, series.StopReason)
	require.Equal(t, 1, series.CurrentIteration)
	_, err = f.service(1).ResumeSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: series.Version, ExpectedNextDueDate: "2026-10-11"})
	require.Error(t, err)
}

func isDeadlineConflict(err error) bool {
	var appErr *models.AppError
	return errors.As(err, &appErr) && appErr.Code == 409
}

func TestDeadlineControlsRejectStaleVersionsAndRollbackEffectsIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	_, err := f.db.Exec(`ALTER TABLE event_outbox ADD CONSTRAINT reject_deadline_journal CHECK(event_type<>'journal_entry')`)
	require.NoError(t, err)
	_, err = f.service(0).CreateSeries(ctx, models.DeadlineSeriesRequest{Control: f.request(), Recurrence: models.DeadlineRecurrence{IntervalUnit: "day", IntervalValue: 1, DayRule: "same_day"}})
	require.Error(t, err)
	for _, table := range []string{"document_deadline_controls", "document_deadline_series"} {
		var count int
		require.NoError(t, f.db.QueryRow(`SELECT count(*) FROM `+table).Scan(&count))
		require.Zero(t, count)
	}
	_, err = f.db.Exec(`ALTER TABLE event_outbox DROP CONSTRAINT reject_deadline_journal`)
	require.NoError(t, err)
	c, err := f.service(0).Create(ctx, f.request())
	require.NoError(t, err)
	stale := f.request()
	stale.Version = c.Version
	stale.Subject = "Изменённый предмет"
	updated, err := f.service(1).Update(ctx, c.ID, stale)
	require.NoError(t, err)
	_, err = f.service(0).Update(ctx, c.ID, stale)
	require.True(t, isDeadlineConflict(err), "%v", err)
	_, err = f.service(0).Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: "2026-10-10"})
	require.True(t, isDeadlineConflict(err), "%v", err)
	var due string
	require.NoError(t, f.db.QueryRow(`SELECT subject FROM document_deadline_controls WHERE id=$1`, c.ID).Scan(&due))
	require.Equal(t, updated.Subject, due)
	// Direct SQL cannot remove a referenced co-executor or insert an invalid set.
	_, err = f.db.Exec(`DELETE FROM deadline_executors WHERE id=$1`, f.external)
	require.Error(t, err)
	_, err = f.db.Exec(`UPDATE document_deadline_controls SET co_executor_ids=ARRAY[executor_id] WHERE id=$1`, c.ID)
	require.Error(t, err)
	_, err = f.db.Exec(`UPDATE document_deadline_controls SET co_executor_ids=ARRAY[$1::uuid] WHERE id=$2`, uuid.New(), c.ID)
	require.Error(t, err)
}
func TestDeadlineSeriesAuthorDeactivationAndPermissionRevocationIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	series, err := f.service(0).CreateSeries(ctx, models.DeadlineSeriesRequest{Control: f.request(), Recurrence: models.DeadlineRecurrence{IntervalUnit: "day", IntervalValue: 1, DayRule: "same_day"}})
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO user_system_permissions(user_id,permission) VALUES($1,'admin')`, f.users[1])
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE users SET is_active=false WHERE id=$1`, f.users[0])
	require.NoError(t, err)
	c, err := f.service(1).Get(ctx, series.CurrentControlID)
	require.NoError(t, err)
	_, err = f.service(0).Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: "2026-10-10"})
	require.ErrorIs(t, err, models.ErrForbidden)
	_, err = f.service(1).Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: "2026-10-10"})
	require.NoError(t, err)
	series, err = f.service(1).GetSeries(ctx, series.ID)
	require.NoError(t, err)
	require.True(t, series.Active)
	require.Equal(t, 2, series.CurrentIteration)
	require.Equal(t, f.users[0].String(), series.CreatedBy)
	_, err = f.db.Exec(`UPDATE document_permissions SET is_allowed=false WHERE subject_key=$1`, f.users[1].String())
	require.NoError(t, err)
	_, err = f.service(1).StopSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: series.Version, Reason: "Пауза"})
	require.ErrorIs(t, err, models.ErrForbidden)
}

func TestDeadlineExecutorDirectoryEligibilityAndDefaultDateIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	_, err := f.db.Exec(`UPDATE users SET first_name='Renamed',is_document_participant=false WHERE id=$1`, f.users[2])
	require.NoError(t, err)
	eligible, err := f.service(0).ListExecutors(ctx, false)
	require.NoError(t, err)
	for _, e := range eligible {
		require.NotEqual(t, f.users[2].String(), e.ID)
	}
	_, err = f.service(0).Create(ctx, f.request())
	require.Error(t, err)
	all, err := f.service(0).ListExecutors(ctx, true)
	require.NoError(t, err)
	count := 0
	for _, e := range all {
		if e.UserID == f.users[2].String() {
			count++
			require.Equal(t, "Test Renamed", e.Name)
			require.False(t, e.IsActive)
		}
	}
	require.Equal(t, 1, count)
	_, err = f.service(0).SaveExternalExecutor(ctx, f.users[2].String(), models.ExternalDeadlineExecutorRequest{Name: "Не изменять пользователя", Version: 1, IsActive: true})
	require.Error(t, err)
	_, err = f.db.Exec(`UPDATE users SET is_document_participant=true WHERE id=$1`, f.users[2])
	require.NoError(t, err)
	c, err := f.service(0).Create(ctx, f.request())
	require.NoError(t, err)
	future := time.Now().AddDate(0, 0, 2)
	_, err = f.service(1).Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: models.OrganizationToday(future)})
	require.Error(t, err)
	completed, err := f.service(1).Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version})
	require.NoError(t, err)
	require.Equal(t, models.OrganizationToday(*completed.CompletedAt), completed.ExecutedOn)
}

func TestDeadlineOptionalSubjectIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	req := f.request()
	req.Subject = ""
	control, err := f.service(0).Create(ctx, req)
	require.NoError(t, err)
	require.Empty(t, control.Subject)
	series, err := f.service(1).CreateSeries(ctx, models.DeadlineSeriesRequest{Control: req, Recurrence: models.DeadlineRecurrence{IntervalUnit: "week", IntervalValue: 1, DayRule: "same_day"}})
	require.NoError(t, err)
	require.Empty(t, series.Subject)
}

func TestDeadlineGlobalReadScopeAndNotificationRevocationIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	control, err := f.service(0).Create(ctx, f.request())
	require.NoError(t, err)
	for _, edge := range [][2]int{{0, 1}, {1, 2}, {2, 0}} {
		_, err = f.db.Exec(`INSERT INTO document_links(source_document_id,target_document_id,link_type,created_by) VALUES($1,$2,'related',$3)`, f.docs[edge[0]], f.docs[edge[1]], f.users[0])
		require.NoError(t, err)
	}
	wrapped := database.Wrap(f.db)
	access := repository.NewDocumentAccessRepository(wrapped)
	readable, err := access.GetReadableDocumentIDs(f.users[2], f.docs[:])
	require.NoError(t, err)
	require.Len(t, readable, 3)
	scopes := map[models.DocumentKind]models.DocumentAccessScope{}
	for _, spec := range models.AllDocumentKindSpecs() {
		scopes[spec.Code] = models.DocumentAccessScope{Restricted: true, UserID: f.users[2].String()}
	}
	recent := repository.NewWorkspaceRepository(wrapped)
	items, err := recent.RecentDocuments(scopes)
	require.NoError(t, err)
	require.Len(t, items, 3)
	events := repository.NewUserEventRepository(wrapped)
	for i, doc := range f.docs {
		err = events.CreateFromOutbox(models.CreateUserEventRequest{RecipientUserID: f.users[2], DocumentID: doc, DocumentKind: "incoming_letter", DocumentNumber: fmt.Sprint(i), EntityType: "control", EventType: "deadline_changed", Title: "Изменение", Message: "Контроль"}, fmt.Sprintf("visibility-%d", i))
		require.NoError(t, err)
	}
	list, err := events.GetList(f.users[2], models.UserEventFilter{PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, 3, list.TotalCount)
	require.Len(t, list.Items, 1)
	unread, err := events.CountUnread(f.users[2])
	require.NoError(t, err)
	require.Equal(t, 3, unread)
	_, err = f.service(1).Cancel(ctx, control.ID, models.CancelDeadlineRequest{Version: control.Version, Reason: "Отменено"})
	require.NoError(t, err)
	readable, err = access.GetReadableDocumentIDs(f.users[2], f.docs[:])
	require.NoError(t, err)
	require.Empty(t, readable)
	items, err = recent.RecentDocuments(scopes)
	require.NoError(t, err)
	require.Empty(t, items)
	list, err = events.GetList(f.users[2], models.UserEventFilter{})
	require.NoError(t, err)
	require.Zero(t, list.TotalCount)
	require.Empty(t, list.Items)
	unread, err = events.CountUnread(f.users[2])
	require.NoError(t, err)
	require.Zero(t, unread)
}

func TestDeadlineSeriesPreviewAndQueueFiltersIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	svc := f.service(0)
	req := models.DeadlineSeriesRequest{Control: f.request(), Recurrence: models.DeadlineRecurrence{IntervalUnit: "month", IntervalValue: 1, DayRule: "fixed", DayOfMonth: 10}}
	series, err := svc.CreateSeries(ctx, req)
	require.NoError(t, err)
	series, err = svc.StopSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: series.Version, Reason: "Пауза"})
	require.NoError(t, err)
	preview, err := svc.PreviewResume(ctx, series.ID)
	require.NoError(t, err)
	require.False(t, preview.CreatesIteration)
	require.Equal(t, "2026-10-10", preview.DueDate)
	queue, err := svc.List(ctx, models.DeadlineControlFilter{ManageOnly: true, SeriesStoppedOnly: true, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, 1, queue.Total)
	require.Len(t, queue.Items, 1)
	mine, err := f.service(2).List(ctx, models.DeadlineControlFilter{ManageOnly: true})
	require.NoError(t, err)
	require.Zero(t, mine.Total)
	_, err = f.service(2).PreviewResume(ctx, series.ID)
	require.ErrorIs(t, err, models.ErrForbidden)
	_, err = svc.Complete(ctx, queue.Items[0].ID, models.CompleteDeadlineRequest{Version: queue.Items[0].Version, ExecutedOn: "2026-10-10"})
	require.NoError(t, err)
	preview, err = svc.PreviewResume(ctx, series.ID)
	require.NoError(t, err)
	require.True(t, preview.CreatesIteration)
	require.Equal(t, "2026-11-10", preview.DueDate)
	// The preview is read-only, and confirmation must match its calculated date.
	_, err = svc.ResumeSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: preview.Version, ExpectedNextDueDate: "2026-11-11"})
	require.Error(t, err)
	_, err = svc.ResumeSeries(ctx, series.ID, models.DeadlineSeriesActionRequest{Version: preview.Version, ExpectedNextDueDate: preview.DueDate})
	require.NoError(t, err)
	queue, err = svc.List(ctx, models.DeadlineControlFilter{ManageOnly: true, SeriesStoppedOnly: true})
	require.NoError(t, err)
	require.Zero(t, queue.Total)
}
