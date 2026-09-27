package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestDepartmentListLoadsNomenclatureIDsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	repo := NewDepartmentRepository(database.Wrap(sqlDB))
	firstDepartmentID, secondDepartmentID := uuid.New(), uuid.New()
	firstNomenclatureID, secondNomenclatureID := uuid.New(), uuid.New()

	_, err := sqlDB.Exec(`INSERT INTO departments (id, name) VALUES ($1, 'А Отдел'), ($2, 'Б Отдел')`, firstDepartmentID, secondDepartmentID)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`INSERT INTO nomenclature (id, name, index, year, kind_code) VALUES
		($1, 'Второе дело', '02', 2026, 'incoming_letter'),
		($2, 'Первое дело', '01', 2026, 'incoming_letter')`, secondNomenclatureID, firstNomenclatureID)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`INSERT INTO department_nomenclature (department_id, nomenclature_id) VALUES ($1, $2), ($1, $3)`, firstDepartmentID, secondNomenclatureID, firstNomenclatureID)
	require.NoError(t, err)

	departments, err := repo.GetAll()
	require.NoError(t, err)
	require.Len(t, departments, 2)
	require.Equal(t, firstDepartmentID, departments[0].ID)
	require.Equal(t, []string{firstNomenclatureID.String(), secondNomenclatureID.String()}, departments[0].NomenclatureIDs)
	require.Equal(t, secondDepartmentID, departments[1].ID)
	require.Empty(t, departments[1].NomenclatureIDs)
}
