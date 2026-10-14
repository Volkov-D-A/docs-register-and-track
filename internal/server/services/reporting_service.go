package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type ReportReader interface {
	Run(context.Context, models.ReportRequest, time.Time, time.Time) ([]models.ReportRow, error)
	UniqueDocumentCount(context.Context, time.Time, time.Time, models.ReportRequest) (int, error)
	FilterDescription(context.Context, models.ReportRequest) (string, error)
	OpenOverdueCount(context.Context, time.Time, models.ReportRequest) (int, error)
	FilterOptions() (*models.ReportFilters, error)
}

type ReportingService struct {
	repo ReportReader
}

func NewReportingService(repo ReportReader) *ReportingService { return &ReportingService{repo: repo} }

func (s *ReportingService) FilterOptions() (*models.ReportFilters, error) {
	return s.repo.FilterOptions()
}

func ValidateReportRequest(req *models.ReportRequest) (time.Time, time.Time, string, error) {
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if len(req.Timezone) > 100 {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("некорректный часовой пояс")
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("некорректный часовой пояс")
	}
	definitions := map[string]string{
		"organizations":   "Число зарегистрированных входящих и исходящих писем по организациям. Одно входящее письмо может учитываться у нескольких организаций.",
		"department_load": "Число созданных поручений основным исполнителям по месяцам. Подразделение определяется на момент формирования отчёта.",
		"execution_time":  "Среднее календарное время от создания поручения до последней сдачи исполнителем, в сутках. Период определяется датой сдачи.",
		"overdue_rate":    "Доля сданных после срока среди сданных в период поручений с установленным сроком, в процентах.",
	}
	definition, ok := definitions[req.Template]
	if !ok {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("неизвестный вид отчёта")
	}
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("некорректная дата начала")
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("некорректная дата окончания")
	}
	if end.Before(start) || end.After(start.AddDate(3, 0, 0)) {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("период должен быть от одного дня до трёх лет")
	}
	if req.OrganizationID != "" {
		if req.Template != "organizations" {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("фильтр организации недоступен для этого отчёта")
		}
		if _, err := uuid.Parse(req.OrganizationID); err != nil {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("некорректная организация")
		}
	}
	if req.DepartmentID != "" {
		if req.Template == "organizations" {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("фильтр подразделения недоступен для этого отчёта")
		}
		if _, err := uuid.Parse(req.DepartmentID); err != nil {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("некорректное подразделение")
		}
	}
	if req.KindCode != "" {
		if _, ok := models.GetDocumentKindSpec(models.DocumentKind(req.KindCode)); !ok {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("неизвестный вид документа")
		}
		if req.Template == "organizations" && req.KindCode != "incoming_letter" && req.KindCode != "outgoing_letter" {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("вид документа недоступен для этого отчёта")
		}
	}
	for _, item := range []struct{ value, message string }{{req.NomenclatureID, "некорректная номенклатура"}, {req.UserID, "некорректный пользователь"}} {
		if item.value != "" {
			if _, err := uuid.Parse(item.value); err != nil {
				return time.Time{}, time.Time{}, "", models.NewBadRequest(item.message)
			}
		}
	}
	if req.Status != "" {
		if req.Template == "organizations" {
			return time.Time{}, time.Time{}, "", models.NewBadRequest("фильтр статуса недоступен для этого отчёта")
		}
		switch req.Status {
		case "new", "in_progress", "completed", "finished", "returned", "cancelled":
		default:
			return time.Time{}, time.Time{}, "", models.NewBadRequest("неизвестный статус поручения")
		}
	}
	if req.Template == "organizations" && req.OnlyWithDeadline {
		return time.Time{}, time.Time{}, "", models.NewBadRequest("фильтр срока недоступен для этого отчёта")
	}
	return start, end, definition, nil
}

func (s *ReportingService) Run(ctx context.Context, req models.ReportRequest) (*models.ReportResult, error) {
	start, end, definition, err := ValidateReportRequest(&req)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Run(ctx, req, start, end)
	if err != nil {
		return nil, err
	}
	filters, err := s.repo.FilterDescription(ctx, req)
	if err != nil {
		return nil, err
	}
	summary := models.ReportSummary{}
	weighted := 0.0
	for _, row := range rows {
		summary.Count += row.Count
		summary.Incoming += row.Incoming
		summary.Outgoing += row.Outgoing
		summary.Overdue += row.Overdue
		weighted += row.Metric * float64(row.Count)
	}
	if summary.Count > 0 {
		if req.Template == "execution_time" {
			summary.Metric = weighted / float64(summary.Count)
		}
		if req.Template == "overdue_rate" {
			summary.Metric = 100 * float64(summary.Overdue) / float64(summary.Count)
		}
	}
	if req.Template == "organizations" {
		summary.UniqueDocuments, err = s.repo.UniqueDocumentCount(ctx, start, end, req)
		if err != nil {
			return nil, err
		}
	}
	if req.Template == "overdue_rate" {
		loc, _ := time.LoadLocation(req.Timezone)
		summary.OpenOverdue, err = s.repo.OpenOverdueCount(ctx, time.Now().In(loc), req)
		if err != nil {
			return nil, err
		}
	}
	return &models.ReportResult{Template: req.Template, StartDate: req.StartDate, EndDate: req.EndDate,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339), Definition: definition, Filters: filters, Version: 1, Timezone: req.Timezone, Rows: rows, Summary: summary}, nil
}
