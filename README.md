# strict-cron

A cron expression parser that refuses to guess.

Cron syntax was never standardized. `cron(8)`, Vixie cron, and every job
scheduler that copied the 5-field format since disagree on details: does
`@daily` exist, does `7` mean Sunday or is that an error, does `23-1` wrap
around midnight or fail to parse, are `JAN`/`MON` valid in the month and
weekday fields. Most parsers just pick an interpretation and apply it
silently. That's fine until you paste a schedule meant for one system into
another and it means something different than you think.

This library parses one thing strictly by default: plain 5-field numeric
cron expressions (`minute hour day-of-month month day-of-week`), all
integers, no aliases. Everything else - nicknames, name aliases, the
`7 = Sunday` alias, wrapping ranges - is rejected unless you explicitly ask
for it.

## Library

```go
package main

import (
	"fmt"

	"strictcron"
)

func main() {
	// Strict: fails, because @daily is not a plain 5-field expression.
	if _, err := cron.Parse("@daily", cron.Options{}); err != nil {
		fmt.Println(err) // cron: nickname "@daily" requires --lenient (expands to "0 0 * * *")
	}

	// Strict: succeeds.
	sched, err := cron.Parse("0 9 * * 1-5", cron.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(sched.Matches(someTime))

	// Lenient: accepts the nickname, weekday names, and the 7=Sunday alias.
	sched, err = cron.Parse("@weekly", cron.Options{Lenient: true})
	if err != nil {
		panic(err)
	}
}
```

## CLI

```
$ go run ./cmd/croncheck "0 9 * * 1-5"
valid: 0 9 * * 1-5

$ go run ./cmd/croncheck "@daily"
cron: nickname "@daily" requires --lenient (expands to "0 0 * * *")

$ go run ./cmd/croncheck --lenient "@daily"
valid: @daily

$ go run ./cmd/croncheck --at 2026-09-16T09:00:00Z "0 9 * * 1-5"
valid: 0 9 * * 1-5
matches 2026-09-16T09:00:00Z
```

## What strict mode rejects

| Syntax | Example | Strict | `--lenient` |
| --- | --- | --- | --- |
| Nicknames | `@daily`, `@hourly` | rejected | expanded |
| Month/weekday names | `JAN`, `MON` | rejected | accepted |
| Day-of-week `7` alias for Sunday | `0 0 * * 7` | rejected | treated as `0` |
| Descending ranges | `22-2` | rejected | wraps around |

Some things are always rejected, in both modes, because they aren't a
matter of taste: a step of zero (`*/0`), a field count other than five, or
a value out of range for its field.

## Status

Early skeleton: expression parsing, validation, and matching a single
`time.Time` against a parsed schedule. No computation of the next N run
times yet - see the roadmap in the issue tracker.
