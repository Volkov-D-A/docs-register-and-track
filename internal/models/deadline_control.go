package models

import (
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/google/uuid"
)

const (
	DocumentActionRegister         DocumentKindAction = "register"
	DocumentActionControlDeadlines DocumentKindAction = "control_deadlines"
	DeadlineActive                                    = "active"
	DeadlineCompleted                                 = "completed"
	DeadlineCancelled                                 = "cancelled"
	OrganizationTimezone                              = "Asia/Yekaterinburg"
)

// Calendar dates use ISO strings, independently of the client or database zone.
func OrganizationToday(now time.Time) string {
	location, _ := time.LoadLocation(OrganizationTimezone)
	return now.In(location).Format(time.DateOnly)
}

func ParseDeadlineDate(value string) (time.Time, error) {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil || date.Format(time.DateOnly) != value || date.Year() < 1 {
		return time.Time{}, NewBadRequest("дата должна иметь формат ГГГГ-ММ-ДД")
	}
	return date, nil
}

type DeadlineExecutor struct {
	ID          string `json:"id"`
	UserID      string `json:"userId,omitempty"`
	Name        string `json:"name"`
	Institution string `json:"institution"`
	Position    string `json:"position"`
	IsActive    bool   `json:"isActive"`
	Version     int64  `json:"version"`
}

type ExternalDeadlineExecutorRequest struct {
	Name        string `json:"name"`
	Institution string `json:"institution"`
	Position    string `json:"position"`
	IsActive    bool   `json:"isActive"`
	Version     int64  `json:"version"`
}

func (r *ExternalDeadlineExecutorRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Institution = strings.TrimSpace(r.Institution)
	r.Position = strings.TrimSpace(r.Position)
	if r.Name == "" || len([]rune(r.Name)) > 500 || len([]rune(r.Institution)) > 500 || len([]rune(r.Position)) > 500 {
		return NewBadRequest("укажите ФИО исполнителя; поля должны быть не длиннее 500 символов")
	}
	return nil
}

type DocumentDeadlineControl struct {
	ID                 string     `json:"id"`
	DocumentID         string     `json:"documentId"`
	DocumentKind       string     `json:"documentKind"`
	DocumentNumber     string     `json:"documentNumber"`
	CreatedBy          string     `json:"createdBy"`
	ExecutorID         string     `json:"executorId"`
	CoExecutorIDs      []string   `json:"coExecutorIds"`
	Subject            string     `json:"subject"`
	DueDate            string     `json:"dueDate"`
	Status             string     `json:"status"`
	ExecutedOn         string     `json:"executedOn,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	CompletedBy        string     `json:"completedBy,omitempty"`
	Comment            string     `json:"comment"`
	CancellationReason string     `json:"cancellationReason"`
	SeriesID           string     `json:"seriesId,omitempty"`
	Iteration          int        `json:"iteration"`
	PlannedDueDate     string     `json:"plannedDueDate,omitempty"`
	Version            int64      `json:"version"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	CanManage          bool       `json:"canManage"`
}

func (c DocumentDeadlineControl) IsOverdue(now time.Time) bool {
	return c.Status == DeadlineActive && c.DueDate < OrganizationToday(now)
}
func (c DocumentDeadlineControl) WasExecutedLate() bool {
	return c.Status == DeadlineCompleted && c.ExecutedOn > c.DueDate
}

type DeadlineControlRequest struct {
	DocumentID    string   `json:"documentId"`
	ExecutorID    string   `json:"executorId"`
	CoExecutorIDs []string `json:"coExecutorIds"`
	Subject       string   `json:"subject"`
	DueDate       string   `json:"dueDate"`
	Version       int64    `json:"version"`
	ChangeReason  string   `json:"changeReason"`
}

func (r *DeadlineControlRequest) Validate() error {
	r.Subject = strings.TrimSpace(r.Subject)
	r.ChangeReason = strings.TrimSpace(r.ChangeReason)
	if len([]rune(r.Subject)) > 2000 || len([]rune(r.ChangeReason)) > 2000 {
		return NewBadRequest("предмет контроля и причина изменения должны быть не длиннее 2000 символов")
	}
	if _, err := ParseDeadlineDate(r.DueDate); err != nil {
		return err
	}
	for _, target := range []*string{&r.DocumentID, &r.ExecutorID} {
		parsed, err := uuid.Parse(*target)
		if err != nil || parsed == uuid.Nil {
			return NewBadRequest("неверный идентификатор документа или исполнителя")
		}
		*target = parsed.String()
	}
	for i, id := range r.CoExecutorIDs {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil {
			return NewBadRequest("неверный идентификатор соисполнителя")
		}
		r.CoExecutorIDs[i] = parsed.String()
	}
	seen := map[string]bool{r.ExecutorID: true}
	for _, id := range r.CoExecutorIDs {
		if seen[id] {
			return NewBadRequest("исполнитель и соисполнители не должны повторяться")
		}
		seen[id] = true
	}
	if len(r.CoExecutorIDs) > 100 {
		return NewBadRequest("допустимо не более 100 соисполнителей")
	}
	return nil
}

type CompleteDeadlineRequest struct {
	Version    int64  `json:"version"`
	ExecutedOn string `json:"executedOn"`
	Comment    string `json:"comment"`
}

type CancelDeadlineRequest struct {
	Version int64  `json:"version"`
	Reason  string `json:"reason"`
}

type DeadlineRecurrence struct {
	IntervalUnit  string `json:"intervalUnit"`
	IntervalValue int    `json:"intervalValue"`
	DayRule       string `json:"dayRule"`
	DayOfMonth    int    `json:"dayOfMonth"`
}

func (r DeadlineRecurrence) Validate() error {
	if r.IntervalValue < 1 || r.IntervalValue > 3650 {
		return NewBadRequest("интервал повторения должен быть от 1 до 3650")
	}
	switch r.IntervalUnit {
	case "day", "week":
		if r.DayRule != "same_day" || r.DayOfMonth != 0 {
			return NewBadRequest("для дней и недель не задаётся число месяца")
		}
	case "month", "year":
		if (r.DayRule != "fixed" && r.DayRule != "last_day") || (r.DayRule == "fixed" && (r.DayOfMonth < 1 || r.DayOfMonth > 31)) || (r.DayRule == "last_day" && r.DayOfMonth != 0) {
			return NewBadRequest("задайте число месяца либо последний день")
		}
	default:
		return NewBadRequest("неизвестная периодичность")
	}
	return nil
}

func (r DeadlineRecurrence) Next(planned string) (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	date, err := ParseDeadlineDate(planned)
	if err != nil {
		return "", err
	}
	if r.IntervalUnit == "day" {
		date = date.AddDate(0, 0, r.IntervalValue)
	} else if r.IntervalUnit == "week" {
		date = date.AddDate(0, 0, 7*r.IntervalValue)
	} else {
		first := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
		if r.IntervalUnit == "year" {
			first = first.AddDate(r.IntervalValue, 0, 0)
		} else {
			first = first.AddDate(0, r.IntervalValue, 0)
		}
		lastDay := first.AddDate(0, 1, -1).Day()
		day := r.DayOfMonth
		if r.DayRule == "last_day" || day > lastDay {
			day = lastDay
		}
		date = first.AddDate(0, 0, day-1)
	}
	if date.Year() > 9999 {
		return "", NewBadRequest("следующий срок выходит за диапазон календарных дат")
	}
	return date.Format(time.DateOnly), nil
}

func (r DeadlineRecurrence) Matches(date string) bool {
	value, err := ParseDeadlineDate(date)
	if err != nil {
		return false
	}
	if r.IntervalUnit == "day" || r.IntervalUnit == "week" {
		return true
	}
	last := time.Date(value.Year(), value.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := r.DayOfMonth
	if r.DayRule == "last_day" || day > last {
		day = last
	}
	return value.Day() == day
}

type DeadlineSeriesRequest struct {
	Control    DeadlineControlRequest `json:"control"`
	Recurrence DeadlineRecurrence     `json:"recurrence"`
	Version    int64                  `json:"version"`
}

type DocumentDeadlineSeries struct {
	ID               string             `json:"id"`
	DocumentID       string             `json:"documentId"`
	CreatedBy        string             `json:"createdBy"`
	ExecutorID       string             `json:"executorId"`
	CoExecutorIDs    []string           `json:"coExecutorIds"`
	Subject          string             `json:"subject"`
	Recurrence       DeadlineRecurrence `json:"recurrence"`
	FirstDueDate     string             `json:"firstDueDate"`
	Active           bool               `json:"active"`
	StopReason       string             `json:"stopReason"`
	CurrentControlID string             `json:"currentControlId"`
	CurrentIteration int                `json:"currentIteration"`
	Version          int64              `json:"version"`
}

type DeadlineSeriesActionRequest struct {
	Version             int64  `json:"version"`
	Reason              string `json:"reason"`
	ExpectedNextDueDate string `json:"expectedNextDueDate"`
}

type DeadlineControlFilter struct {
	ManageOnly        bool   `json:"manageOnly"`
	SeriesStoppedOnly bool   `json:"seriesStoppedOnly"`
	DocumentID        string `json:"documentId"`
	SeriesID          string `json:"seriesId"`
	MineOnly          bool   `json:"mineOnly"`
	Status            string `json:"status"`
	OverdueOnly       bool   `json:"overdueOnly"`
	DueFrom           string `json:"dueFrom"`
	DueTo             string `json:"dueTo"`
	Page              int    `json:"page"`
	PageSize          int    `json:"pageSize"`
}

type DeadlineControlPage struct {
	Items    []DocumentDeadlineControl `json:"items"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

// Resume preview is computed by the server from the current planned calendar.
type DeadlineSeriesResumePreview struct {
	Version          int64  `json:"version"`
	DueDate          string `json:"dueDate"`
	CreatesIteration bool   `json:"createsIteration"`
}
