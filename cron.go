// Package cron parses cron schedule expressions.
//
// Cron syntax has never been standardized: cron(8), Vixie cron, and every
// job scheduler that copies the format disagree on nicknames (@daily),
// weekday name spelling, whether 7 means Sunday, and whether "23-1" wraps
// around midnight or is just an error. Parse is strict by default and
// rejects all of that; pass Options{Lenient: true} to accept it.
package cron

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Options controls how permissive Parse is about non-standard syntax.
type Options struct {
	// Lenient allows nicknames (@daily), month/weekday names (JAN, MON),
	// the day-of-week alias 7 for Sunday, and descending ranges that wrap
	// around (e.g. 22-2). Without it, Parse only accepts plain 5-field
	// numeric cron expressions.
	Lenient bool
}

// Schedule is a parsed cron expression.
type Schedule struct {
	minute fieldSet
	hour   fieldSet
	dom    fieldSet
	month  fieldSet
	dow    fieldSet
	raw    string
}

type fieldSet map[int]bool

// nicknames are only expanded when Options.Lenient is set, since they are
// a convention popularized by Vixie cron rather than part of the original
// 5-field format.
var nicknames = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// Parse parses a 5-field cron expression (minute hour day-of-month month
// day-of-week). By default it rejects nicknames, name aliases, the 7 =
// Sunday alias, and descending ranges; set opts.Lenient to accept them.
func Parse(expr string, opts Options) (*Schedule, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return nil, errors.New("cron: empty expression")
	}

	if strings.HasPrefix(trimmed, "@") {
		expanded, ok := nicknames[strings.ToLower(trimmed)]
		if !ok {
			return nil, fmt.Errorf("cron: unknown nickname %q", trimmed)
		}
		if !opts.Lenient {
			return nil, fmt.Errorf("cron: nickname %q requires --lenient (expands to %q)", trimmed, expanded)
		}
		trimmed = expanded
	}

	fields := strings.Fields(trimmed)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields (minute hour dom month dow), got %d", len(fields))
	}

	specs := [5]fieldSpec{minuteSpec, hourSpec, domSpec, monthSpec, dowSpec}
	var sets [5]fieldSet
	for i, raw := range fields {
		set, err := parseField(raw, specs[i], opts.Lenient)
		if err != nil {
			return nil, fmt.Errorf("cron: %s field %q: %w", specs[i].name, raw, err)
		}
		sets[i] = set
	}

	return &Schedule{
		minute: sets[0],
		hour:   sets[1],
		dom:    sets[2],
		month:  sets[3],
		dow:    sets[4],
		raw:    expr,
	}, nil
}

// Matches reports whether t falls on this schedule, to the minute.
func (s *Schedule) Matches(t time.Time) bool {
	return s.minute[t.Minute()] &&
		s.hour[t.Hour()] &&
		s.dom[t.Day()] &&
		s.month[int(t.Month())] &&
		s.dow[int(t.Weekday())]
}

// String returns the original expression as passed to Parse.
func (s *Schedule) String() string {
	return s.raw
}

// maxSearchSpan bounds how far into the future Next will look before giving
// up. Some valid-looking expressions never fire (e.g. day-of-month 31 paired
// with month February), and without a bound the search would run forever.
const maxSearchSpan = 8 * 365 * 24 * time.Hour

// Next returns the first time strictly after "after" that the schedule
// matches, to the minute. It reports false if no match is found within the
// search bound, which means the expression can never fire (e.g. day 31 in a
// month that never has one).
func (s *Schedule) Next(after time.Time) (time.Time, bool) {
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := after.Add(maxSearchSpan)
	for t.Before(limit) {
		if s.Matches(t) {
			return t, true
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, false
}

// NextN returns up to n times strictly after "after" that the schedule
// matches, in order. It returns fewer than n if the expression can never
// fire again within the search bound.
func (s *Schedule) NextN(after time.Time, n int) []time.Time {
	if n <= 0 {
		return nil
	}
	out := make([]time.Time, 0, n)
	t := after
	for len(out) < n {
		next, ok := s.Next(t)
		if !ok {
			break
		}
		out = append(out, next)
		t = next
	}
	return out
}
