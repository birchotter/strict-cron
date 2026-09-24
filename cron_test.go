package cron

import (
	"strings"
	"testing"
	"time"
)

func mustParse(t *testing.T, expr string, opts Options) *Schedule {
	t.Helper()
	s, err := Parse(expr, opts)
	if err != nil {
		t.Fatalf("Parse(%q, %+v): unexpected error: %v", expr, opts, err)
	}
	return s
}

func TestParse_Empty(t *testing.T) {
	for _, expr := range []string{"", "   "} {
		if _, err := Parse(expr, Options{}); err == nil {
			t.Errorf("Parse(%q, ...): expected error for empty expression", expr)
		}
	}
}

func TestParse_FieldCount(t *testing.T) {
	for _, expr := range []string{"0 0 * *", "0 0 * * * *"} {
		if _, err := Parse(expr, Options{}); err == nil {
			t.Errorf("Parse(%q, ...): expected field count error", expr)
		}
	}
}

func TestParse_Nicknames(t *testing.T) {
	cases := []struct {
		expr    string
		lenient bool
		wantErr bool
		match   string
	}{
		{expr: "@daily", lenient: false, wantErr: true, match: "requires --lenient"},
		{expr: "@daily", lenient: true, wantErr: false},
		{expr: "@bogus", lenient: true, wantErr: true, match: "unknown nickname"},
		{expr: "@bogus", lenient: false, wantErr: true, match: "unknown nickname"},
	}
	for _, tc := range cases {
		_, err := Parse(tc.expr, Options{Lenient: tc.lenient})
		if tc.wantErr {
			if err == nil {
				t.Errorf("Parse(%q, Lenient:%v): expected error", tc.expr, tc.lenient)
				continue
			}
			if tc.match != "" && !strings.Contains(err.Error(), tc.match) {
				t.Errorf("Parse(%q, Lenient:%v): error %q does not contain %q", tc.expr, tc.lenient, err, tc.match)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q, Lenient:%v): unexpected error: %v", tc.expr, tc.lenient, err)
		}
	}
}

func TestParse_StrictRejectsNamesAcrossFields(t *testing.T) {
	const expr = "0 0 1 JAN MON"
	if _, err := Parse(expr, Options{}); err == nil {
		t.Errorf("Parse(%q, strict): expected error", expr)
	}
	if _, err := Parse(expr, Options{Lenient: true}); err != nil {
		t.Errorf("Parse(%q, lenient): unexpected error: %v", expr, err)
	}
}

func TestSchedule_String(t *testing.T) {
	const expr = "0 9 * * 1-5"
	s := mustParse(t, expr, Options{})
	if got := s.String(); got != expr {
		t.Errorf("String() = %q, want %q", got, expr)
	}
}

func TestSchedule_Matches(t *testing.T) {
	s := mustParse(t, "30 9 * * 1-5", Options{})
	monday := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC) // a Monday
	saturday := time.Date(2024, 1, 6, 9, 30, 0, 0, time.UTC)

	if !s.Matches(monday) {
		t.Errorf("Matches(%v) = false, want true", monday)
	}
	if s.Matches(saturday) {
		t.Errorf("Matches(%v) = true, want false", saturday)
	}
	if s.Matches(monday.Add(time.Minute)) {
		t.Errorf("Matches(%v) = true, want false", monday.Add(time.Minute))
	}
}

func TestSchedule_Next(t *testing.T) {
	s := mustParse(t, "0 12 * * *", Options{})
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	next, ok := s.Next(after)
	if !ok {
		t.Fatal("Next: expected a match")
	}
	want := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("Next(%v) = %v, want %v", after, next, want)
	}

	// Next is exclusive of the given time.
	next2, ok := s.Next(want)
	if !ok {
		t.Fatal("Next: expected a match")
	}
	want2 := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	if !next2.Equal(want2) {
		t.Errorf("Next(%v) = %v, want %v", want, next2, want2)
	}
}

func TestSchedule_NextN(t *testing.T) {
	s := mustParse(t, "0 12 * * *", Options{})
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	got := s.NextN(after, 3)
	if len(got) != 3 {
		t.Fatalf("NextN: got %d results, want 3", len(got))
	}
	for i, want := range []time.Time{
		time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC),
	} {
		if !got[i].Equal(want) {
			t.Errorf("NextN[%d] = %v, want %v", i, got[i], want)
		}
	}
}

func TestSchedule_NextN_ZeroOrNegative(t *testing.T) {
	s := mustParse(t, "0 12 * * *", Options{})
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, n := range []int{0, -1} {
		if got := s.NextN(after, n); got != nil {
			t.Errorf("NextN(after, %d) = %v, want nil", n, got)
		}
	}
}

func TestSchedule_Next_NeverFires(t *testing.T) {
	// April never has 31 days, and the month field pins the search to April,
	// so this schedule can never match.
	s := mustParse(t, "0 0 31 4 *", Options{})
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, ok := s.Next(after); ok {
		t.Error("Next: expected no match for day 31 in April")
	}
}
