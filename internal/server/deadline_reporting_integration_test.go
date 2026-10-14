package server

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/repository"
	services "github.com/Volkov-D-A/docs-register-and-track/internal/server/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeadlineStatisticsAndReportsScopeIntegration(t *testing.T) {
	f := newDeadlineFixture(t)
	ctx := context.Background()
	wrapped := database.Wrap(f.db)
	_, err := f.db.Exec(`UPDATE documents SET registration_date='2026-09-15' WHERE id=$1`, f.docs[0])
	require.NoError(t, err)
	for _, executed := range []string{"2026-10-09", "2026-10-12"} {
		req := f.request()
		req.ExecutorID = f.external
		req.CoExecutorIDs = nil
		c, err := f.service(0).Create(ctx, req)
		require.NoError(t, err)
		_, err = f.db.Exec(`UPDATE document_deadline_controls SET created_at='2026-10-01T20:30:00Z' WHERE id=$1`, c.ID)
		require.NoError(t, err)
		_, err = f.service(0).Complete(ctx, c.ID, models.CompleteDeadlineRequest{Version: c.Version, ExecutedOn: executed})
		require.NoError(t, err)
	}
	_, err = f.db.Exec(`INSERT INTO document_deadline_controls(id,document_id,created_by,executor_id,subject,due_date) VALUES($1,$2,$3,$4,'Hidden','2026-10-10')`, uuid.New(), f.docs[3], f.users[0], f.users[4])
	require.NoError(t, err)
	stats := repository.NewStatisticsRepository(wrapped).ForUser(f.users[0])
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)
	total, err := stats.GetDocumentTotalByYear(start, end)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	_, err = stats.GetMonthlyDocumentCountsByKind(start, end)
	require.NoError(t, err)
	_, err = stats.GetMonthlyDocumentCountsByRegistrar(start, end)
	require.NoError(t, err)
	_, err = stats.GetDocumentReport(start, end, "nomenclature", "", "", "")
	require.NoError(t, err)
	monthly, err := stats.GetDeadlineMonthlyOverview(start, end)
	require.NoError(t, err)
	require.Equal(t, 2, monthly[9].Total)
	require.Equal(t, 1, monthly[9].Overdue)
	executor, err := stats.GetDeadlineMonthlyByExecutor(start, end)
	require.NoError(t, err)
	require.Len(t, executor, 1)
	require.Equal(t, "Внешний исполнитель", executor[0].CategoryName)
	_, err = stats.GetDeadlineOverdueRating(start, end)
	require.NoError(t, err)
	states, err := stats.GetDeadlineStatusCounts()
	require.NoError(t, err)
	require.Len(t, states, 1)
	require.Equal(t, "completed", states[0].Key)
	require.Equal(t, 2, states[0].Count)
	report, err := stats.GetDeadlineReport(start, end, true, f.external)
	require.NoError(t, err)
	require.Len(t, report, 1)
	require.Equal(t, 1, report[0].Count)
	people, err := stats.GetExecutorOptions()
	require.NoError(t, err)
	require.Len(t, people, 1)
	require.Equal(t, f.external, people[0].Value)
	empty := repository.NewStatisticsRepository(wrapped).ForUser(f.users[3])
	total, err = empty.GetDocumentTotalByYear(start, end)
	require.NoError(t, err)
	require.Zero(t, total)
	people, err = empty.GetExecutorOptions()
	require.NoError(t, err)
	require.Empty(t, people)
	reports := services.NewReportingService(repository.NewReportingRepository(wrapped).ForUser(f.users[0]))
	for _, template := range []string{"department_load", "execution_time", "overdue_rate"} {
		result, err := reports.Run(ctx, models.ReportRequest{Template: template, StartDate: "2026-10-01", EndDate: "2026-10-31", PeriodBasis: "executed_on"})
		require.NoError(t, err)
		require.Equal(t, 2, result.Summary.Count)
		if template == "overdue_rate" {
			require.Equal(t, 1, result.Summary.Overdue)
			require.Equal(t, float64(50), result.Summary.Metric)
		}
	}
	for _, basis := range []string{"registration_date", "due_date", "executed_on"} {
		result, err := reports.Run(ctx, models.ReportRequest{Template: "department_load", StartDate: "2026-09-01", EndDate: "2026-09-30", PeriodBasis: basis})
		require.NoError(t, err)
		if basis == "registration_date" {
			require.Equal(t, 2, result.Summary.Count)
		} else {
			require.Zero(t, result.Summary.Count)
		}
	}
	_, err = f.db.Exec(`INSERT INTO organizations(id,name) VALUES($1,'Visible organization')`, uuid.New())
	require.NoError(t, err)
	_, err = f.db.Exec(`INSERT INTO document_correspondent_registrations(document_id,registration_number,registration_date,correspondent_org_id,position) SELECT $1,'A','2026-09-15',id,1 FROM organizations WHERE name='Visible organization'`, f.docs[0])
	require.NoError(t, err)
	result, err := reports.Run(ctx, models.ReportRequest{Template: "organizations", StartDate: "2026-10-01", EndDate: "2026-10-31", PeriodBasis: "due_date"})
	require.NoError(t, err)
	require.Equal(t, 1, result.Summary.UniqueDocuments)
	require.Equal(t, 1, result.Summary.Incoming)
	// The whole linked graph contributes; removal revokes every aggregate and filter.
	_, err = f.db.Exec(`INSERT INTO document_links(source_document_id,target_document_id,link_type,created_by) VALUES($1,$2,'related',$3)`, f.docs[0], f.docs[3], f.users[0])
	require.NoError(t, err)
	people, err = stats.GetExecutorOptions()
	require.NoError(t, err)
	require.Len(t, people, 2)
	_, err = f.db.Exec(`DELETE FROM document_links WHERE source_document_id=$1 AND target_document_id=$2`, f.docs[0], f.docs[3])
	require.NoError(t, err)
	people, err = stats.GetExecutorOptions()
	require.NoError(t, err)
	require.Len(t, people, 1)
	options, err := services.NewReportingService(repository.NewReportingRepository(wrapped).ForUser(f.users[3])).FilterOptions()
	require.NoError(t, err)
	require.Empty(t, options.Executors)
	require.Empty(t, options.Users)
	require.Empty(t, options.Organizations)
	require.Empty(t, options.Nomenclature)
}
