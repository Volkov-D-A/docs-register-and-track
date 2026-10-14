package server

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestReportSchedulePeriods(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	require.NoError(t, err)
	weekly := nextReportOccurrence(time.Date(2026, 3, 2, 9, 0, 0, 0, loc), "weekly", loc)
	require.Equal(t, "2026-03-09 08:00", weekly.Format("2006-01-02 15:04"))
	req, err := scheduledReportRequest(models.ReportScheduleRequest{Frequency: "weekly", Timezone: "Asia/Yekaterinburg", Report: models.ReportRequest{Template: "department_load"}}, weekly)
	require.NoError(t, err)
	require.Equal(t, "2026-03-02", req.StartDate)
	require.Equal(t, "2026-03-08", req.EndDate)
	monthly := nextReportOccurrence(time.Date(2026, 3, 1, 9, 0, 0, 0, loc), "monthly", loc)
	require.Equal(t, "2026-04-01 08:00", monthly.Format("2006-01-02 15:04"))
	req, err = scheduledReportRequest(models.ReportScheduleRequest{Frequency: "monthly", Timezone: "Asia/Yekaterinburg", Report: models.ReportRequest{Template: "organizations"}}, monthly)
	require.NoError(t, err)
	require.Equal(t, "2026-03-01", req.StartDate)
	require.Equal(t, "2026-03-31", req.EndDate)
}

func TestReportExportsPreserveNumbersAndCyrillic(t *testing.T) {
	report := &models.ReportResult{Template: "organizations", StartDate: "2026-02-01", EndDate: "2026-02-28", GeneratedAt: "2026-03-01T08:00:00Z",
		Definition: "Входящие и исходящие", Rows: []models.ReportRow{{Name: "Организация А", Incoming: 2, Outgoing: 3}},
		Summary: models.ReportSummary{Incoming: 2, Outgoing: 3, UniqueDocuments: 4}}
	xlsx, err := reportXLSX(report)
	require.NoError(t, err)
	book, err := excelize.OpenReader(bytes.NewReader(xlsx))
	require.NoError(t, err)
	defer book.Close()
	value, err := book.GetCellValue("Отчёт", "A8")
	require.NoError(t, err)
	require.Equal(t, "Организация А", value)
	value, err = book.GetCellValue("Отчёт", "B8")
	require.NoError(t, err)
	require.Equal(t, "2", value)
	pdf, err := reportPDF(report)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")))
}
