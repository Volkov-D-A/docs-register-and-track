package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationStatusJSONContract(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		status        MigrationStatus
	}{
		{"ready", `{"currentVersion":8,"dirty":false,"availableCount":7,"latestAvailableVersion":8,"upToDate":true,"schemaTooNew":false,"compatible":true}`, MigrationStatus{CurrentVersion: 8, AvailableCount: 7, LatestAvailableVersion: 8, UpToDate: true, Compatible: true}},
		{"empty", `{"currentVersion":0,"dirty":false,"availableCount":0,"latestAvailableVersion":0,"upToDate":false,"schemaTooNew":false,"compatible":false}`, MigrationStatus{}},
		{"incompatible", `{"currentVersion":9,"dirty":true,"availableCount":7,"latestAvailableVersion":8,"upToDate":false,"schemaTooNew":true,"compatible":false}`, MigrationStatus{CurrentVersion: 9, Dirty: true, AvailableCount: 7, LatestAvailableVersion: 8, SchemaTooNew: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.status)
			require.NoError(t, err)
			require.JSONEq(t, tc.payload, string(encoded))
			var decoded MigrationStatus
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &decoded))
			require.Equal(t, tc.status, decoded)
		})
	}
}
