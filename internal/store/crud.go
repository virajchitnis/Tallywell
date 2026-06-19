package store

import (
	"database/sql"

	"github.com/tallywell/tallywell/internal/model"
)

func queryPractices(db *sql.DB) ([]model.Practice, error) {
	rows, err := db.Query(`SELECT id,name,kind FROM practices ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Practice
	for rows.Next() {
		var p model.Practice
		var kind string
		if err := rows.Scan(&p.ID, &p.Name, &kind); err != nil {
			return nil, err
		}
		p.Kind = model.PracticeKind(kind)
		out = append(out, p)
	}
	return out, rows.Err()
}

func queryPayers(db *sql.DB) ([]model.Payer, error) {
	rows, err := db.Query(`SELECT id,name,practice_id,kind,importer_key FROM payers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Payer
	for rows.Next() {
		var p model.Payer
		var kind string
		if err := rows.Scan(&p.ID, &p.Name, &p.PracticeID, &kind, &p.ImporterKey); err != nil {
			return nil, err
		}
		p.Kind = model.PayerKind(kind)
		out = append(out, p)
	}
	return out, rows.Err()
}

func queryRates(db *sql.DB) ([]model.Rate, error) {
	rows, err := db.Query(`SELECT id,payer_id,service,amount FROM rates ORDER BY payer_id,service`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Rate
	for rows.Next() {
		var r model.Rate
		var amt int64
		if err := rows.Scan(&r.ID, &r.PayerID, &r.Service, &amt); err != nil {
			return nil, err
		}
		r.Amount = model.Cents(amt)
		out = append(out, r)
	}
	return out, rows.Err()
}

func queryRecords(db *sql.DB) ([]model.Record, error) {
	rows, err := db.Query(`SELECT id,date,client_id,practice_id,payer_id,service,status,expected,paid,date_paid,source
		FROM records ORDER BY date,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Record
	for rows.Next() {
		var r model.Record
		var date, datePaid, status string
		var expected, paid int64
		if err := rows.Scan(&r.ID, &date, &r.ClientID, &r.PracticeID, &r.PayerID, &r.Service,
			&status, &expected, &paid, &datePaid, &r.Source); err != nil {
			return nil, err
		}
		r.Date = parseDateStr(date)
		r.DatePaid = parseDateStr(datePaid)
		r.Status = model.SessionStatus(status)
		r.Expected = model.Cents(expected)
		r.Paid = model.Cents(paid)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Practices returns all practices.
func (s *Store) Practices() ([]model.Practice, error) { return queryPractices(s.db) }

// Payers returns all payers.
func (s *Store) Payers() ([]model.Payer, error) { return queryPayers(s.db) }

// Rates returns all rates.
func (s *Store) Rates() ([]model.Rate, error) { return queryRates(s.db) }

// Records returns all records, ordered by date.
func (s *Store) Records() ([]model.Record, error) { return queryRecords(s.db) }

// PutPractice inserts or updates a practice, then persists.
func (s *Store) PutPractice(p model.Practice) error {
	if _, err := s.db.Exec(`INSERT INTO practices(id,name,kind) VALUES(?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name,kind=excluded.kind`,
		p.ID, p.Name, string(p.Kind)); err != nil {
		return err
	}
	return s.persist()
}

// PutPayer inserts or updates a payer, then persists.
func (s *Store) PutPayer(p model.Payer) error {
	if _, err := s.db.Exec(`INSERT INTO payers(id,name,practice_id,kind,importer_key) VALUES(?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name,practice_id=excluded.practice_id,kind=excluded.kind,importer_key=excluded.importer_key`,
		p.ID, p.Name, p.PracticeID, string(p.Kind), p.ImporterKey); err != nil {
		return err
	}
	return s.persist()
}

// PutRate inserts or updates a rate, then persists.
func (s *Store) PutRate(r model.Rate) error {
	if _, err := s.db.Exec(`INSERT INTO rates(id,payer_id,service,amount) VALUES(?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET payer_id=excluded.payer_id,service=excluded.service,amount=excluded.amount`,
		r.ID, r.PayerID, r.Service, int64(r.Amount)); err != nil {
		return err
	}
	return s.persist()
}

// PutRecord inserts or updates a record, then persists.
func (s *Store) PutRecord(r model.Record) error {
	if err := s.putRecordNoPersist(r); err != nil {
		return err
	}
	return s.persist()
}

// PutRecords inserts or updates many records in one transaction, then persists
// once. Used by importers to avoid an encrypt-write per row.
func (s *Store) PutRecords(recs []model.Record) error {
	for _, r := range recs {
		if err := s.putRecordNoPersist(r); err != nil {
			return err
		}
	}
	return s.persist()
}

func (s *Store) putRecordNoPersist(r model.Record) error {
	_, err := s.db.Exec(`INSERT INTO records(id,date,client_id,practice_id,payer_id,service,status,expected,paid,date_paid,source)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET date=excluded.date,client_id=excluded.client_id,practice_id=excluded.practice_id,
		  payer_id=excluded.payer_id,service=excluded.service,status=excluded.status,expected=excluded.expected,
		  paid=excluded.paid,date_paid=excluded.date_paid,source=excluded.source`,
		r.ID, dateStr(r.Date), r.ClientID, r.PracticeID, r.PayerID, r.Service,
		string(r.Status), int64(r.Expected), int64(r.Paid), dateStr(r.DatePaid), r.Source)
	return err
}

// DeleteRecord removes a record by id, then persists.
func (s *Store) DeleteRecord(id string) error {
	if _, err := s.db.Exec(`DELETE FROM records WHERE id=?`, id); err != nil {
		return err
	}
	return s.persist()
}
