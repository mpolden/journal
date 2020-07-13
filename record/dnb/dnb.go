package dnb

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/mpolden/journal/record"
	"github.com/tealeg/xlsx"
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
	data, err := ioutil.ReadAll(r.rd)
	if err != nil {
		return nil, err
	}
	f, err := xlsx.OpenBinary(data)
	if err != nil {
		return nil, err
	}
	if len(f.Sheets) == 0 {
		return nil, fmt.Errorf("xlsx contains 0 sheets")
	}
	var rs []record.Record
	for _, row := range f.Sheets[0].Rows {
		cells := row.Cells
		if len(cells) < 6 {
			continue
		}
		if cells[0].String() == firstHeaderCell { // Header row
			continue
		}
		time, err := cells[0].GetTime(false)
		if err != nil {
			return nil, err
		}
		amountIn, err := r.parseAmount(cells[4].String())
		if err != nil {
			return nil, err
		}
		amountOut, err := r.parseAmount(cells[5].String())
		if err != nil {
			return nil, err
		}
		amount := amountIn - amountOut
		r := record.Record{
			Time:   time,
			Text:   cells[1].String(),
			Amount: amount,
		}
		rs = append(rs, r)
	}
	return rs, nil
}
