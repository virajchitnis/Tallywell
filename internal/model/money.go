package model

import (
	"errors"
	"fmt"
	"strings"
)

// Cents is a monetary amount stored as an integer number of cents. Money is
// never represented as a float to avoid rounding artifacts.
type Cents int64

// ErrBadMoney is returned when a money string cannot be parsed.
var ErrBadMoney = errors.New("invalid money value")

// ParseMoney parses a human or CSV money string into Cents. It accepts an
// optional leading "$", thousands separators, surrounding whitespace, and
// parentheses for negatives (e.g. "($1,234.50)" -> -123450).
func ParseMoney(s string) (Cents, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrBadMoney
	}

	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	// Sign and "$" may appear in either order, e.g. "-$12" or "$-12".
	s = strings.TrimSpace(strings.TrimPrefix(s, "$"))
	if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimSpace(s[1:])
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "$"))
	if strings.HasPrefix(s, "-") {
		return 0, ErrBadMoney
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrBadMoney
	}

	whole := s
	frac := ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		whole = s[:dot]
		frac = s[dot+1:]
	}
	if whole == "" {
		whole = "0"
	}

	var dollars int64
	for i := 0; i < len(whole); i++ {
		c := whole[i]
		if c < '0' || c > '9' {
			return 0, ErrBadMoney
		}
		dollars = dollars*10 + int64(c-'0')
	}

	// Round to two decimal places using the third fractional digit.
	var cents int64
	switch frac {
	case "":
		cents = 0
	default:
		for i := 0; i < len(frac); i++ {
			if frac[i] < '0' || frac[i] > '9' {
				return 0, ErrBadMoney
			}
		}
		padded := frac + "00"
		c0 := int64(padded[0] - '0')
		c1 := int64(padded[1] - '0')
		cents = c0*10 + c1
		if len(frac) >= 3 && padded[2] >= '5' {
			cents++
		}
	}

	total := dollars*100 + cents
	if neg {
		total = -total
	}
	return Cents(total), nil
}

// String formats the amount as a plain decimal with two places, e.g. "1234.50".
func (c Cents) String() string {
	n := int64(c)
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	return fmt.Sprintf("%s%d.%02d", sign, n/100, n%100)
}

// Display formats the amount for the UI with a leading "$" and thousands
// separators, e.g. "$1,234.50".
func (c Cents) Display() string {
	n := int64(c)
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	whole := fmt.Sprintf("%d", n/100)
	var grouped strings.Builder
	for i, d := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(d)
	}
	return fmt.Sprintf("%s$%s.%02d", sign, grouped.String(), n%100)
}
