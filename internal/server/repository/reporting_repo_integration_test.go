package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestReportingAggregatesIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	ctx := context.Background()
	department, user, nomenclature, firstOrg, secondOrg := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	incoming, outgoing := uuid.New(), uuid.New()
	_, err := db.Exec(`INSERT INTO departments(id,name) VALUES($1,'Reporting')`, department)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users(id,login,password_hash,last_name,first_name,no_patronymic,department_id) VALUES($1,'report-user','hash','Report','User',true,$2)`, user, department)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO nomenclature(id,name,index,year,kind_code) VALUES($1,'Letters','R',2026,'incoming_letter')`, nomenclature)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations(id,name) VALUES($1,'First'),($2,'Second')`, firstOrg, secondOrg)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO documents(id,kind,nomenclature_id,registration_number,registration_date,document_type,content,created_by) VALUES
		($1,'incoming_letter',$3,'R-1','2026-02-10','Письмо','Incoming',$4),
		($2,'outgoing_letter',$3,'R-2','2026-02-11','Письмо','Outgoing',$4)`, incoming, outgoing, nomenclature, user)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO document_correspondent_registrations(document_id,registration_number,registration_date,correspondent_org_id,position) VALUES
		($1,'C-1','2026-02-10',$2,1),($1,'C-2','2026-02-10',$3,2),($1,'C-3','2026-02-10',$2,3)`, incoming, firstOrg, secondOrg)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO outgoing_document_details(document_id,outgoing_number,outgoing_date,sender_signatory,sender_executor,recipient_org_id,addressee) VALUES
		($1,'O-1','2026-02-11','Signer','Executor',$2,'Addressee')`, outgoing, firstOrg)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO assignments(document_id,executor_id,creator_id,content,type,status,deadline,created_at,completed_at) VALUES
		($1,$2,$2,'First','execution','finished','2026-02-12','2026-02-10 08:00:00+00','2026-02-12 08:00:00+00'),
		($1,$2,$2,'Second','execution','completed','2026-02-12','2026-02-10 08:00:00+00','2026-02-13 08:00:00+00'),
		($1,NULL,$2,'Acknowledgment','acknowledgment','finished',NULL,'2026-02-10 08:00:00+00',NULL)`, incoming, user)
	require.NoError(t, err)
	repo := NewReportingRepository(db)
	start, end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	orgs, err := repo.Run(ctx, models.ReportRequest{Template: "organizations"}, start, end)
	require.NoError(t, err)
	require.Len(t, orgs, 2)
	require.Equal(t, 1, orgs[0].Incoming)
	require.Equal(t, 1, orgs[0].Outgoing)
	require.Equal(t, 1, orgs[1].Incoming)
	_, err = db.Exec(`INSERT INTO documents(id,kind,nomenclature_id,registration_number,registration_date,document_type,content,created_by)
		VALUES($1,'incoming_letter',$2,'R-3','2026-02-12','Письмо','Without organization',$3)`, uuid.New(), nomenclature, user)
	require.NoError(t, err)
	unique, err := repo.UniqueDocumentCount(ctx, start, end, models.ReportRequest{Template: "organizations"})
	require.NoError(t, err)
	require.Equal(t, 2, unique)
	incomingOnly := models.ReportRequest{Template: "organizations", KindCode: "incoming_letter", OrganizationID: firstOrg.String(), UserID: user.String()}
	filteredOrgs, err := repo.Run(ctx, incomingOnly, start, end)
	require.NoError(t, err)
	require.Len(t, filteredOrgs, 1)
	require.Equal(t, 1, filteredOrgs[0].Incoming)
	require.Zero(t, filteredOrgs[0].Outgoing)
	filteredUnique, err := repo.UniqueDocumentCount(ctx, start, end, incomingOnly)
	require.NoError(t, err)
	require.Equal(t, 1, filteredUnique)
	load, err := repo.Run(ctx, models.ReportRequest{Template: "department_load"}, start, end)
	require.NoError(t, err)
	require.Len(t, load, 1)
	require.Equal(t, 2, load[0].Count)
	require.Equal(t, "2026-02", load[0].Period)
	filteredLoad, err := repo.Run(ctx, models.ReportRequest{Template: "department_load", KindCode: "incoming_letter", NomenclatureID: nomenclature.String(), UserID: user.String(), Status: "finished", OnlyWithDeadline: true, DepartmentID: department.String()}, start, end)
	require.NoError(t, err)
	require.Len(t, filteredLoad, 1)
	require.Equal(t, 1, filteredLoad[0].Count)
	average, err := repo.Run(ctx, models.ReportRequest{Template: "execution_time"}, start, end)
	require.NoError(t, err)
	require.InDelta(t, 2.5, average[0].Metric, 0.01)
	overdue, err := repo.Run(ctx, models.ReportRequest{Template: "overdue_rate"}, start, end)
	require.NoError(t, err)
	require.Equal(t, 2, overdue[0].Count)
	require.Equal(t, 1, overdue[0].Overdue)
	require.InDelta(t, 50, overdue[0].Metric, 0.01)
	_, err = db.Exec(`INSERT INTO assignments(document_id,executor_id,creator_id,content,type,status,created_at,completed_at)
		VALUES($1,$2,$2,'Time zone boundary','execution','completed','2026-02-27 22:00:00+00','2026-02-28 22:00:00+00')`, incoming, user)
	require.NoError(t, err)
	march := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	localRows, err := repo.Run(ctx, models.ReportRequest{Template: "execution_time", Timezone: "Asia/Yekaterinburg"}, march, march)
	require.NoError(t, err)
	require.Len(t, localRows, 1)
	require.Equal(t, 1, localRows[0].Count)
	utcRows, err := repo.Run(ctx, models.ReportRequest{Template: "execution_time", Timezone: "UTC"}, march, march)
	require.NoError(t, err)
	require.Empty(t, utcRows)
	_, err = db.Exec(`INSERT INTO assignments(document_id,executor_id,creator_id,content,type,status,deadline,created_at)
		VALUES($1,$2,$2,'Open overdue','execution','new','2026-02-01','2026-01-20 08:00:00+00')`, incoming, user)
	require.NoError(t, err)
	open, err := repo.OpenOverdueCount(ctx, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), models.ReportRequest{DepartmentID: department.String()})
	require.NoError(t, err)
	require.Equal(t, 1, open)
}
