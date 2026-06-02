package main

import "testing"

func TestWithDatabaseName(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "url dsn",
			dsn:  "postgres://user:pass@localhost:5432/postgres?sslmode=disable",
			want: "postgres://user:pass@localhost:5432/docflow_test_1?sslmode=disable",
		},
		{
			name: "key value dsn replaces dbname",
			dsn:  "host=localhost port=5432 user=user password=pass dbname=postgres sslmode=disable",
			want: "host=localhost port=5432 user=user password=pass dbname=docflow_test_1 sslmode=disable",
		},
		{
			name: "key value dsn appends dbname",
			dsn:  "host=localhost port=5432 user=user password=pass sslmode=disable",
			want: "host=localhost port=5432 user=user password=pass sslmode=disable dbname=docflow_test_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := withDatabaseName(tt.dsn, "docflow_test_1")
			if err != nil {
				t.Fatalf("withDatabaseName() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("withDatabaseName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithDatabaseNameRejectsUnsafeName(t *testing.T) {
	if _, err := withDatabaseName("host=localhost", "production"); err == nil {
		t.Fatal("expected unsafe database name to be rejected")
	}
}
