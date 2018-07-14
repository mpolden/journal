package norwegian

import (
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/bank"
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"
)

const (
	firstHeaderCell   = "TransactionDate"
	decimalSeparator  = "."
	thousandSeparator = ","
)

var replacer = strings.NewReplacer(decimalSeparator, "", thousandSeparator, "")

func parseAmount(s string) (int64, error) {
	if strings.LastIndex(s, decimalSeparator) == len(s)-2 { // Pad single digit decimal
		s += "0"
	}
	hasDecimals := strings.Contains(s, decimalSeparator)
	v := replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	if !hasDecimals {
		return n * 100, nil
	}
	return n, nil
}

func ReadFrom(r io.Reader) ([]bank.Transaction, error) {
	data, err := ioutil.ReadAll(r)
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
	var ts []bank.Transaction
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
		amount, err := parseAmount(cells[6].String())
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount: %q", cells[6].String())
		}
		t := bank.Transaction{
			Time:   time,
			Text:   cells[1].String(),
			Amount: amount,
		}
		ts = append(ts, t)
	}
	return ts, nil
}
