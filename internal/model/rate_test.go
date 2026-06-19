package model

import "testing"

func TestExpectedFor(t *testing.T) {
	rates := []Rate{
		{ID: "1", PayerID: "p1", Service: "", Amount: 10000},
		{ID: "2", PayerID: "p1", Service: "90837", Amount: 12000},
		{ID: "3", PayerID: "p2", Service: "", Amount: 8000},
	}
	tests := []struct {
		payer, service string
		want           Cents
		ok             bool
	}{
		{"p1", "90837", 12000, true}, // exact service match
		{"p1", "90834", 10000, true}, // falls back to payer default
		{"p1", "", 10000, true},      // default
		{"p2", "90837", 8000, true},  // p2 default
		{"p3", "90837", 0, false},    // unknown payer
	}
	for _, tt := range tests {
		got, ok := ExpectedFor(rates, tt.payer, tt.service)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ExpectedFor(%q,%q) = %d,%v want %d,%v", tt.payer, tt.service, got, ok, tt.want, tt.ok)
		}
	}
}
