package bank

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	decimalSeparator  = "."
	thousandSeparator = ","
)

var replacer = strings.NewReplacer(decimalSeparator, "", thousandSeparator, "")

// Parser parses transactions from given reader
type Parser func(io.Reader) ([]Transaction, error)

type Transaction struct {
	Time   time.Time
	Text   string
	Amount int64
}

func (t *Transaction) StringAmount() string {
	s := strconv.FormatInt(t.Amount, 10)
	off := len(s) - 2
	return s[:off] + "," + s[off:]
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%s\t%s\t%s", t.Time.Format("2006-01-02"), t.Text, t.StringAmount())
}

func parseAmount(s string) (int64, error) {
	v := replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func Parse(r io.Reader) ([]Transaction, error) {
	c := csv.NewReader(r)
	c.Comma = ';'
	var ts []Transaction
	line := 0
	for {
		record, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if len(record) < 4 {
			continue
		}
		t, err := time.Parse("02.01.2006", record[0])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid time found on line %d: %q", line, record[0])
		}
		text := record[2]
		amount, err := parseAmount(record[3])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount found on line %d: %q", line, record[3])
		}
		ts = append(ts, Transaction{Time: t, Text: text, Amount: amount})
	}
	return ts, nil
}
