package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	serverservices "github.com/Volkov-D-A/docs-register-and-track/internal/server/services"
)

type reportScheduleStore struct {
	db      *database.DB
	files   ports.FileStorage
	reports *serverservices.ReportingService
}

func nextReportOccurrence(after time.Time, frequency string, loc *time.Location) time.Time {
	local := after.In(loc)
	year, month, day := local.Date()
	if frequency == "monthly" {
		next := time.Date(year, month, 1, 8, 0, 0, 0, loc)
		if !next.After(after) {
			next = time.Date(year, month+1, 1, 8, 0, 0, 0, loc)
		}
		return next
	}
	days := (int(time.Monday) - int(local.Weekday()) + 7) % 7
	next := time.Date(year, month, day+days, 8, 0, 0, 0, loc)
	if !next.After(after) {
		next = next.AddDate(0, 0, 7)
	}
	return next
}

func scheduledReportRequest(req models.ReportScheduleRequest, planned time.Time) (models.ReportRequest, error) {
	loc, err := time.LoadLocation(req.Timezone)
	if err != nil {
		return models.ReportRequest{}, err
	}
	local := planned.In(loc)
	end := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -1)
	start := end
	if req.Frequency == "monthly" {
		start = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, loc)
	} else {
		start = end.AddDate(0, 0, -6)
	}
	report := req.Report
	report.StartDate, report.EndDate = start.Format("2006-01-02"), end.Format("2006-01-02")
	report.Timezone = req.Timezone
	return report, nil
}

func (s *reportScheduleStore) Create(ctx context.Context, owner uuid.UUID, req models.ReportScheduleRequest) (*models.ReportSchedule, error) {
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if req.Frequency != "weekly" && req.Frequency != "monthly" {
		return nil, models.NewBadRequest("периодичность должна быть недельной или месячной")
	}
	if req.Format != "xlsx" && req.Format != "pdf" {
		return nil, models.NewBadRequest("неизвестный формат отчёта")
	}
	if len(req.Timezone) > 100 {
		return nil, models.NewBadRequest("некорректный часовой пояс")
	}
	loc, err := time.LoadLocation(req.Timezone)
	if err != nil {
		return nil, models.NewBadRequest("некорректный часовой пояс")
	}
	testReq := req.Report
	today := time.Now().In(loc).Format("2006-01-02")
	testReq.StartDate, testReq.EndDate, testReq.Timezone = today, today, req.Timezone
	if _, _, _, err := serverservices.ValidateReportRequest(&testReq); err != nil {
		return nil, err
	}
	req.Report.StartDate, req.Report.EndDate = "", ""
	req.Report.Timezone = req.Timezone
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	next := nextReportOccurrence(time.Now(), req.Frequency, loc)
	var id uuid.UUID
	err = s.db.QueryRowContext(ctx, `INSERT INTO report_schedules(owner_id,request,frequency,timezone,format,next_run_at)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, owner, raw, req.Frequency, req.Timezone, req.Format, next).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &models.ReportSchedule{ID: id.String(), Request: req, Enabled: true, NextRunAt: next.Format(time.RFC3339)}, nil
}

func (s *reportScheduleStore) List(ctx context.Context, owner uuid.UUID) ([]models.ReportSchedule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,request,enabled,next_run_at FROM report_schedules WHERE owner_id=$1 ORDER BY created_at DESC LIMIT 100`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ReportSchedule, 0)
	for rows.Next() {
		var item models.ReportSchedule
		var raw []byte
		var next time.Time
		if err := rows.Scan(&item.ID, &raw, &item.Enabled, &next); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Request); err != nil {
			return nil, err
		}
		item.NextRunAt = next.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *reportScheduleStore) Disable(ctx context.Context, owner, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `UPDATE report_schedules SET enabled=false WHERE id=$1 AND owner_id=$2`, id, owner)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return models.NewNotFound("расписание не найдено")
	}
	return nil
}

func (s *reportScheduleStore) RetryRun(ctx context.Context, owner, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `UPDATE report_runs r SET status='pending',error=NULL,lease_until=NULL,finished_at=NULL
		FROM report_schedules s WHERE r.id=$1 AND r.schedule_id=s.id AND s.owner_id=$2 AND s.enabled AND r.status='failed'`, id, owner)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return models.NewNotFound("неудачный запуск не найден")
	}
	return nil
}

func (s *reportScheduleStore) Runs(ctx context.Context, owner, id uuid.UUID) ([]models.ReportRun, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM report_schedules WHERE id=$1 AND owner_id=$2)`, id, owner).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, models.NewNotFound("расписание не найдено")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,planned_at,status,format,definition_version,request,COALESCE(error,'') FROM report_runs WHERE schedule_id=$1 ORDER BY planned_at DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ReportRun, 0)
	for rows.Next() {
		var item models.ReportRun
		var planned time.Time
		var raw []byte
		if err := rows.Scan(&item.ID, &planned, &item.Status, &item.Format, &item.Version, &raw, &item.Error); err != nil {
			return nil, err
		}
		var req models.ReportScheduleRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, err
		}
		item.StartDate, item.EndDate = req.Report.StartDate, req.Report.EndDate
		item.ScheduleID = id.String()
		item.PlannedAt = planned.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *reportScheduleStore) DownloadInfo(ctx context.Context, owner, id uuid.UUID) (string, string, int64, error) {
	var key, format string
	var size int64
	err := s.db.QueryRowContext(ctx, `SELECT r.storage_key,r.format,r.file_size FROM report_runs r
		JOIN report_schedules s ON s.id=r.schedule_id WHERE r.id=$1 AND s.owner_id=$2 AND r.status='ready'`, id, owner).Scan(&key, &format, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", 0, models.NewNotFound("готовый отчёт не найден")
	}
	return key, format, size, err
}

func (s *reportScheduleStore) enqueueDue(ctx context.Context) error {
	for i := 0; i < 20; i++ {
		tx, err := s.db.SQLDB().BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		var id uuid.UUID
		var raw []byte
		var frequency, zone, format string
		var planned time.Time
		err = tx.QueryRowContext(ctx, `SELECT id,request,frequency,timezone,format,next_run_at FROM report_schedules
			WHERE enabled AND next_run_at <= now() ORDER BY next_run_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &raw, &frequency, &zone, &format, &planned)
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return nil
		}
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		loc, err := time.LoadLocation(zone)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		next := nextReportOccurrence(planned, frequency, loc)
		var scheduleReq models.ReportScheduleRequest
		if err = json.Unmarshal(raw, &scheduleReq); err != nil {
			_ = tx.Rollback()
			return err
		}
		resolved, err := scheduledReportRequest(scheduleReq, planned)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		scheduleReq.Report = resolved
		runRaw, err := json.Marshal(scheduleReq)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO report_runs(schedule_id,planned_at,request,format) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, id, planned, runRaw, format)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE report_schedules SET next_run_at=$2 WHERE id=$1`, id, next)
		}
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

type claimedReportRun struct {
	id      uuid.UUID
	owner   uuid.UUID
	planned time.Time
	request models.ReportScheduleRequest
	format  string
}

func (s *reportScheduleStore) claimRun(ctx context.Context) (*claimedReportRun, error) {
	var run claimedReportRun
	var raw []byte
	err := s.db.QueryRowContext(ctx, `UPDATE report_runs SET status='running',lease_until=now()+interval '5 minutes',error=NULL
		WHERE id=(SELECT r.id FROM report_runs r JOIN report_schedules s ON s.id=r.schedule_id
			WHERE (r.status='pending' OR (r.status='running' AND r.lease_until<now())) AND s.enabled
			ORDER BY r.created_at LIMIT 1 FOR UPDATE OF r SKIP LOCKED)
		RETURNING id,planned_at,request,format,(SELECT owner_id FROM report_schedules WHERE id=report_runs.schedule_id)`).Scan(&run.id, &run.planned, &raw, &run.format, &run.owner)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &run.request); err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *reportScheduleStore) completeRun(ctx context.Context, run *claimedReportRun, key string, size int64, runErr error) error {
	status := "ready"
	message := ""
	if runErr != nil {
		status = "failed"
		message = runErr.Error()
		if len(message) > 500 {
			message = message[:500]
		}
	}
	_, err := s.db.ExecContext(ctx, `UPDATE report_runs SET status=$2,storage_key=$3,file_size=$4,error=$5,lease_until=NULL,finished_at=now() WHERE id=$1`, run.id, status, nullableString(key), nullableInt64(size), nullableString(message))
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func (s *reportScheduleStore) generateRun(ctx context.Context, run *claimedReportRun) error {
	var allowed bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN user_system_permissions p ON p.user_id=u.id
		WHERE u.id=$1 AND u.is_active AND p.permission='reports' AND p.is_allowed)`, run.owner).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("право на отчёты отозвано или пользователь отключён")
	}
	req := run.request.Report
	if req.StartDate == "" || req.EndDate == "" {
		var err error
		req, err = scheduledReportRequest(run.request, run.planned)
		if err != nil {
			return err
		}
	}
	result, err := s.reports.Run(ctx, req)
	if err != nil {
		return err
	}
	var body []byte
	var mimeType string
	if run.format == "xlsx" {
		body, err = reportXLSX(result)
		mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	} else {
		body, err = reportPDF(result)
		mimeType = "application/pdf"
	}
	if err != nil {
		return err
	}
	if len(body) > 50<<20 {
		return fmt.Errorf("отчёт превышает 50 МБ")
	}
	key := "reports/" + run.id.String() + "." + run.format
	if err = s.files.UploadFile(ctx, key, bytes.NewReader(body), int64(len(body)), mimeType); err != nil {
		return err
	}
	return s.completeRun(ctx, run, key, int64(len(body)), nil)
}

func (s *reportScheduleStore) cleanupExpired(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,COALESCE(storage_key,'') FROM report_runs
		WHERE created_at<now()-interval '90 days' AND status IN ('ready','failed') LIMIT 20`)
	if err != nil {
		return err
	}
	items := make([]struct {
		id  uuid.UUID
		key string
	}, 0)
	for rows.Next() {
		var item struct {
			id  uuid.UUID
			key string
		}
		if err := rows.Scan(&item.id, &item.key); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.key != "" {
			if err := s.files.DeleteFile(ctx, item.key); err != nil {
				return err
			}
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM report_runs WHERE id=$1`, item.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *reportScheduleStore) Run(ctx context.Context, ready func() error) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	var lastCleanup time.Time
	for {
		if ready() == nil {
			workCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
			if err := s.enqueueDue(workCtx); err != nil {
				slog.Warn("enqueue reports failed", "error", err)
			}
			for i := 0; i < 5; i++ {
				run, err := s.claimRun(workCtx)
				if err != nil {
					slog.Warn("claim report failed", "error", err)
					break
				}
				if run == nil {
					break
				}
				if err := s.generateRun(workCtx, run); err != nil {
					slog.Warn("scheduled report failed", "run", run.id, "error", err)
					if saveErr := s.completeRun(workCtx, run, "", 0, err); saveErr != nil {
						slog.Warn("record report failure failed", "error", saveErr)
					}
				}
			}
			if time.Since(lastCleanup) >= time.Hour {
				if err := s.cleanupExpired(workCtx); err != nil {
					slog.Warn("report retention cleanup failed", "error", err)
				} else {
					lastCleanup = time.Now()
				}
			}
			cancel()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func validReportID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, models.NewBadRequest("некорректный идентификатор отчёта")
	}
	return id, nil
}
