package model

import "testing"

func TestParseDate(t *testing.T) {
	tests := []struct {
		in   string
		want string // canonical form, or "" if error expected
	}{
		{"2026-06-17", "2026-06-17"},
		{"06/17/2026", "2026-06-17"},
		{"6/7/2026", "2026-06-07"},
		{"06-17-2026", "2026-06-17"},
		{"2026/06/17", "2026-06-17"},
		{"Jun 17, 2026", "2026-06-17"},
		{"January 2, 2026", "2026-01-02"},
		{"  2026-06-17  ", "2026-06-17"},
		{"", ""},
		{"not a date", ""},
		{"2026-13-01", ""},
	}
	for _, tt := range tests {
		got, err := ParseDate(tt.in)
		if tt.want == "" {
			if err == nil {
				t.Errorf("ParseDate(%q) expected error, got %s", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDate(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got.String() != tt.want {
			t.Errorf("ParseDate(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestDaysBetweenAndWithin(t *testing.T) {
	a := MustParseDate("2026-06-01")
	b := MustParseDate("2026-06-16")
	if d := DaysBetween(a, b); d != 15 {
		t.Errorf("DaysBetween = %d, want 15", d)
	}
	if d := DaysBetween(b, a); d != 15 {
		t.Errorf("DaysBetween (reversed) = %d, want 15", d)
	}
	if !WithinDays(a, b, 15) {
		t.Error("WithinDays(15) should be true at exactly 15 days")
	}
	if WithinDays(a, b, 14) {
		t.Error("WithinDays(14) should be false at 15 days apart")
	}
}

func TestDateIsZero(t *testing.T) {
	if !(Date{}).IsZero() {
		t.Error("zero Date should report IsZero")
	}
	if MustParseDate("2026-06-17").IsZero() {
		t.Error("non-zero Date should not report IsZero")
	}
}
