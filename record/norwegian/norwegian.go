package norwegian

import (
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/record"
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"
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
	data, err := ioutil.ReadAll(r.rd)
	if err != nil {
		return nil, err
	}
	f, err := xlsx.OpenBinary(data)
	if err != nil {
		return nil, err
	}
	if len(f.Sheets) == 0 {
		return nil, errors.New("xlsx contains 0 sheets")
	}
	var rs []record.Record
	for _, row := range f.Sheets[0].Rows {
		cells := row.Cells
		if len(cells) < 7 {
			continue
		}
		if cells[0].String() == firstHeaderCell { // Header row
			continue
		}
		if cells[0].String() == "" { // Empty row
			continue
		}
		time, err := time.Parse("01-02-06", cells[0].String())
		if err != nil {
			return nil, errors.Wrapf(err, "invalid date: %q", cells[0].String())
		}
		amount, err := r.parseAmount(cells[6].String())
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount: %q", cells[6].String())
		}
		t := record.Record{
			Time:   time,
			Text:   cells[1].String(),
			Amount: amount,
		}
		rs = append(rs, t)
	}
	return rs, nil
}
