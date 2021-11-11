package bulder

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/record"
)

// Reader implements a reader for Bulder-encoded (CSV) records.
type Reader struct {
	rd io.Reader
}

// NewReader returns a new reader for Bulder-encoded records.
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd: rd,
	}
}

// Read all records from the underlying reader.
func (r *Reader) Read() ([]record.Record, error) {
	buf := bufio.NewReader(r.rd)
	c := csv.NewReader(buf)
	c.Comma = ';'
	var rs []record.Record
	line := 0
	for {
		csvRecord, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if line == 1 {
			continue // Skip header
		}
		if len(csvRecord) < 12 {
			continue
		}
		t, err := time.Parse("2006-01-02", csvRecord[0])
		if err != nil {
			return nil, fmt.Errorf("invalid time on line %d: %q: %w", line, csvRecord[0], err)
		}
		amountValue := csvRecord[1]
		if amountValue == "" {
			amountValue = csvRecord[2]
		}
		amount, err := parseAmount(amountValue)
		if err != nil {
			return nil, fmt.Errorf("invalid amount on line %d: %q: %w", line, amountValue, err)
		}
		var balance int64
		balanceValue := csvRecord[3]
		if balanceValue != "" {
			balance, err = parseAmount(balanceValue)
			if err != nil {
				return nil, fmt.Errorf("invalid balance on line %d: %q: %w", line, balanceValue, err)
			}
		}
		text := csvRecord[9]
		rs = append(rs, record.Record{Time: t, Text: text, Amount: amount, Balance: balance})
	}
	return rs, nil
}

func parseAmount(s string) (int64, error) {
	v := strings.ReplaceAll(s, ",", "")
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}
