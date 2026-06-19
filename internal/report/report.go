// Package report generates the Tallywell spreadsheet export: a Sessions ledger,
// a Dashboard of roll-ups, a Tax summary (own-practice vs employer income), and
// an Unmatched/Review tab. The workbook is the CPA-friendly, human-readable
// mirror of the encrypted data.
package report

import (
	"io"

	"github.com/xuri/excelize/v2"

	"github.com/tallywell/tallywell/internal/model"
	"github.com/tallywell/tallywell/internal/reconcile"
)

// Input is everything needed to build the workbook.
type Input struct {
	Records   []model.Record
	Practices []model.Practice
	Payers    []model.Payer
	Unmatched []model.Record // payouts that reconciled to no session
}

// dollars converts integer cents to a float for spreadsheet cells.
func dollars(c model.Cents) float64 { return float64(c) / 100 }

// WriteXLSX builds the workbook and writes it to w.
func WriteXLSX(w io.Writer, in Input) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sw := &sheet{f: f}

	practiceName := practiceNameMap(in.Practices)
	payerName := payerNameMap(in.Payers)

	writeSessions(sw, in.Records, practiceName, payerName)
	writeDashboard(sw, in)
	writeTax(sw, in)
	writeUnmatched(sw, in.Unmatched, practiceName, payerName)

	// Remove the default sheet created by excelize.
	if idx, err := f.GetSheetIndex("Sheet1"); err == nil && idx >= 0 {
		_ = f.DeleteSheet("Sheet1")
	}
	if sw.err != nil {
		return sw.err
	}
	_, err := f.WriteTo(w)
	return err
}

func writeSessions(sw *sheet, records []model.Record, practiceName, payerName map[string]string) {
	sw.use("Sessions")
	sw.row("Date", "Client", "Practice", "Payer", "Service", "Status", "Expected", "Paid", "Outstanding", "Date paid", "Source")
	for _, r := range records {
		sw.row(
			dateCell(r.Date), r.ClientID, practiceName[r.PracticeID], payerName[r.PayerID],
			r.Service, string(r.Status),
			dollars(r.Expected), dollars(r.Paid), dollars(r.Outstanding()),
			dateCell(r.DatePaid), r.Source,
		)
	}
}

func writeDashboard(sw *sheet, in Input) {
	s := reconcile.BuildSummary(in.Records, in.Practices, in.Payers)
	sw.use("Dashboard")

	sw.row("Overall")
	sw.row("Sessions seen", s.Overall.Sessions)
	sw.row("Earned", dollars(s.Overall.Earned))
	sw.row("Paid", dollars(s.Overall.Paid))
	sw.row("Outstanding", dollars(s.Overall.Outstanding))
	sw.row()

	sw.row("Income by payer")
	sw.row("Payer", "Sessions", "Earned", "Paid", "Outstanding")
	for _, p := range s.ByPayer {
		sw.row(p.PayerName, p.Sessions, dollars(p.Earned), dollars(p.Paid), dollars(p.Outstanding))
	}
	sw.row()

	sw.row("Income by month")
	sw.row("Month", "Sessions", "Earned", "Paid", "Outstanding")
	for _, m := range s.ByMonth {
		sw.row(m.Month, m.Sessions, dollars(m.Earned), dollars(m.Paid), dollars(m.Outstanding))
	}
}

func writeTax(sw *sheet, in Input) {
	s := reconcile.BuildSummary(in.Records, in.Practices, in.Payers)
	sw.use("Tax summary")
	sw.row("Informational only — not tax advice. See DISCLAIMER.")
	sw.row()
	sw.row("Category", "Sessions", "Earned", "Paid")
	sw.row("Own practice (self-employed)", s.OwnPractice.Sessions, dollars(s.OwnPractice.Earned), dollars(s.OwnPractice.Paid))
	sw.row("Employer (W-2)", s.Employer.Sessions, dollars(s.Employer.Earned), dollars(s.Employer.Paid))
}

func writeUnmatched(sw *sheet, unmatched []model.Record, practiceName, payerName map[string]string) {
	sw.use("Unmatched")
	sw.row("Payouts that did not match a session — review these.")
	sw.row()
	sw.row("Date", "Client", "Payer", "Service", "Paid", "Source")
	for _, r := range unmatched {
		sw.row(dateCell(r.Date), r.ClientID, payerName[r.PayerID], r.Service, dollars(r.Paid), r.Source)
	}
}

func dateCell(d model.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}
