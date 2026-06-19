package reconcile

import (
	"testing"

	"github.com/tallywell/tallywell/internal/model"
)

func TestMatchFillsPaid(t *testing.T) {
	sessions := []model.Record{
		{ID: "s1", Date: d("2026-06-01"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 12000},
	}
	payouts := []model.Record{
		{ClientID: "AB", Service: "90837", Date: d("2026-06-20"), Paid: 11000},
	}
	res := Match(sessions, payouts, DefaultWindowDays)
	if len(res.UnmatchedPayouts) != 0 {
		t.Fatalf("expected no unmatched payouts, got %d", len(res.UnmatchedPayouts))
	}
	m := res.Merged[0]
	if m.Paid != 11000 || m.DatePaid.String() != "2026-06-20" {
		t.Errorf("payout not merged: %+v", m)
	}
}

func TestMatchOutOfWindow(t *testing.T) {
	sessions := []model.Record{
		{ID: "s1", Date: d("2026-06-01"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 12000},
	}
	payouts := []model.Record{
		{ClientID: "AB", Service: "90837", Date: d("2026-09-01"), Paid: 12000}, // ~92 days later
	}
	res := Match(sessions, payouts, DefaultWindowDays)
	if len(res.UnmatchedPayouts) != 1 {
		t.Fatalf("expected 1 unmatched payout, got %d", len(res.UnmatchedPayouts))
	}
	if res.Merged[0].Paid != 0 {
		t.Errorf("session should remain unpaid: %+v", res.Merged[0])
	}
}

func TestMatchDifferentClientOrService(t *testing.T) {
	sessions := []model.Record{
		{ID: "s1", Date: d("2026-06-01"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 12000},
	}
	payouts := []model.Record{
		{ClientID: "CD", Service: "90837", Date: d("2026-06-10"), Paid: 12000},
		{ClientID: "AB", Service: "90834", Date: d("2026-06-10"), Paid: 9000},
	}
	res := Match(sessions, payouts, DefaultWindowDays)
	if len(res.UnmatchedPayouts) != 2 {
		t.Fatalf("expected both payouts unmatched, got %d", len(res.UnmatchedPayouts))
	}
}

func TestMatchClosestSessionWins(t *testing.T) {
	sessions := []model.Record{
		{ID: "far", Date: d("2026-06-01"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 12000},
		{ID: "near", Date: d("2026-06-18"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 12000},
	}
	payouts := []model.Record{
		{ClientID: "AB", Service: "90837", Date: d("2026-06-20"), Paid: 12000},
	}
	res := Match(sessions, payouts, DefaultWindowDays)
	var nearPaid, farPaid bool
	for _, m := range res.Merged {
		if m.ID == "near" && m.Paid == 12000 {
			nearPaid = true
		}
		if m.ID == "far" && m.Paid == 12000 {
			farPaid = true
		}
	}
	if !nearPaid || farPaid {
		t.Errorf("expected the nearer session to be paid only: %+v", res.Merged)
	}
}

func TestMatchSkipsAlreadyPaid(t *testing.T) {
	sessions := []model.Record{
		{ID: "s1", Date: d("2026-06-01"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 12000, Paid: 12000, DatePaid: d("2026-06-10")},
	}
	payouts := []model.Record{
		{ClientID: "AB", Service: "90837", Date: d("2026-06-15"), Paid: 12000},
	}
	res := Match(sessions, payouts, DefaultWindowDays)
	if len(res.UnmatchedPayouts) != 1 {
		t.Fatalf("payout should not re-match an already-paid session; unmatched=%d", len(res.UnmatchedPayouts))
	}
}

func TestMatchUpdatesExpectedFromPayout(t *testing.T) {
	sessions := []model.Record{
		{ID: "s1", Date: d("2026-06-01"), ClientID: "AB", Service: "90837", Status: model.StatusCompleted, Expected: 0},
	}
	payouts := []model.Record{
		{ClientID: "AB", Service: "90837", Date: d("2026-06-12"), Paid: 11500, Expected: 11500},
	}
	res := Match(sessions, payouts, DefaultWindowDays)
	if res.Merged[0].Expected != 11500 {
		t.Errorf("expected amount should be filled from payout: %+v", res.Merged[0])
	}
}

func TestMatchEmptyInputs(t *testing.T) {
	res := Match(nil, nil, DefaultWindowDays)
	if len(res.Merged) != 0 || len(res.UnmatchedPayouts) != 0 {
		t.Errorf("empty match should be empty: %+v", res)
	}
}
