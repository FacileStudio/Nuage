package database

import "testing"

func TestWithStatementTimeout(t *testing.T) {
	got, err := withStatementTimeout("postgres://u:p@h:5432/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	want := "postgres://u:p@h:5432/db?options=-c+statement_timeout%3D30000&sslmode=disable"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
