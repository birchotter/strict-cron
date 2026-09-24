package cron

import (
	"reflect"
	"strings"
	"testing"
)

func ints(vals ...int) fieldSet {
	s := fieldSet{}
	for _, v := range vals {
		s[v] = true
	}
	return s
}

func fullRange(min, max int) fieldSet {
	s := fieldSet{}
	for v := min; v <= max; v++ {
		s[v] = true
	}
	return s
}

type fieldCase struct {
	input        string
	lenient      bool
	want         fieldSet
	wantErr      bool
	wantErrMatch string // substring the error must contain, when set
}

func runFieldCases(t *testing.T, spec fieldSpec, cases []fieldCase) {
	t.Helper()
	for _, tc := range cases {
		got, err := parseField(tc.input, spec, tc.lenient)
		mode := "strict"
		if tc.lenient {
			mode = "lenient"
		}
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s %q (%s): expected error, got %v", spec.name, tc.input, mode, got)
				continue
			}
			if tc.wantErrMatch != "" && !strings.Contains(err.Error(), tc.wantErrMatch) {
				t.Errorf("%s %q (%s): error %q does not contain %q", spec.name, tc.input, mode, err, tc.wantErrMatch)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s %q (%s): unexpected error: %v", spec.name, tc.input, mode, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s %q (%s): got %v, want %v", spec.name, tc.input, mode, got, tc.want)
		}
	}
}

// TestParseField_Common covers behavior that does not depend on which field
// is being parsed: wildcards, steps, lists, and malformed input. minuteSpec
// stands in for any field with no name aliases.
func TestParseField_Common(t *testing.T) {
	cases := []fieldCase{
		{input: "*", lenient: false, want: fullRange(0, 59)},
		{input: "*", lenient: true, want: fullRange(0, 59)},
		{input: "0-5", lenient: false, want: ints(0, 1, 2, 3, 4, 5)},
		{input: "*/15", lenient: false, want: ints(0, 15, 30, 45)},
		{input: "10/20", lenient: false, want: ints(10, 30, 50)},
		{input: "0,30,45", lenient: false, want: ints(0, 30, 45)},
		{input: "59", lenient: false, want: ints(59)},
		{input: "60", lenient: false, wantErr: true},
		{input: "60", lenient: true, wantErr: true},
		{input: "abc", lenient: false, wantErr: true},
		{input: "*/0", lenient: false, wantErr: true, wantErrMatch: "invalid step"},
		{input: "*/0", lenient: true, wantErr: true, wantErrMatch: "invalid step"},
		{input: "*/-1", lenient: true, wantErr: true, wantErrMatch: "invalid step"},
		{input: "*/x", lenient: false, wantErr: true, wantErrMatch: "invalid step"},
		{input: "1,,2", lenient: false, wantErr: true, wantErrMatch: "empty list item"},
		{input: "50-10", lenient: false, wantErr: true, wantErrMatch: "descending range"},
		{
			input:   "50-10",
			lenient: true,
			want:    ints(50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
		},
	}
	runFieldCases(t, minuteSpec, cases)
}

func TestParseField_Hour(t *testing.T) {
	cases := []fieldCase{
		{input: "*", lenient: false, want: fullRange(0, 23)},
		{input: "9-17/2", lenient: false, want: ints(9, 11, 13, 15, 17)},
		{input: "23", lenient: false, want: ints(23)},
		{input: "24", lenient: false, wantErr: true},
		{input: "24", lenient: true, wantErr: true},
	}
	runFieldCases(t, hourSpec, cases)
}

func TestParseField_DayOfMonth(t *testing.T) {
	cases := []fieldCase{
		{input: "1-31", lenient: false, want: fullRange(1, 31)},
		{input: "0", lenient: false, wantErr: true},
		{input: "32", lenient: false, wantErr: true},
		{input: "29", lenient: false, want: ints(29)},
	}
	runFieldCases(t, domSpec, cases)
}

func TestParseField_Month(t *testing.T) {
	cases := []fieldCase{
		{input: "1", lenient: false, want: ints(1)},
		{input: "0", lenient: false, wantErr: true},
		{input: "13", lenient: false, wantErr: true},
		{input: "*/3", lenient: false, want: ints(1, 4, 7, 10)},
		{input: "JAN", lenient: false, wantErr: true, wantErrMatch: "not allowed without --lenient"},
		{input: "JAN", lenient: true, want: ints(1)},
		{input: "jan", lenient: true, want: ints(1)},
		{input: "JAN-MAR", lenient: false, wantErr: true},
		{input: "JAN-MAR", lenient: true, want: ints(1, 2, 3)},
		{input: "JAN,JUN,DEC", lenient: false, wantErr: true},
		{input: "JAN,JUN,DEC", lenient: true, want: ints(1, 6, 12)},
	}
	runFieldCases(t, monthSpec, cases)
}

func TestParseField_DayOfWeek(t *testing.T) {
	cases := []fieldCase{
		{input: "0", lenient: false, want: ints(0)},
		{input: "6", lenient: false, want: ints(6)},
		{input: "7", lenient: false, wantErr: true, wantErrMatch: "not allowed without --lenient"},
		{input: "7", lenient: true, want: ints(0)},
		{input: "8", lenient: false, wantErr: true},
		{input: "8", lenient: true, wantErr: true},
		{input: "MON", lenient: false, wantErr: true},
		{input: "MON", lenient: true, want: ints(1)},
		{input: "MON-FRI", lenient: false, wantErr: true},
		{input: "MON-FRI", lenient: true, want: ints(1, 2, 3, 4, 5)},
		{input: "5-1", lenient: false, wantErr: true, wantErrMatch: "descending range"},
		{input: "5-1", lenient: true, want: ints(5, 6, 0, 1)},
		{input: "1-5/2", lenient: false, want: ints(1, 3, 5)},
	}
	runFieldCases(t, dowSpec, cases)
}
