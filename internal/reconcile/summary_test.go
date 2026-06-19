package reconcile

import (
	"testing"

	"github.com/tallywell/tallywell/internal/model"
)

func d(s string) model.Date { return model.MustParseDate(s) }

func TestBuildSummary(t *testing.T) {
	practices := []model.Practice{
		{ID: "own", Kind: model.PracticeOwn},
		{ID: "emp", Kind: model.PracticeEmployer},
	}
	payers := []model.Payer{
		{ID: "alma", Name: "Platform A"},
		{ID: "priv", Name: "Private Pay"},
		{ID: "job", Name: "Employer"},
	}
	records := []model.Record{
		// own practice, platform A: one paid, one unpaid (June)
		{ID: "1", Date: d("2026-06-02"), PracticeID: "own", PayerID: "alma", Status: model.StatusCompleted, Expected: 12000, Paid: 12000},
		{ID: "2", Date: d("2026-06-09"), PracticeID: "own", PayerID: "alma", Status: model.StatusCompleted, Expected: 12000},
		// own practice, private (July), partial
		{ID: "3", Date: d("2026-07-01"), PracticeID: "own", PayerID: "priv", Status: model.StatusCompleted, Expected: 15000, Paid: 5000},
		// employer (June)
		{ID: "4", Date: d("2026-06-15"), PracticeID: "emp", PayerID: "job", Status: model.StatusCompleted, Expected: 8000, Paid: 8000},
		// cancelled — excluded everywhere
		{ID: "5", Date: d("2026-06-20"), PracticeID: "own", PayerID: "alma", Status: model.StatusCancelled, Expected: 12000},
	}

	s := BuildSummary(records, practices, payers)

	if s.Overall.Sessions != 4 {
		t.Errorf("sessions = %d, want 4 (cancelled excluded)", s.Overall.Sessions)
	}
	if s.Overall.Earned != 47000 {
		t.Errorf("earned = %d, want 47000", s.Overall.Earned)
	}
	if s.Overall.Paid != 25000 {
		t.Errorf("paid = %d, want 25000", s.Overall.Paid)
	}
	if s.Overall.Outstanding != 22000 {
		t.Errorf("outstanding = %d, want 22000", s.Overall.Outstanding)
	}

	// Own vs employer split.
	if s.Employer.Earned != 8000 || s.Employer.Sessions != 1 {
		t.Errorf("employer totals = %+v", s.Employer)
	}
	if s.OwnPractice.Earned != 39000 || s.OwnPractice.Sessions != 3 {
		t.Errorf("own totals = %+v", s.OwnPractice)
	}

	// ByPayer sorted by earned desc: alma(24000) > priv(15000) > job(8000).
	if len(s.ByPayer) != 3 || s.ByPayer[0].PayerID != "alma" || s.ByPayer[2].PayerID != "job" {
		t.Fatalf("ByPayer order wrong: %+v", s.ByPayer)
	}
	if s.ByPayer[0].PayerName != "Platform A" {
		t.Errorf("payer name not joined: %q", s.ByPayer[0].PayerName)
	}

	// ByMonth sorted ascending: 2026-06 then 2026-07.
	if len(s.ByMonth) != 2 || s.ByMonth[0].Month != "2026-06" || s.ByMonth[1].Month != "2026-07" {
		t.Fatalf("ByMonth wrong: %+v", s.ByMonth)
	}
	if s.ByMonth[0].Earned != 32000 { // 12000+12000+8000
		t.Errorf("June earned = %d, want 32000", s.ByMonth[0].Earned)
	}
}

func TestBuildSummaryEmpty(t *testing.T) {
	s := BuildSummary(nil, nil, nil)
	if s.Overall.Sessions != 0 || len(s.ByPayer) != 0 || len(s.ByMonth) != 0 {
		t.Errorf("empty summary not empty: %+v", s)
	}
}

func TestMonthKeyUnknown(t *testing.T) {
	recs := []model.Record{{ID: "x", PayerID: "p", Status: model.StatusCompleted, Expected: 100}}
	s := BuildSummary(recs, nil, nil)
	if len(s.ByMonth) != 1 || s.ByMonth[0].Month != "unknown" {
		t.Errorf("zero-date record should bucket as unknown month: %+v", s.ByMonth)
	}
}
