// Package model defines Tallywell's core domain types: user-configured
// practices and payers, and the normalized session/payout record that both
// manual entry and CSV imports produce. Nothing here hardcodes any real-world
// practice, payer, or person — those are all user-supplied data.
package model

// PracticeKind distinguishes the user's own practice from an employer where
// they see clients as staff.
type PracticeKind string

const (
	// PracticeOwn is the user's own practice (typically self-employed income).
	PracticeOwn PracticeKind = "own"
	// PracticeEmployer is an employer the user works for (typically W-2 income).
	PracticeEmployer PracticeKind = "employer"
)

// Practice is a user-defined place of work. No names are shipped with the app.
type Practice struct {
	ID   string
	Name string
	Kind PracticeKind
}

// PayerKind categorizes how a payer pays for sessions.
type PayerKind string

const (
	// PayerInsurancePlatform is a credentialing/billing platform that pays a
	// contracted per-session rate, usually in arrears (e.g. Alma, Headway).
	PayerInsurancePlatform PayerKind = "insurance_platform"
	// PayerPrivate is direct private pay collected by the therapist.
	PayerPrivate PayerKind = "private"
	// PayerEmployer is compensation paid by an employer.
	PayerEmployer PayerKind = "employer"
)

// Payer is a user-defined source of income attached to a practice. ImporterKey
// names a generic CSV importer integration (e.g. "alma") or "manual" when there
// is no import; it never ties the app to a specific user's practice.
type Payer struct {
	ID          string
	Name        string
	PracticeID  string
	Kind        PayerKind
	ImporterKey string
}

// SessionStatus is the lifecycle state of a session.
type SessionStatus string

const (
	StatusScheduled SessionStatus = "scheduled"
	StatusCompleted SessionStatus = "completed"
	StatusNoShow    SessionStatus = "no_show"
	StatusCancelled SessionStatus = "cancelled"
)

// Valid reports whether s is a known status.
func (s SessionStatus) Valid() bool {
	switch s {
	case StatusScheduled, StatusCompleted, StatusNoShow, StatusCancelled:
		return true
	}
	return false
}

// CountsAsSeen reports whether the status represents a session that was
// actually delivered (and therefore expected to generate income).
func (s SessionStatus) CountsAsSeen() bool {
	return s == StatusCompleted
}

// Record is the normalized unit shared by manual entries and imported payout
// rows. Only minimal, money-relevant fields are kept — never client names,
// notes, diagnoses, or addresses (see SECURITY-AND-HIPAA.md).
type Record struct {
	ID        string
	Date      Date
	ClientID  string // initials or opaque code — never a full name
	PracticeID string
	PayerID   string
	Service   string // service/CPT code, e.g. "90837"
	Status    SessionStatus

	Expected Cents // amount expected for this session
	Paid     Cents // amount actually paid (0 until a payout is matched)
	DatePaid Date  // zero until paid

	// Source identifies where the record came from: "manual" or an importer
	// key plus original file, for traceability.
	Source string
}

// Outstanding reports the amount still owed for a seen-but-unpaid session.
func (r Record) Outstanding() Cents {
	if !r.Status.CountsAsSeen() {
		return 0
	}
	owed := r.Expected - r.Paid
	if owed < 0 {
		return 0
	}
	return owed
}

// IsPaid reports whether the record has been fully paid.
func (r Record) IsPaid() bool {
	return r.Status.CountsAsSeen() && r.Paid >= r.Expected && r.Expected > 0
}
