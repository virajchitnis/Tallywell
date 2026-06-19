package store

import (
	"database/sql"
	"fmt"

	"github.com/tallywell/tallywell/internal/model"
)

// Snapshot is the serializable, persisted form of all data. It is encoded as
// JSON, encrypted, and written to disk; the in-memory SQLite database is
// rebuilt from it on load.
type Snapshot struct {
	SchemaVersion int              `json:"schema_version"`
	Practices     []model.Practice `json:"practices"`
	Payers        []model.Payer    `json:"payers"`
	Rates         []model.Rate     `json:"rates"`
	Records       []model.Record   `json:"records"`
}

// emptySnapshot returns a fresh snapshot at the current schema version.
func emptySnapshot() *Snapshot {
	return &Snapshot{SchemaVersion: schemaVersion}
}

// load rebuilds the in-memory schema and populates it from the snapshot.
func (s *Snapshot) load(db *sql.DB) error {
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("store: create schema: %w", err)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`INSERT INTO meta(key,value) VALUES('schema_version',?)`, schemaVersion); err != nil {
		return err
	}
	for _, p := range s.Practices {
		if _, err := tx.Exec(`INSERT INTO practices(id,name,kind) VALUES(?,?,?)`,
			p.ID, p.Name, string(p.Kind)); err != nil {
			return err
		}
	}
	for _, p := range s.Payers {
		if _, err := tx.Exec(`INSERT INTO payers(id,name,practice_id,kind,importer_key) VALUES(?,?,?,?,?)`,
			p.ID, p.Name, p.PracticeID, string(p.Kind), p.ImporterKey); err != nil {
			return err
		}
	}
	for _, r := range s.Rates {
		if _, err := tx.Exec(`INSERT INTO rates(id,payer_id,service,amount) VALUES(?,?,?,?)`,
			r.ID, r.PayerID, r.Service, int64(r.Amount)); err != nil {
			return err
		}
	}
	for _, r := range s.Records {
		if _, err := tx.Exec(`INSERT INTO records(id,date,client_id,practice_id,payer_id,service,status,expected,paid,date_paid,source)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			r.ID, dateStr(r.Date), r.ClientID, r.PracticeID, r.PayerID, r.Service,
			string(r.Status), int64(r.Expected), int64(r.Paid), dateStr(r.DatePaid), r.Source); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// snapshot reads the full in-memory state back into a Snapshot for persistence.
func snapshotFromDB(db *sql.DB) (*Snapshot, error) {
	out := emptySnapshot()

	prs, err := queryPractices(db)
	if err != nil {
		return nil, err
	}
	out.Practices = prs

	pys, err := queryPayers(db)
	if err != nil {
		return nil, err
	}
	out.Payers = pys

	rts, err := queryRates(db)
	if err != nil {
		return nil, err
	}
	out.Rates = rts

	recs, err := queryRecords(db)
	if err != nil {
		return nil, err
	}
	out.Records = recs
	return out, nil
}

func dateStr(d model.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}

func parseDateStr(s string) model.Date {
	if s == "" {
		return model.Date{}
	}
	d, err := model.ParseDate(s)
	if err != nil {
		return model.Date{}
	}
	return d
}
