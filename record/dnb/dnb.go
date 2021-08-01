package dnb

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/mpolden/journal/record"
)

const (
	firstHeaderCell   = "Dato"
	decimalSeparator  = "."
	thousandSeparator = ","
)

// Reader implements a reader for DNB-encoded (XLSX) records.
type Reader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

// NewReader returns a new reader for DNB-encoded records.
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

func (r *Reader) parseAmount(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
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
		if len(cells) < 6 {
			continue
		}
		if cells[0] == firstHeaderCell { // Header row
			continue
		}
		if cells[0] == "" { // Missing date
			continue
		}
		time, err := time.Parse("02.01.2006", cells[0])
		if err != nil {
			return nil, fmt.Errorf("invalid date: %q: %w", cells[0], err)
		}
		amountIn, err := r.parseAmount(cells[4])
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %q: %w", cells[4], err)
		}
		amountOut, err := r.parseAmount(cells[5])
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %q: %w", cells[5], err)
		}
		amount := amountIn - amountOut
		r := record.Record{
			Time:   time,
			Text:   cells[1],
			Amount: amount,
		}
		rs = append(rs, r)
	}
	return rs, nil
}
