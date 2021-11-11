package norwegian

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/record"
	"github.com/xuri/excelize/v2"
)

const (
	firstHeaderCell   = "TransactionDate"
	decimalSeparator  = "."
	thousandSeparator = ","
)

// Reader implements a reader for Norwegian-encoded (XLSX) records.
type Reader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

// NewReader returns a new reader for Norwegian-encoded records.
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

func (r *Reader) parseAmount(s string) (int64, error) {
	if strings.LastIndex(s, decimalSeparator) == len(s)-2 { // Pad single digit decimal
		s += "0"
	}
	hasDecimals := strings.Contains(s, decimalSeparator)
	v := r.replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	if !hasDecimals {
		return n * 100, nil
	}
	return n, nil
}

func (r *Reader) Read() ([]record.Record, error) {
	data, err := excelize.OpenReader(r.rd)
	if err != nil {
		return nil, err
	}
	if len(data.GetSheetList()) == 0 {
		return nil, fmt.Errorf("xlsx contains 0 sheets")
	}
	firstSheet := data.GetSheetName(0)
	rows, err := data.GetRows(firstSheet)
	if err != nil {
		return nil, err
	}
	var rs []record.Record
	for _, cells := range rows {
		if len(cells) < 7 {
			continue
		}
		if cells[0] == firstHeaderCell { // Header row
			continue
		}
		if cells[0] == "" { // Empty row
			continue
		}
		time, err := time.Parse("01-02-06", cells[0])
		if err != nil {
			return nil, fmt.Errorf("invalid date: %q: %w", cells[0], err)
		}
		amount, err := r.parseAmount(cells[6])
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %q: %w", cells[6], err)
		}
		t := record.Record{
			Time:   time,
			Text:   cells[1],
			Amount: amount,
		}
		rs = append(rs, t)
	}
	return rs, nil
}
