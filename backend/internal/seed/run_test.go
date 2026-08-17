package seed

import "testing"

func TestRequireComposePostgres(t *testing.T) {
	ok := "postgres://emp_user:emp_password@postgres:5432/event_management_platform?sslmode=disable"
	if err := requireComposePostgres(ok); err != nil {
		t.Fatalf("compose url: %v", err)
	}
	if err := requireComposePostgres("postgres://emp_user:emp_password@localhost:5432/event_management_platform"); err == nil {
		t.Fatal("localhost should be refused")
	}
}
