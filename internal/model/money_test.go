package model

import "testing"

func TestParseMoney(t *testing.T) {
	tests := []struct {
		in      string
		want    Cents
		wantErr bool
	}{
		{"0", 0, false},
		{"$0.00", 0, false},
		{"150", 15000, false},
		{"150.00", 15000, false},
		{"$1,234.50", 123450, false},
		{"  $1,234.50  ", 123450, false},
		{".5", 50, false},
		{"99.9", 9990, false},
		{"1.005", 101, false},   // rounds half up on 3rd digit
		{"1.004", 100, false},   // rounds down
		{"2.999", 300, false},   // rounds up across the dollar
		{"($1,234.50)", -123450, false}, // parenthesized negative
		{"-12.34", -1234, false},
		{"", 0, true},
		{"abc", 0, true},
		{"1.2.3", 0, true},
		{"$", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseMoney(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseMoney(%q) expected error, got %v", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseMoney(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseMoney(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestCentsString(t *testing.T) {
	tests := []struct {
		in       Cents
		str      string
		display  string
	}{
		{0, "0.00", "$0.00"},
		{15000, "150.00", "$150.00"},
		{123450, "1234.50", "$1,234.50"},
		{100050000, "1000500.00", "$1,000,500.00"},
		{-1234, "-12.34", "-$12.34"},
		{5, "0.05", "$0.05"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.str {
			t.Errorf("Cents(%d).String() = %q, want %q", tt.in, got, tt.str)
		}
		if got := tt.in.Display(); got != tt.display {
			t.Errorf("Cents(%d).Display() = %q, want %q", tt.in, got, tt.display)
		}
	}
}

func TestParseMoneyRoundTrip(t *testing.T) {
	for _, s := range []string{"0.00", "150.00", "1234.50", "0.05"} {
		c, err := ParseMoney(s)
		if err != nil {
			t.Fatalf("ParseMoney(%q): %v", s, err)
		}
		if c.String() != s {
			t.Errorf("round trip %q -> %d -> %q", s, c, c.String())
		}
	}
}
