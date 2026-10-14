package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/effects"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	"github.com/google/uuid"
)

type DeadlineControlService struct {
	store     ports.DeadlineControlStore
	principal ports.DocumentAccessPrincipal
	now       func() time.Time
}

func NewDeadlineControlService(store ports.DeadlineControlStore, principal ports.DocumentAccessPrincipal) *DeadlineControlService {
	return &DeadlineControlService{store: store, principal: principal, now: time.Now}
}
func (s *DeadlineControlService) actor() (uuid.UUID, error) {
	if err := s.principal.RequireAuthenticated(); err != nil {
		return uuid.Nil, err
	}
	return s.principal.GetCurrentUserUUID()
}
func validateDeadlineID(id string) error {
	value, err := uuid.Parse(id)
	if err != nil || value == uuid.Nil {
		return models.NewBadRequest("неверный идентификатор")
	}
	return nil
}
func executorIDs(main string, co []string) []string { return append([]string{main}, co...) }
func requireDeadlineVersion(actual, expected int64) error {
	if expected < 1 || actual != expected {
		return models.NewConflict("запись изменена; перечитайте данные")
	}
	return nil
}
func requireActiveDeadline(c *models.DocumentDeadlineControl) error {
	if c.Status != models.DeadlineActive {
		return models.NewConflict("контроль уже закрыт; завершение окончательное")
	}
	return nil
}
func requireEligibleDeadlineExecutors(tx ports.DeadlineTransaction, ids []string) error {
	eligible, err := tx.EligibleExecutors(ids)
	if err != nil {
		return err
	}
	if !eligible {
		return models.NewBadRequest("исполнитель или соисполнитель недоступен для назначения")
	}
	return nil
}

func (s *DeadlineControlService) ListExecutors(ctx context.Context, includeInactive bool) ([]models.DeadlineExecutor, error) {
	if _, err := s.actor(); err != nil {
		return nil, err
	}
	return s.store.ListExecutors(ctx, includeInactive)
}
func (s *DeadlineControlService) SaveExternalExecutor(ctx context.Context, id string, req models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if id != "" {
		if err = validateDeadlineID(id); err != nil {
			return nil, err
		}
	}
	if err = req.Validate(); err != nil {
		return nil, err
	}
	return s.store.SaveExternalExecutor(ctx, actor, id, req)
}
func (s *DeadlineControlService) Get(ctx context.Context, id string) (*models.DocumentDeadlineControl, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	c, err := s.store.GetControl(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed, err := s.store.CanRead(ctx, actor, c.DocumentID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrForbidden
	}
	c.CanManage, err = s.store.CanManage(ctx, actor, c.DocumentID)
	return c, err
}
func (s *DeadlineControlService) List(ctx context.Context, f models.DeadlineControlFilter) (*models.DeadlineControlPage, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	for _, id := range []string{f.DocumentID, f.SeriesID} {
		if id != "" {
			if err = validateDeadlineID(id); err != nil {
				return nil, err
			}
		}
	}
	if f.Status != "" && f.Status != models.DeadlineActive && f.Status != models.DeadlineCompleted && f.Status != models.DeadlineCancelled {
		return nil, models.NewBadRequest("неверное состояние контроля")
	}
	for _, date := range []string{f.DueFrom, f.DueTo} {
		if date != "" {
			if _, err = models.ParseDeadlineDate(date); err != nil {
				return nil, err
			}
		}
	}
	if f.DueFrom != "" && f.DueTo != "" && f.DueFrom > f.DueTo {
		return nil, models.NewBadRequest("начало периода позже окончания")
	}
	return s.store.ListControls(ctx, actor, f)
}
func newDeadlineControl(actor uuid.UUID, req models.DeadlineControlRequest) *models.DocumentDeadlineControl {
	return &models.DocumentDeadlineControl{ID: uuid.NewString(), DocumentID: req.DocumentID, CreatedBy: actor.String(), ExecutorID: req.ExecutorID, CoExecutorIDs: append([]string{}, req.CoExecutorIDs...), Subject: req.Subject, DueDate: req.DueDate, Status: models.DeadlineActive, CanManage: true}
}
func (s *DeadlineControlService) Create(ctx context.Context, req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = req.Validate(); err != nil {
		return nil, err
	}
	if req.Version != 0 {
		return nil, models.NewBadRequest("версия нового контроля должна быть нулевой")
	}
	c := newDeadlineControl(actor, req)
	err = s.store.WithDocument(ctx, actor, c.DocumentID, func(tx ports.DeadlineTransaction) error {
		if err := requireEligibleDeadlineExecutors(tx, executorIDs(c.ExecutorID, c.CoExecutorIDs)); err != nil {
			return err
		}
		if err := tx.SaveControl(c, true); err != nil {
			return err
		}
		return emitDeadlineChange(tx, actor, "DEADLINE_CONTROL_CREATE", nil, c, req.ChangeReason)
	})
	return c, err
}
func (s *DeadlineControlService) Update(ctx context.Context, id string, req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	if err = req.Validate(); err != nil {
		return nil, err
	}
	current, err := s.store.GetControl(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.DocumentID != req.DocumentID {
		return nil, models.NewBadRequest("документ контроля нельзя изменить")
	}
	var result *models.DocumentDeadlineControl
	err = s.store.WithDocument(ctx, actor, current.DocumentID, func(tx ports.DeadlineTransaction) error {
		c, err := tx.GetControl(id)
		if err != nil {
			return err
		}
		if err = requireDeadlineVersion(c.Version, req.Version); err != nil {
			return err
		}
		if err = requireActiveDeadline(c); err != nil {
			return err
		}
		if c.DueDate != req.DueDate && req.ChangeReason == "" {
			return models.NewBadRequest("для переноса срока обязательна причина")
		}
		if err = requireEligibleDeadlineExecutors(tx, executorIDs(req.ExecutorID, req.CoExecutorIDs)); err != nil {
			return err
		}
		before := *c
		c.ExecutorID = req.ExecutorID
		c.CoExecutorIDs = append([]string{}, req.CoExecutorIDs...)
		c.Subject = req.Subject
		c.DueDate = req.DueDate
		c.CanManage = true
		if err = tx.SaveControl(c, false); err != nil {
			return err
		}
		result = c
		return emitDeadlineChange(tx, actor, "DEADLINE_CONTROL_UPDATE", &before, c, req.ChangeReason)
	})
	return result, err
}
func (s *DeadlineControlService) Complete(ctx context.Context, id string, req models.CompleteDeadlineRequest) (*models.DocumentDeadlineControl, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	now := s.now()
	if req.ExecutedOn == "" {
		req.ExecutedOn = models.OrganizationToday(now)
	}
	if _, err = models.ParseDeadlineDate(req.ExecutedOn); err != nil {
		return nil, err
	}
	if req.ExecutedOn > models.OrganizationToday(now) {
		return nil, models.NewBadRequest("фактическая дата исполнения не может быть будущей")
	}
	req.Comment = strings.TrimSpace(req.Comment)
	if len([]rune(req.Comment)) > 2000 {
		return nil, models.NewBadRequest("комментарий слишком длинный")
	}
	current, err := s.store.GetControl(ctx, id)
	if err != nil {
		return nil, err
	}
	var result *models.DocumentDeadlineControl
	err = s.store.WithDocument(ctx, actor, current.DocumentID, func(tx ports.DeadlineTransaction) error {
		c, err := tx.GetControl(id)
		if err != nil {
			return err
		}
		if err = requireDeadlineVersion(c.Version, req.Version); err != nil {
			return err
		}
		if err = requireActiveDeadline(c); err != nil {
			return err
		}
		before := *c
		c.Status = models.DeadlineCompleted
		c.ExecutedOn = req.ExecutedOn
		recordedAt := s.now()
		c.CompletedAt = &recordedAt
		c.CompletedBy = actor.String()
		c.Comment = req.Comment
		c.CanManage = true
		if err = tx.SaveControl(c, false); err != nil {
			return err
		}
		if err = emitDeadlineChange(tx, actor, "DEADLINE_CONTROL_COMPLETE", &before, c, ""); err != nil {
			return err
		}
		if c.SeriesID != "" {
			series, err := tx.GetSeries(c.SeriesID)
			if err != nil {
				return err
			}
			if series.CurrentControlID != c.ID {
				return models.NewConflict("контроль не является текущей итерацией серии")
			}
			if series.Active {
				eligible, err := tx.EligibleExecutors(executorIDs(series.ExecutorID, series.CoExecutorIDs))
				if err != nil {
					return err
				}
				if !eligible {
					previous := *series
					series.Active = false
					series.StopReason = "Исполнитель или соисполнитель недоступен для назначения"
					if err = tx.SaveSeries(series, false); err != nil {
						return err
					}
					if err = emitDeadlineSeriesChange(tx, actor, "DEADLINE_SERIES_AUTO_STOP", &previous, series); err != nil {
						return err
					}
				} else if err = s.advanceSeries(tx, actor, series, c); err != nil {
					return err
				}
			}
		}
		result = c
		return nil
	})
	return result, err
}
func (s *DeadlineControlService) Cancel(ctx context.Context, id string, req models.CancelDeadlineRequest) (*models.DocumentDeadlineControl, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" || len([]rune(req.Reason)) > 2000 {
		return nil, models.NewBadRequest("укажите причину отмены длиной не более 2000 символов")
	}
	current, err := s.store.GetControl(ctx, id)
	if err != nil {
		return nil, err
	}
	var result *models.DocumentDeadlineControl
	err = s.store.WithDocument(ctx, actor, current.DocumentID, func(tx ports.DeadlineTransaction) error {
		c, err := tx.GetControl(id)
		if err != nil {
			return err
		}
		if err = requireDeadlineVersion(c.Version, req.Version); err != nil {
			return err
		}
		if err = requireActiveDeadline(c); err != nil {
			return err
		}
		before := *c
		c.Status = models.DeadlineCancelled
		c.CancellationReason = req.Reason
		c.CanManage = true
		if err = tx.SaveControl(c, false); err != nil {
			return err
		}
		if err = emitDeadlineChange(tx, actor, "DEADLINE_CONTROL_CANCEL", &before, c, req.Reason); err != nil {
			return err
		}
		if c.SeriesID != "" {
			series, err := tx.GetSeries(c.SeriesID)
			if err != nil {
				return err
			}
			if series.CurrentControlID != c.ID {
				return models.NewConflict("контроль не является текущей итерацией серии")
			}
			previous := *series
			series.Active = false
			series.StopReason = req.Reason
			if err = tx.SaveSeries(series, false); err != nil {
				return err
			}
			if err = emitDeadlineSeriesChange(tx, actor, "DEADLINE_SERIES_STOP", &previous, series); err != nil {
				return err
			}
		}
		result = c
		return nil
	})
	return result, err
}
func (s *DeadlineControlService) CreateSeries(ctx context.Context, req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = req.Control.Validate(); err != nil {
		return nil, err
	}
	if err = req.Recurrence.Validate(); err != nil {
		return nil, err
	}
	if req.Version != 0 || req.Control.Version != 0 {
		return nil, models.NewBadRequest("версия новой серии должна быть нулевой")
	}
	if !req.Recurrence.Matches(req.Control.DueDate) {
		return nil, models.NewBadRequest("первый срок не соответствует календарному правилу")
	}
	c := newDeadlineControl(actor, req.Control)
	series := &models.DocumentDeadlineSeries{ID: uuid.NewString(), DocumentID: c.DocumentID, CreatedBy: actor.String(), ExecutorID: c.ExecutorID, CoExecutorIDs: append([]string{}, c.CoExecutorIDs...), Subject: c.Subject, Recurrence: req.Recurrence, FirstDueDate: c.DueDate, Active: true, CurrentControlID: c.ID, CurrentIteration: 1}
	c.SeriesID = series.ID
	c.Iteration = 1
	c.PlannedDueDate = c.DueDate
	err = s.store.WithDocument(ctx, actor, c.DocumentID, func(tx ports.DeadlineTransaction) error {
		if err := requireEligibleDeadlineExecutors(tx, executorIDs(c.ExecutorID, c.CoExecutorIDs)); err != nil {
			return err
		}
		if err := tx.SaveSeries(series, true); err != nil {
			return err
		}
		if err := tx.SaveControl(c, true); err != nil {
			return err
		}
		if err := emitDeadlineSeriesChange(tx, actor, "DEADLINE_SERIES_CREATE", nil, series); err != nil {
			return err
		}
		return emitDeadlineChange(tx, actor, "DEADLINE_CONTROL_CREATE", nil, c, "")
	})
	return series, err
}
func (s *DeadlineControlService) GetSeries(ctx context.Context, id string) (*models.DocumentDeadlineSeries, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	series, err := s.store.GetSeries(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed, err := s.store.CanRead(ctx, actor, series.DocumentID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrForbidden
	}
	return series, nil
}
func (s *DeadlineControlService) UpdateSeries(ctx context.Context, id string, req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	if err = req.Control.Validate(); err != nil {
		return nil, err
	}
	if err = req.Recurrence.Validate(); err != nil {
		return nil, err
	}
	current, err := s.store.GetSeries(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.DocumentID != req.Control.DocumentID || current.FirstDueDate != req.Control.DueDate {
		return nil, models.NewBadRequest("документ и первый плановый срок серии нельзя изменить")
	}
	var result *models.DocumentDeadlineSeries
	err = s.store.WithDocument(ctx, actor, current.DocumentID, func(tx ports.DeadlineTransaction) error {
		series, err := tx.GetSeries(id)
		if err != nil {
			return err
		}
		if err = requireDeadlineVersion(series.Version, req.Version); err != nil {
			return err
		}
		if err = requireEligibleDeadlineExecutors(tx, executorIDs(req.Control.ExecutorID, req.Control.CoExecutorIDs)); err != nil {
			return err
		}
		before := *series
		series.ExecutorID = req.Control.ExecutorID
		series.CoExecutorIDs = append([]string{}, req.Control.CoExecutorIDs...)
		series.Subject = req.Control.Subject
		series.Recurrence = req.Recurrence
		if err = tx.SaveSeries(series, false); err != nil {
			return err
		}
		result = series
		return emitDeadlineSeriesChange(tx, actor, "DEADLINE_SERIES_UPDATE", &before, series)
	})
	return result, err
}
func (s *DeadlineControlService) StopSeries(ctx context.Context, id string, req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
	return s.changeSeriesActivity(ctx, id, req, false)
}
func (s *DeadlineControlService) ResumeSeries(ctx context.Context, id string, req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
	return s.changeSeriesActivity(ctx, id, req, true)
}
func (s *DeadlineControlService) changeSeriesActivity(ctx context.Context, id string, req models.DeadlineSeriesActionRequest, resume bool) (*models.DocumentDeadlineSeries, error) {
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	if err = validateDeadlineID(id); err != nil {
		return nil, err
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if !resume && (req.Reason == "" || len([]rune(req.Reason)) > 2000) {
		return nil, models.NewBadRequest("укажите причину остановки")
	}
	current, err := s.store.GetSeries(ctx, id)
	if err != nil {
		return nil, err
	}
	var result *models.DocumentDeadlineSeries
	err = s.store.WithDocument(ctx, actor, current.DocumentID, func(tx ports.DeadlineTransaction) error {
		series, err := tx.GetSeries(id)
		if err != nil {
			return err
		}
		if err = requireDeadlineVersion(series.Version, req.Version); err != nil {
			return err
		}
		if series.Active == resume {
			return models.NewConflict("состояние серии уже изменено")
		}
		before := *series
		series.Active = resume
		series.StopReason = req.Reason
		action := "DEADLINE_SERIES_STOP"
		if resume {
			action = "DEADLINE_SERIES_RESUME"
			series.StopReason = ""
			if err = requireEligibleDeadlineExecutors(tx, executorIDs(series.ExecutorID, series.CoExecutorIDs)); err != nil {
				return err
			}
			c, err := tx.GetControl(series.CurrentControlID)
			if err != nil {
				return err
			}
			expected := c.DueDate
			if c.Status != models.DeadlineActive {
				expected, err = series.Recurrence.Next(c.PlannedDueDate)
				if err != nil {
					return err
				}
			}
			if req.ExpectedNextDueDate != expected {
				return models.NewConflict("подтвердите рассчитанный срок после перечитывания серии")
			}
			if c.Status != models.DeadlineActive {
				if err = s.advanceSeries(tx, actor, series, c); err != nil {
					return err
				}
			} else {
				if err = tx.SaveSeries(series, false); err != nil {
					return err
				}
			}
		} else if err = tx.SaveSeries(series, false); err != nil {
			return err
		}
		result = series
		return emitDeadlineSeriesChange(tx, actor, action, &before, series)
	})
	return result, err
}
func (s *DeadlineControlService) advanceSeries(tx ports.DeadlineTransaction, actor uuid.UUID, series *models.DocumentDeadlineSeries, previous *models.DocumentDeadlineControl) error {
	nextDate, err := series.Recurrence.Next(previous.PlannedDueDate)
	if err != nil {
		return err
	}
	c := newDeadlineControl(actor, models.DeadlineControlRequest{DocumentID: series.DocumentID, ExecutorID: series.ExecutorID, CoExecutorIDs: series.CoExecutorIDs, Subject: series.Subject, DueDate: nextDate})
	c.SeriesID = series.ID
	c.Iteration = series.CurrentIteration + 1
	c.PlannedDueDate = nextDate
	if err = tx.SaveControl(c, true); err != nil {
		return err
	}
	series.CurrentControlID = c.ID
	series.CurrentIteration = c.Iteration
	if err = tx.SaveSeries(series, false); err != nil {
		return err
	}
	return emitDeadlineChange(tx, actor, "DEADLINE_CONTROL_CREATE", nil, c, "")
}

func deadlineJournalEvent(actor uuid.UUID, doc, id string, version int64, action string, before, after any, reason string) (models.OutboxEvent, error) {
	details, err := json.Marshal(struct {
		Before any    `json:"before"`
		After  any    `json:"after"`
		Reason string `json:"reason,omitempty"`
	}{before, after, reason})
	if err != nil {
		return models.OutboxEvent{}, err
	}
	documentID, err := uuid.Parse(doc)
	if err != nil {
		return models.OutboxEvent{}, err
	}
	return effects.NewJournalOutboxEvent(fmt.Sprintf("deadline:%s:%d:%s:journal", id, version, action), models.CreateJournalEntryRequest{DocumentID: documentID, UserID: actor, Action: action, Details: string(details)})
}
func emitDeadlineChange(tx ports.DeadlineTransaction, actor uuid.UUID, action string, before, after *models.DocumentDeadlineControl, reason string) error {
	journal, err := deadlineJournalEvent(actor, after.DocumentID, after.ID, after.Version, action, before, after, reason)
	if err != nil {
		return err
	}
	events := []models.OutboxEvent{journal}
	ids, err := tx.RecipientUserIDs(executorIDs(after.ExecutorID, after.CoExecutorIDs))
	if err != nil {
		return err
	}
	controllers, err := tx.ControllerUserIDs()
	if err != nil {
		return err
	}
	// Cancellation and removed assignments no longer grant read. Notify controllers;
	// recipient notifications for other states use only the current assignment set.
	if after.Status == models.DeadlineCancelled {
		ids = nil
	}
	subject := after.Subject
	if subject == "" {
		subject = "Документ " + after.DocumentNumber
	}
	seen := map[uuid.UUID]bool{}
	for _, recipient := range append(ids, controllers...) {
		if recipient == actor || seen[recipient] {
			continue
		}
		seen[recipient] = true
		event, err := effects.NewUserEventOutboxEvent(fmt.Sprintf("deadline:%s:%d:%s:user:%s", after.ID, after.Version, action, recipient), models.CreateUserEventRequest{RecipientUserID: recipient, DocumentID: uuid.MustParse(after.DocumentID), DocumentKind: after.DocumentKind, DocumentNumber: after.DocumentNumber, EntityType: "deadline_control", EventType: strings.ToLower(action), Title: "Контроль срока", Message: fmt.Sprintf("%s; срок %s", subject, after.DueDate)})
		if err != nil {
			return err
		}
		events = append(events, event)
	}
	return tx.Emit(events)
}
func emitDeadlineSeriesChange(tx ports.DeadlineTransaction, actor uuid.UUID, action string, before, after *models.DocumentDeadlineSeries) error {
	journal, err := deadlineJournalEvent(actor, after.DocumentID, after.ID, after.Version, action, before, after, after.StopReason)
	if err != nil {
		return err
	}
	events := []models.OutboxEvent{journal}
	if action == "DEADLINE_SERIES_AUTO_STOP" {
		recipients, err := tx.ControllerUserIDs()
		if err != nil {
			return err
		}
		for _, id := range recipients {
			event, err := effects.NewUserEventOutboxEvent(fmt.Sprintf("deadline-series:%s:%d:user:%s", after.ID, after.Version, id), models.CreateUserEventRequest{RecipientUserID: id, DocumentID: uuid.MustParse(after.DocumentID), EntityType: "deadline_series", EventType: "deadline_series_stopped", Title: "Регулярный контроль остановлен", Message: after.StopReason})
			if err != nil {
				return err
			}
			events = append(events, event)
		}
	}
	return tx.Emit(events)
}

func (s *DeadlineControlService) PreviewResume(ctx context.Context, id string) (*models.DeadlineSeriesResumePreview, error) {
	series, err := s.GetSeries(ctx, id)
	if err != nil {
		return nil, err
	}
	actor, err := s.actor()
	if err != nil {
		return nil, err
	}
	allowed, err := s.store.CanManage(ctx, actor, series.DocumentID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrForbidden
	}
	if series.Active {
		return nil, models.NewConflict("серия уже активна")
	}
	current, err := s.store.GetControl(ctx, series.CurrentControlID)
	if err != nil {
		return nil, err
	}
	due := current.DueDate
	create := current.Status != models.DeadlineActive
	if create {
		due, err = series.Recurrence.Next(current.PlannedDueDate)
		if err != nil {
			return nil, err
		}
	}
	return &models.DeadlineSeriesResumePreview{Version: series.Version, DueDate: due, CreatesIteration: create}, nil
}
