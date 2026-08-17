package tz

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestMain(m *testing.M) {
	// Machine zone must not be the event zone. Auckland is UTC+12 in June, so a
	// 09:00 Asia/Colombo instant is 15:30 here — different from 09:00 and from UTC.
	loc, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		panic(err)
	}
	time.Local = loc
	os.Exit(m.Run())
}

func TestDSTSpringForwardNonexistentLocalTime(t *testing.T) {
	const zone = "America/New_York" // seeded event "DST Spring Forward"

	_, err := Instant(zone, 2026, time.March, 8, 2, 30, 0)
	if !errors.Is(err, ErrNonexistentLocalTime) {
		t.Fatalf("02:30 on the spring-forward day must be refused, got %v", err)
	}

	ny, err := Location(zone)
	if err != nil {
		t.Fatal(err)
	}
	// Go's time.Date does not error. On Go 1.26 it yields 01:30 EST, not 02:30.
	// Instant must not return that invented instant.
	silent := time.Date(2026, time.March, 8, 2, 30, 0, 0, ny)
	if silent.Hour() == 2 && silent.Minute() == 30 {
		t.Fatal("time.Date accepted 02:30; the reject path would not be testing a gap")
	}
	if silent.Format(time.RFC3339) == "2026-03-08T02:30:00-05:00" ||
		silent.Format(time.RFC3339) == "2026-03-08T02:30:00-04:00" {
		t.Fatalf("time.Date produced a 02:30 offset, gap detection is untested: %s", silent.Format(time.RFC3339))
	}

	// 09:00 that day exists (after the jump, EDT). Same civil time the seeder uses.
	got, err := Instant(zone, 2026, time.March, 8, 9, 0, 0)
	if err != nil {
		t.Fatalf("09:00 on the spring-forward day must exist, got %v", err)
	}
	if got.Format(time.RFC3339) != "2026-03-08T09:00:00-04:00" {
		t.Fatalf("09:00 EDT, got %s", got.Format(time.RFC3339))
	}
}

func TestDSTFallBackAmbiguousLocalTime(t *testing.T) {
	const zone = "America/New_York" // seeded event "DST Fall Back"

	_, err := Instant(zone, 2026, time.November, 1, 1, 30, 0)
	if !errors.Is(err, ErrAmbiguousLocalTime) {
		t.Fatalf("01:30 on the fall-back day must be refused, got %v", err)
	}

	got, err := Occurrences(zone, 2026, time.November, 1, 1, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want two instants, got %d", len(got))
	}

	first := got[0].Format(time.RFC3339)
	second := got[1].Format(time.RFC3339)
	if first != "2026-11-01T01:30:00-04:00" {
		t.Fatalf("first occurrence (EDT) got %s", first)
	}
	if second != "2026-11-01T01:30:00-05:00" {
		t.Fatalf("second occurrence (EST) got %s", second)
	}
	if got[0].Format("15:04") != "01:30" || got[1].Format("15:04") != "01:30" {
		t.Fatal("both must show 01:30 on the wall")
	}
	if got[0].Equal(got[1]) || got[0].UTC().Equal(got[1].UTC()) {
		t.Fatal("the two 01:30s must be different instants")
	}
	if !got[0].IsDST() || got[1].IsDST() {
		t.Fatalf("first must be DST, second must not: %v %v", got[0].IsDST(), got[1].IsDST())
	}
	if got[1].Sub(got[0]) != time.Hour {
		t.Fatalf("occurrences should be one real hour apart, got %s", got[1].Sub(got[0]))
	}

	// After the fold, 03:30 exists once.
	unique, err := Instant(zone, 2026, time.November, 1, 3, 30, 0)
	if err != nil {
		t.Fatalf("03:30 on the fall-back day must exist, got %v", err)
	}
	if unique.Format(time.RFC3339) != "2026-11-01T03:30:00-05:00" {
		t.Fatalf("03:30 EST, got %s", unique.Format(time.RFC3339))
	}
}

func TestCrossZoneLocalRendering(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var row struct {
		Name     string    `db:"name"`
		TimeZone string    `db:"time_zone"`
		StartsAt time.Time `db:"starts_at"`
	}
	err = db.Get(&row, `
		SELECT name, time_zone, starts_at
		FROM   events
		WHERE  name = $1
	`, "Conference 10")
	if err != nil {
		t.Fatalf("seeded Asia/Colombo event: %v", err)
	}
	if row.TimeZone != "Asia/Colombo" {
		t.Fatalf("Conference 10 zone got %q", row.TimeZone)
	}

	local, err := WallClock(row.StartsAt, row.TimeZone)
	if err != nil {
		t.Fatal(err)
	}
	if local.Format(time.RFC3339) != "2026-06-08T09:00:00+05:30" {
		t.Fatalf("event-local wall clock got %s", local.Format(time.RFC3339))
	}
	if local.Location().String() != "Asia/Colombo" {
		t.Fatalf("location got %s", local.Location())
	}

	utc := row.StartsAt.UTC()
	if utc.Format(time.RFC3339) != "2026-06-08T03:30:00Z" {
		t.Fatalf("stored instant as UTC got %s", utc.Format(time.RFC3339))
	}
	if local.Hour() == utc.Hour() && local.Minute() == utc.Minute() {
		t.Fatal("local rendering collapsed to UTC")
	}

	machine := row.StartsAt.In(time.Local)
	if time.Local.String() != "Pacific/Auckland" {
		t.Fatalf("test machine zone must be Auckland, got %s", time.Local)
	}
	if local.Format("15:04") == machine.Format("15:04") {
		t.Fatalf("local rendering used the machine zone (%s)", machine.Format(time.RFC3339))
	}

	want, err := Instant(row.TimeZone, 2026, time.June, 8, 9, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !local.Equal(want) {
		t.Fatalf("round-trip: stored instant %s vs Instant(09:00) %s",
			local.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
