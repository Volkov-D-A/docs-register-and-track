package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrationCompatibilityErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *MigrationCompatibilityError
		want string
	}{
		{
			name: "schema too new",
			err:  &MigrationCompatibilityError{CurrentVersion: 9, LatestAvailableVersion: 8, SchemaTooNew: true},
			want: "database schema version 9 is newer than embedded migrations 8",
		},
		{
			name: "dirty schema",
			err:  &MigrationCompatibilityError{CurrentVersion: 7, Dirty: true},
			want: "database schema version 7 is dirty",
		},
		{
			name: "generic incompatible schema",
			err:  &MigrationCompatibilityError{},
			want: "database schema is incompatible with this binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}
