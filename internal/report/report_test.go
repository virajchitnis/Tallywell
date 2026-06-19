package report

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/tallywell/tallywell/internal/model"
)

func d(s string) model.Date { return model.MustParseDate(s) }

func sampleInput() Input {
	return Input{
		Practices: []model.Practice{
			{ID: "own", Name: "Own Practice", Kind: model.PracticeOwn},
			{ID: "emp", Name: "Day Job", Kind: model.PracticeEmployer},
		},
		Payers: []model.Payer{
			{ID: "alma", Name: "Platform A"},
			{ID: "job", Name: "Day Job Payer"},
		},
		Records: []model.Record{
			{ID: "1", Date: d("2026-06-02"), ClientID: "AB", PracticeID: "own", PayerID: "alma", Service: "90837", Status: model.StatusCompleted, Expected: 12000, Paid: 12000, DatePaid: d("2026-06-20")},
			{ID: "2", Date: d("2026-06-09"), ClientID: "CD", PracticeID: "own", PayerID: "alma", Service: "90837", Status: model.StatusCompleted, Expected: 12000},
			{ID: "3", Date: d("2026-06-15"), ClientID: "EF", PracticeID: "emp", PayerID: "job", Service: "90834", Status: model.StatusCompleted, Expected: 8000, Paid: 8000},
		},
		Unmatched: []model.Record{
			{Date: d("2026-06-25"), ClientID: "ZZ", PayerID: "alma", Service: "90837", Paid: 12000, Source: "platform_a"},
		},
	}
}

func TestWriteXLSXStructure(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, sampleInput()); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	names := f.GetSheetList()
	want := map[string]bool{"Sessions": true, "Dashboard": true, "Tax summary": true, "Unmatched": true}
	for _, n := range names {
		delete(want, n)
	}
	if len(want) != 0 {
		t.Fatalf("missing sheets: %v (have %v)", want, names)
	}
	if has, _ := f.GetSheetIndex("Sheet1"); has != -1 {
		t.Error("default Sheet1 should have been removed")
	}

	// Sessions: header + 3 data rows.
	rows, err := f.GetRows("Sessions")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("Sessions rows = %d, want 4", len(rows))
	}
	if rows[0][0] != "Date" || rows[0][6] != "Expected" {
		t.Errorf("unexpected Sessions header: %v", rows[0])
	}
	if rows[1][3] != "Platform A" {
		t.Errorf("payer name not resolved in Sessions: %v", rows[1])
	}
}

func TestWriteXLSXDashboardValues(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, sampleInput()); err != nil {
		t.Fatal(err)
	}
	f, _ := excelize.OpenReader(&buf)
	defer f.Close()

	rows, err := f.GetRows("Dashboard")
	if err != nil {
		t.Fatal(err)
	}
	// Find the "Earned" overall line and check it equals 32000 cents = 320.
	var earned string
	for _, r := range rows {
		if len(r) >= 2 && r[0] == "Earned" {
			earned = r[1]
			break
		}
	}
	if earned != "320" {
		t.Errorf("overall Earned = %q, want 320", earned)
	}
}

func TestWriteXLSXTaxSplit(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, sampleInput()); err != nil {
		t.Fatal(err)
	}
	f, _ := excelize.OpenReader(&buf)
	defer f.Close()
	rows, _ := f.GetRows("Tax summary")

	var own, emp string
	for _, r := range rows {
		if len(r) >= 3 {
			switch r[0] {
			case "Own practice (self-employed)":
				own = r[2]
			case "Employer (W-2)":
				emp = r[2]
			}
		}
	}
	if own != "240" { // 12000+12000 cents = 240
		t.Errorf("own earned = %q, want 240", own)
	}
	if emp != "80" { // 8000 cents = 80
		t.Errorf("employer earned = %q, want 80", emp)
	}
}

func TestWriteXLSXEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, Input{}); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if len(f.GetSheetList()) != 4 {
		t.Errorf("expected 4 sheets even when empty, got %v", f.GetSheetList())
	}
}
