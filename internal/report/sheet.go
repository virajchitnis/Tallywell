package report

import (
	"github.com/xuri/excelize/v2"

	"github.com/tallywell/tallywell/internal/model"
)

// sheet is a tiny cursor over an excelize file that appends rows to the current
// sheet and accumulates the first error, so call sites stay free of repetitive
// error handling.
type sheet struct {
	f       *excelize.File
	name    string
	nextRow int
	err     error
}

// use switches to (creating if needed) the named sheet and resets the cursor.
func (s *sheet) use(name string) {
	if s.err != nil {
		return
	}
	if _, err := s.f.NewSheet(name); err != nil {
		s.err = err
		return
	}
	s.name = name
	s.nextRow = 1
}

// row appends one row of values to the current sheet.
func (s *sheet) row(values ...any) {
	if s.err != nil {
		return
	}
	for i, v := range values {
		cell, err := excelize.CoordinatesToCellName(i+1, s.nextRow)
		if err != nil {
			s.err = err
			return
		}
		if err := s.f.SetCellValue(s.name, cell, v); err != nil {
			s.err = err
			return
		}
	}
	s.nextRow++
}

func practiceNameMap(practices []model.Practice) map[string]string {
	m := make(map[string]string, len(practices))
	for _, p := range practices {
		m[p.ID] = p.Name
	}
	return m
}

func payerNameMap(payers []model.Payer) map[string]string {
	m := make(map[string]string, len(payers))
	for _, p := range payers {
		m[p.ID] = p.Name
	}
	return m
}
