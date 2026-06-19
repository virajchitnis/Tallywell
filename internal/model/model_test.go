package model

import "testing"

func TestSessionStatus(t *testing.T) {
	if !StatusCompleted.Valid() || !StatusScheduled.Valid() {
		t.Error("known statuses should be valid")
	}
	if SessionStatus("bogus").Valid() {
		t.Error("unknown status should be invalid")
	}
	if !StatusCompleted.CountsAsSeen() {
		t.Error("completed should count as seen")
	}
	for _, s := range []SessionStatus{StatusScheduled, StatusNoShow, StatusCancelled} {
		if s.CountsAsSeen() {
			t.Errorf("%s should not count as seen", s)
		}
	}
}

func TestRecordOutstandingAndPaid(t *testing.T) {
	tests := []struct {
		name        string
		rec         Record
		outstanding Cents
		paid        bool
	}{
		{
			name:        "completed unpaid",
			rec:         Record{Status: StatusCompleted, Expected: 12000},
			outstanding: 12000,
			paid:        false,
		},
		{
			name:        "completed fully paid",
			rec:         Record{Status: StatusCompleted, Expected: 12000, Paid: 12000},
			outstanding: 0,
			paid:        true,
		},
		{
			name:        "completed partially paid",
			rec:         Record{Status: StatusCompleted, Expected: 12000, Paid: 5000},
			outstanding: 7000,
			paid:        false,
		},
		{
			name:        "overpaid clamps to zero outstanding",
			rec:         Record{Status: StatusCompleted, Expected: 12000, Paid: 13000},
			outstanding: 0,
			paid:        true,
		},
		{
			name:        "cancelled owes nothing",
			rec:         Record{Status: StatusCancelled, Expected: 12000},
			outstanding: 0,
			paid:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rec.Outstanding(); got != tt.outstanding {
				t.Errorf("Outstanding() = %d, want %d", got, tt.outstanding)
			}
			if got := tt.rec.IsPaid(); got != tt.paid {
				t.Errorf("IsPaid() = %v, want %v", got, tt.paid)
			}
		})
	}
}
