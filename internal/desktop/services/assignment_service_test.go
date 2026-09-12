package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
	"github.com/stretchr/testify/require"
)

type assignmentClientStub struct {
	serverclient.AssignmentClient
	create func(context.Context, string, string, string, string, []string) (*dto.Assignment, error)
	status func(context.Context, string, string, string) (*dto.Assignment, error)
	series func(context.Context, models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error)
}

func (c assignmentClientStub) CreateAssignment(ctx context.Context, doc, executor, content, deadline string, coExecutors []string) (*dto.Assignment, error) {
	return c.create(ctx, doc, executor, content, deadline, coExecutors)
}
func (c assignmentClientStub) UpdateAssignmentStatus(ctx context.Context, id, status, report string) (*dto.Assignment, error) {
	return c.status(ctx, id, status, report)
}
func (c assignmentClientStub) CreateAssignmentSeries(ctx context.Context, request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	return c.series(ctx, request)
}

func TestAssignmentAdapterPreservesServerResultsAndRequestContext(t *testing.T) {
	var requestContext context.Context
	want := &dto.Assignment{ID: "server-assignment"}
	client := assignmentClientStub{create: func(ctx context.Context, doc, executor, content, deadline string, coExecutors []string) (*dto.Assignment, error) {
		requestContext = ctx
		require.Equal(t, []string{"document", "executor", "Проверить документ", "2026-09-30"}, []string{doc, executor, content, deadline})
		require.Equal(t, []string{"co-executor"}, coExecutors)
		limit, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(30*time.Second), limit, time.Second)
		require.NoError(t, ctx.Err())
		return want, nil
	}}
	result, err := NewAssignmentService(client).Create("document", "executor", "Проверить документ", "2026-09-30", []string{"co-executor"})
	require.NoError(t, err)
	require.Same(t, want, result)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestAssignmentAdapterLeavesStatusValidationToServer(t *testing.T) {
	denied := models.NewForbidden("нет права на приемку")
	client := assignmentClientStub{status: func(_ context.Context, id, status, report string) (*dto.Assignment, error) {
		require.Equal(t, []string{"assignment", "finished", "  Отчет  "}, []string{id, status, report})
		return nil, denied
	}}
	result, err := NewAssignmentService(client).UpdateStatus("assignment", "finished", "  Отчет  ")
	require.Nil(t, result)
	require.ErrorIs(t, err, denied)
}

func TestAssignmentAdapterPreservesSeriesCalendarRule(t *testing.T) {
	request := models.AssignmentSeriesRequest{DocumentID: "document", ExecutorID: "executor", Content: "Ежемесячный отчет", FirstDeadline: "2026-09-30", IntervalUnit: "month", IntervalValue: 1, DayRule: "last_day", CoExecutorIDs: []string{"co-executor"}}
	want := &dto.AssignmentSeries{ID: "series"}
	client := assignmentClientStub{series: func(_ context.Context, got models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
		require.Equal(t, request, got)
		return want, nil
	}}
	result, err := NewAssignmentService(client).CreateSeries(request)
	require.NoError(t, err)
	require.Same(t, want, result)
}

func TestAssignmentAdapterMissingClient(t *testing.T) {
	s := NewAssignmentService(nil)
	_, err := s.Create("", "", "", "", nil)
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.Update("", "", "", "", nil)
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.UpdateStatus("", "", "")
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.GetByID("")
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.GetList(models.AssignmentFilter{})
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	require.ErrorIs(t, s.Delete(""), errAssignmentClientNotConfigured)
	_, err = s.CreateSeries(models.AssignmentSeriesRequest{})
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.UpdateSeries("", models.AssignmentSeriesRequest{})
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.GetSeries("")
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	_, err = s.GetSeriesHistory("")
	require.ErrorIs(t, err, errAssignmentClientNotConfigured)
	require.ErrorIs(t, s.CancelSeries(""), errAssignmentClientNotConfigured)
}
