package reconcile

import (
	"sort"

	"github.com/tallywell/tallywell/internal/model"
)

// DefaultWindowDays is the default tolerance between a session date and the
// payout date when matching, accounting for payers that report payouts weeks
// after the session.
const DefaultWindowDays = 45

// MatchResult is the outcome of reconciling imported payouts against existing
// sessions. Nothing is dropped: every payout ends up either merged into a
// session or in UnmatchedPayouts for review.
type MatchResult struct {
	// Merged are existing sessions, with Paid/DatePaid filled in where a payout
	// matched. Sessions with no payout are included unchanged.
	Merged []model.Record
	// UnmatchedPayouts are payout rows that matched no existing session.
	UnmatchedPayouts []model.Record
}

// matchable reports whether a payout can settle a session: same client and
// service, within the day window, and not already fully paid.
func matchable(session, payout model.Record, windowDays int) bool {
	if session.ClientID != payout.ClientID || session.Service != payout.Service {
		return false
	}
	if session.IsPaid() {
		return false
	}
	return model.WithinDays(session.Date, payout.Date, windowDays)
}

// Match reconciles payouts against sessions within windowDays. Each payout is
// matched to at most one session (the closest unpaid one by date); the session
// is marked paid with the payout's amount and date. Unmatched payouts are
// returned separately.
func Match(sessions, payouts []model.Record, windowDays int) MatchResult {
	merged := make([]model.Record, len(sessions))
	copy(merged, sessions)
	claimed := make([]bool, len(merged))

	var unmatched []model.Record

	// Process payouts oldest-first for deterministic assignment.
	order := make([]int, len(payouts))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return model.DaysBetween(model.Date{}, payouts[order[a]].Date) <
			model.DaysBetween(model.Date{}, payouts[order[b]].Date)
	})

	for _, pi := range order {
		payout := payouts[pi]
		best := -1
		bestDist := windowDays + 1
		for i := range merged {
			if claimed[i] {
				continue
			}
			if !matchable(merged[i], payout, windowDays) {
				continue
			}
			if d := model.DaysBetween(merged[i].Date, payout.Date); d < bestDist {
				best = i
				bestDist = d
			}
		}
		if best < 0 {
			unmatched = append(unmatched, payout)
			continue
		}
		claimed[best] = true
		merged[best].Paid = payout.Paid
		merged[best].DatePaid = payout.Date
		if payout.Expected != 0 {
			merged[best].Expected = payout.Expected
		}
	}

	return MatchResult{Merged: merged, UnmatchedPayouts: unmatched}
}
