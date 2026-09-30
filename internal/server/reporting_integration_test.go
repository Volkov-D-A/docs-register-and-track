package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestReportsPermissionAndScheduleUniquenessIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	admin, reporter, viewer := uuid.New(), uuid.New(), uuid.New()
	hash, err := security.HashPassword("ReportingPassw0rd!")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users(id,login,password_hash,last_name,first_name,no_patronymic,password_change_required) VALUES
		($1,'reports-admin',$4,'Reports','Admin',true,false),
		($2,'reports-user',$4,'Reports','User',true,false),
		($3,'reports-viewer',$4,'Reports','Viewer',true,false)`, admin, reporter, viewer, hash)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_system_permissions(user_id,permission,is_allowed) VALUES
		($1,'admin',true),($2,'reports',true),($3,'stats_documents',true)`, admin, reporter, viewer)
	require.NoError(t, err)
	api := newIntegrationManagementAPI(t, &App{db: db, cfg: validConfig()})
	login := func(name string) string {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"login":"`+name+`","password":"ReportingPassw0rd!"}`))
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var data struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&data))
		return data.AccessToken
	}
	request := func(token, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"template":"organizations","startDate":"2026-01-01","endDate":"2026-01-31"}`))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, r)
		return w
	}
	require.Equal(t, http.StatusOK, request(login("reports-user"), "/api/v1/reports/run").Code)
	require.Equal(t, http.StatusForbidden, request(login("reports-viewer"), "/api/v1/reports/run").Code)
	require.Equal(t, http.StatusForbidden, request(login("reports-viewer"), "/api/v1/reports/export/pdf").Code)

	store := &reportScheduleStore{db: db}
	item, err := store.Create(context.Background(), reporter, models.ReportScheduleRequest{
		Report: models.ReportRequest{Template: "organizations"}, Frequency: "weekly", Timezone: "Asia/Yekaterinburg", Format: "xlsx"})
	require.NoError(t, err)
	id, err := uuid.Parse(item.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE report_schedules SET next_run_at=now()-interval '1 minute' WHERE id=$1`, id)
	require.NoError(t, err)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errors <- store.enqueueDue(context.Background()) }()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM report_runs WHERE schedule_id=$1`, id).Scan(&count))
	require.Equal(t, 1, count)
	runs, err := store.Runs(context.Background(), reporter, id)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "pending", runs[0].Status)
	_, _, _, err = store.DownloadInfo(context.Background(), viewer, uuid.MustParse(runs[0].ID))
	require.Error(t, err)
}
