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
	oldFormat := false
	for {
		csvRecord, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if len(csvRecord) < 12 {
			continue
		}
		if line == 1 {
			oldFormat = csvRecord[3] == "Balanse"
			continue // Skip header
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
		if balanceValue := csvRecord[3]; oldFormat && balanceValue != "" {
			balance, err = parseAmount(balanceValue)
			if err != nil {
				return nil, fmt.Errorf("invalid balance on line %d: %q: %w", line, balanceValue, err)
			}
		}
		indexOffset := 0
		if oldFormat {
			indexOffset = 1
		}
		var text strings.Builder
		paymentType := csvRecord[7+indexOffset]
		paymentText := csvRecord[8+indexOffset]
		text.WriteString(paymentType)
		text.WriteString(",")
		text.WriteString(paymentText)
		category := csvRecord[10]
		subCategory := csvRecord[11]
		if category != "" {
			text.WriteString(",")
			text.WriteString(category)
		}
		if subCategory != "" {
			text.WriteString(",")
			text.WriteString(subCategory)
		}
		rs = append(rs, record.Record{Time: t, Text: text.String(), Amount: amount, Balance: balance})
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
