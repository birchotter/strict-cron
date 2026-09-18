// Command croncheck validates a cron expression and optionally tests it
// against a specific time.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"strictcron"
)

func main() {
	lenient := flag.Bool("lenient", false, "allow non-standard syntax: nicknames, month/weekday names, the 7=Sunday alias, and wrapping ranges")
	at := flag.String("at", "", "check whether the schedule matches this RFC3339 time, and use it as the base for --next")
	next := flag.Int("next", 0, "print this many upcoming run times after --at (or now, if --at is omitted)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: croncheck [--lenient] [--at RFC3339-time] [--next N] \"expression\"\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	expr := flag.Arg(0)
	sched, err := cron.Parse(expr, cron.Options{Lenient: *lenient})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("valid: %s\n", expr)

	base := time.Now()
	if *at != "" {
		t, err := time.Parse(time.RFC3339, *at)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --at time: %v\n", err)
			os.Exit(1)
		}
		base = t
		if sched.Matches(t) {
			fmt.Printf("matches %s\n", t.Format(time.RFC3339))
		} else {
			fmt.Printf("does not match %s\n", t.Format(time.RFC3339))
		}
	}

	if *next <= 0 {
		return
	}
	for _, t := range sched.NextN(base, *next) {
		fmt.Printf("next: %s\n", t.Format(time.RFC3339))
	}
}
