// Package reconcile turns raw session/payout records into the roll-ups the
// dashboard and reports need (income by month and payer, earned vs paid vs
// outstanding, employer vs own-practice split) and matches imported payouts to
// existing sessions.
package reconcile

import (
	"fmt"
	"sort"

	"github.com/tallywell/tallywell/internal/model"
)

// Totals is a set of money/count aggregates.
type Totals struct {
	Earned      model.Cents
	Paid        model.Cents
	Outstanding model.Cents
	Sessions    int
}

func (t *Totals) add(r model.Record) {
	if !r.Status.CountsAsSeen() {
		return
	}
	t.Earned += r.Expected
	t.Paid += r.Paid
	t.Outstanding += r.Outstanding()
	t.Sessions++
}

// PayerTotals is per-payer aggregates with display context.
type PayerTotals struct {
	PayerID   string
	PayerName string
	Totals
}

// MonthTotals is per-calendar-month aggregates. Month is "YYYY-MM".
type MonthTotals struct {
	Month string
	Totals
}

// Summary is the full set of roll-ups over a collection of records.
type Summary struct {
	Overall      Totals
	ByPayer      []PayerTotals
	ByMonth      []MonthTotals
	OwnPractice  Totals // records whose practice is PracticeOwn
	Employer     Totals // records whose practice is PracticeEmployer
}

func monthKey(d model.Date) string {
	if d.IsZero() {
		return "unknown"
	}
	return fmt.Sprintf("%04d-%02d", d.Year, int(d.Month))
}

// BuildSummary computes all roll-ups. practices and payers provide display
// names and practice kinds; records missing a known practice/payer are still
// counted in the overall and month totals.
func BuildSummary(records []model.Record, practices []model.Practice, payers []model.Payer) Summary {
	practiceKind := make(map[string]model.PracticeKind, len(practices))
	for _, p := range practices {
		practiceKind[p.ID] = p.Kind
	}
	payerName := make(map[string]string, len(payers))
	for _, p := range payers {
		payerName[p.ID] = p.Name
	}

	var s Summary
	byPayer := make(map[string]*PayerTotals)
	byMonth := make(map[string]*MonthTotals)

	for _, r := range records {
		if !r.Status.CountsAsSeen() {
			continue
		}
		s.Overall.add(r)

		pt := byPayer[r.PayerID]
		if pt == nil {
			pt = &PayerTotals{PayerID: r.PayerID, PayerName: payerName[r.PayerID]}
			byPayer[r.PayerID] = pt
		}
		pt.add(r)

		mk := monthKey(r.Date)
		mt := byMonth[mk]
		if mt == nil {
			mt = &MonthTotals{Month: mk}
			byMonth[mk] = mt
		}
		mt.add(r)

		switch practiceKind[r.PracticeID] {
		case model.PracticeEmployer:
			s.Employer.add(r)
		case model.PracticeOwn:
			s.OwnPractice.add(r)
		}
	}

	for _, pt := range byPayer {
		s.ByPayer = append(s.ByPayer, *pt)
	}
	sort.Slice(s.ByPayer, func(i, j int) bool {
		if s.ByPayer[i].Earned != s.ByPayer[j].Earned {
			return s.ByPayer[i].Earned > s.ByPayer[j].Earned
		}
		return s.ByPayer[i].PayerID < s.ByPayer[j].PayerID
	})

	for _, mt := range byMonth {
		s.ByMonth = append(s.ByMonth, *mt)
	}
	sort.Slice(s.ByMonth, func(i, j int) bool {
		return s.ByMonth[i].Month < s.ByMonth[j].Month
	})

	return s
}
