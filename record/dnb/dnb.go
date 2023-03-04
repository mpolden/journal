package dnb

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
	rows, err := data.GetRows(firstSheet, excelize.Options{RawCellValue: true})
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
		excelTime, err := strconv.ParseFloat(cells[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid time value: %q: %w", cells[0], err)
		}
		recordTime, err := excelize.ExcelDateToTime(excelTime, false)
		if err != nil {
			return nil, fmt.Errorf("invalid date: %f: %w", excelTime, err)
		}
		recordTime = recordTime.Truncate(24 * time.Hour)
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
			Time:   recordTime,
			Text:   cells[1],
			Amount: amount,
		}
		rs = append(rs, r)
	}
	return rs, nil
}
