// Package tz is the edge conversion in ADR 0005: a timestamptz plus an IANA
// zone name become a local wall clock, and a wall clock becomes an instant.
//
// Go's time.Date does not error on DST gaps or overlaps. It picks some nearby
// instant and says nothing. Instant refuses those inputs instead of accepting
// a time the caller did not write.
package tz

import (
	"errors"
	"fmt"
	"slices"
	"time"

	_ "time/tzdata"
)

var (
	ErrUnknownZone          = errors.New("unknown IANA time zone")
	ErrNonexistentLocalTime = errors.New("nonexistent local time")
	ErrAmbiguousLocalTime   = errors.New("ambiguous local time")
)

func Location(name string) (*time.Location, error) {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownZone, name)
	}
	return loc, nil
}

// Instant is a unique timestamptz for a civil local time in zone.
// A spring-forward gap returns ErrNonexistentLocalTime.
// A fall-back overlap returns ErrAmbiguousLocalTime.
func Instant(zone string, year int, month time.Month, day, hour, minute, second int) (time.Time, error) {
	found, err := Occurrences(zone, year, month, day, hour, minute, second)
	if err != nil {
		return time.Time{}, err
	}
	switch len(found) {
	case 0:
		return time.Time{}, ErrNonexistentLocalTime
	case 1:
		return found[0], nil
	default:
		return time.Time{}, ErrAmbiguousLocalTime
	}
}

// Occurrences lists every instant that has this wall clock in zone.
// Zero, one, or two results: a DST gap, a unique local time, or a fall-back fold.
func Occurrences(zone string, year int, month time.Month, day, hour, minute, second int) ([]time.Time, error) {
	loc, err := Location(zone)
	if err != nil {
		return nil, err
	}

	guess := time.Date(year, month, day, hour, minute, second, 0, loc)
	var found []time.Time
	seen := map[int64]struct{}{}
	for _, cand := range []time.Time{guess, guess.Add(-time.Hour), guess.Add(time.Hour)} {
		local := cand.In(loc)
		if !sameWall(local, year, month, day, hour, minute, second) {
			continue
		}
		unix := local.Unix()
		if _, ok := seen[unix]; ok {
			continue
		}
		seen[unix] = struct{}{}
		found = append(found, local)
	}
	slices.SortFunc(found, func(a, b time.Time) int { return a.Compare(b) })
	return found, nil
}

// WallClock renders an instant in the event's IANA zone, not UTC and not the
// process local zone.
func WallClock(instant time.Time, zone string) (time.Time, error) {
	loc, err := Location(zone)
	if err != nil {
		return time.Time{}, err
	}
	return instant.In(loc), nil
}

func sameWall(t time.Time, year int, month time.Month, day, hour, minute, second int) bool {
	y, m, d := t.Date()
	h, min, sec := t.Clock()
	return y == year && m == month && d == day && h == hour && min == minute && sec == second
}
