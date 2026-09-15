package cron

import (
	"fmt"
	"strconv"
	"strings"
)

type fieldSpec struct {
	name  string
	min   int
	max   int
	names map[string]int
}

var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var dayNames = map[string]int{
	"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
}

var (
	minuteSpec = fieldSpec{name: "minute", min: 0, max: 59}
	hourSpec   = fieldSpec{name: "hour", min: 0, max: 23}
	domSpec    = fieldSpec{name: "day of month", min: 1, max: 31}
	monthSpec  = fieldSpec{name: "month", min: 1, max: 12, names: monthNames}
	dowSpec    = fieldSpec{name: "day of week", min: 0, max: 6, names: dayNames}
)

func parseField(raw string, spec fieldSpec, lenient bool) (fieldSet, error) {
	set := fieldSet{}
	for _, part := range strings.Split(raw, ",") {
		if part == "" {
			return nil, fmt.Errorf("empty list item")
		}
		if err := parsePart(part, spec, lenient, set); err != nil {
			return nil, err
		}
	}
	return set, nil
}

func parsePart(part string, spec fieldSpec, lenient bool, set fieldSet) error {
	base, step := part, 1
	hasStep := false
	if idx := strings.IndexByte(part, '/'); idx >= 0 {
		base = part[:idx]
		n, err := strconv.Atoi(part[idx+1:])
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid step %q", part[idx+1:])
		}
		step = n
		hasStep = true
	}

	var start, end int
	switch {
	case base == "*":
		start, end = spec.min, spec.max

	case strings.Contains(base, "-"):
		bounds := strings.SplitN(base, "-", 2)
		a, err := resolveValue(bounds[0], spec, lenient)
		if err != nil {
			return err
		}
		b, err := resolveValue(bounds[1], spec, lenient)
		if err != nil {
			return err
		}
		if a > b {
			if !lenient {
				return fmt.Errorf("descending range %q not allowed without --lenient", base)
			}
			// Treat as wrapping past the field's max back to its min,
			// e.g. hours "22-2" means 22,23,0,1,2.
			addRange(set, a, spec.max, step)
			addRange(set, spec.min, b, step)
			return nil
		}
		start, end = a, b

	default:
		v, err := resolveValue(base, spec, lenient)
		if err != nil {
			return err
		}
		if !hasStep {
			set[v] = true
			return nil
		}
		start, end = v, spec.max
	}

	addRange(set, start, end, step)
	return nil
}

func addRange(set fieldSet, start, end, step int) {
	for v := start; v <= end; v += step {
		set[v] = true
	}
}

func resolveValue(token string, spec fieldSpec, lenient bool) (int, error) {
	token = strings.TrimSpace(token)
	if spec.names != nil {
		if v, ok := spec.names[strings.ToUpper(token)]; ok {
			if !lenient {
				return 0, fmt.Errorf("name %q not allowed without --lenient", token)
			}
			return v, nil
		}
	}

	v, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", token)
	}

	if spec.name == "day of week" && v == 7 {
		if !lenient {
			return 0, fmt.Errorf("day of week 7 (Sunday alias) not allowed without --lenient")
		}
		v = 0
	}

	if v < spec.min || v > spec.max {
		return 0, fmt.Errorf("value %d out of range [%d-%d]", v, spec.min, spec.max)
	}
	return v, nil
}
