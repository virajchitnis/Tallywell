package model

// Rate is a user-configured expected amount for a payer, optionally specific to
// a service code. A Rate with an empty Service is the payer's default rate.
type Rate struct {
	ID      string
	PayerID string
	Service string // empty = default for the payer
	Amount  Cents
}

// ExpectedFor returns the best-matching rate amount for a payer and service
// from the given rates: an exact (payer, service) match wins, otherwise the
// payer's default (empty-service) rate. The second result reports whether any
// rate matched.
func ExpectedFor(rates []Rate, payerID, service string) (Cents, bool) {
	var def Cents
	haveDef := false
	for _, r := range rates {
		if r.PayerID != payerID {
			continue
		}
		if r.Service == service && service != "" {
			return r.Amount, true
		}
		if r.Service == "" {
			def = r.Amount
			haveDef = true
		}
	}
	if haveDef {
		return def, true
	}
	return 0, false
}
