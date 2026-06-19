package model

import (
	"fmt"
	"strings"
	"time"
)

// Date is a calendar date (no time, no zone). Stored and serialized as
// ISO-8601 "YYYY-MM-DD".
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// isoLayout is the canonical wire format for Date.
const isoLayout = "2006-01-02"

// acceptedLayouts are the formats ParseDate will try, in order. Covers the
// common shapes seen in payer CSV exports.
var acceptedLayouts = []string{
	isoLayout,
	"01/02/2006",
	"1/2/2006",
	"01-02-2006",
	"2006/01/02",
	"Jan 2, 2006",
	"January 2, 2006",
	"02 Jan 2006",
}

// ParseDate parses a date string using the accepted layouts.
func ParseDate(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Date{}, fmt.Errorf("empty date")
	}
	for _, layout := range acceptedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}, nil
		}
	}
	return Date{}, fmt.Errorf("unrecognized date %q", s)
}

// MustParseDate is ParseDate that panics on error; for tests and constants.
func MustParseDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d Date) time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// String renders the canonical "YYYY-MM-DD" form.
func (d Date) String() string {
	return d.time().Format(isoLayout)
}

// IsZero reports whether the date is the zero value.
func (d Date) IsZero() bool {
	return d.Year == 0 && d.Month == 0 && d.Day == 0
}

// DaysBetween returns the absolute number of whole days between two dates.
func DaysBetween(a, b Date) int {
	diff := a.time().Sub(b.time()).Hours() / 24
	if diff < 0 {
		diff = -diff
	}
	return int(diff + 0.5)
}

// WithinDays reports whether a and b are at most n days apart (inclusive).
func WithinDays(a, b Date, n int) bool {
	return DaysBetween(a, b) <= n
}
