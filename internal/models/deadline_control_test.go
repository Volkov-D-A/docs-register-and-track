package models

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDeadlineCalendarKeepsMonthlyAnchor(t *testing.T) {
	fixed := DeadlineRecurrence{IntervalUnit: "month", IntervalValue: 1, DayRule: "fixed", DayOfMonth: 31}
	current := "2024-01-31"
	for _, expected := range []string{"2024-02-29", "2024-03-31", "2024-04-30", "2024-05-31"} {
		actual, err := fixed.Next(current)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		current = actual
	}
	last := DeadlineRecurrence{IntervalUnit: "month", IntervalValue: 1, DayRule: "last_day"}
	next, err := last.Next("2025-01-31")
	require.NoError(t, err)
	require.Equal(t, "2025-02-28", next)
	yearly := DeadlineRecurrence{IntervalUnit: "year", IntervalValue: 1, DayRule: "fixed", DayOfMonth: 29}
	current = "2024-02-29"
	for _, expected := range []string{"2025-02-28", "2026-02-28", "2027-02-28", "2028-02-29"} {
		next, err := yearly.Next(current)
		require.NoError(t, err)
		require.Equal(t, expected, next)
		current = next
	}
	for _, item := range []struct {
		rule       DeadlineRecurrence
		date, want string
	}{
		{DeadlineRecurrence{IntervalUnit: "day", IntervalValue: 1, DayRule: "same_day"}, "2024-02-28", "2024-02-29"},
		{DeadlineRecurrence{IntervalUnit: "week", IntervalValue: 2, DayRule: "same_day"}, "2026-12-25", "2027-01-08"},
	} {
		next, err := item.rule.Next(item.date)
		require.NoError(t, err)
		require.Equal(t, item.want, next)
	}
}
func TestDeadlineDatesUseOrganizationDay(t *testing.T) {
	before := time.Date(2026, 10, 14, 18, 59, 59, 0, time.UTC)
	after := before.Add(time.Second)
	require.Equal(t, "2026-10-14", OrganizationToday(before))
	require.Equal(t, "2026-10-15", OrganizationToday(after))
	for _, invalid := range []string{"2026-02-29", "2026-1-01", "0000-01-01", "2026-10-14T00:00:00Z", ""} {
		_, err := ParseDeadlineDate(invalid)
		require.Error(t, err)
	}
}
func TestDeadlineRejectsInvalidRecurrenceAndDuplicateExecutors(t *testing.T) {
	for _, rule := range []DeadlineRecurrence{
		{IntervalUnit: "day", IntervalValue: 1, DayRule: "fixed", DayOfMonth: 10},
		{IntervalUnit: "month", IntervalValue: 1, DayRule: "same_day"},
		{IntervalUnit: "month", IntervalValue: 0, DayRule: "last_day"},
		{IntervalUnit: "month", IntervalValue: 1, DayRule: "fixed", DayOfMonth: 32},
	} {
		require.Error(t, rule.Validate())
	}
	id := uuid.NewString()
	req := DeadlineControlRequest{DocumentID: uuid.NewString(), ExecutorID: id, CoExecutorIDs: []string{id}, Subject: "Контроль", DueDate: "2026-10-14"}
	require.Error(t, req.Validate())
}

func TestDeadlineControlRequiresSubject(t *testing.T) {
	for _, subject := range []string{"", "   ", "\t\n"} {
		req := DeadlineControlRequest{DocumentID: uuid.NewString(), ExecutorID: uuid.NewString(), Subject: subject, DueDate: "2026-10-20"}
		err := req.Validate()
		require.ErrorContains(t, err, "укажите предмет контроля")
	}
	req := DeadlineControlRequest{DocumentID: uuid.NewString(), ExecutorID: uuid.NewString(), Subject: "  Ответ на письмо  ", DueDate: "2026-10-20"}
	require.NoError(t, req.Validate())
	require.Equal(t, "Ответ на письмо", req.Subject)
}
