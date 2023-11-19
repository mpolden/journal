package morrow

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

// Reader implements a reader for Morrow-encoded (CSV) records.
type Reader struct {
	rd io.Reader
}

// NewReader returns a new reader for Morrow-encoded records.
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd: rd,
	}
}

// Read all records from the underlying reader.
func (r *Reader) Read() ([]record.Record, error) {
	buf := bufio.NewReader(r.rd)
	c := csv.NewReader(buf)
	c.FieldsPerRecord = -1 // Morrow export has an additional field which is not in the header
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
		if len(csvRecord) < 10 {
			continue
		}
		if line == 1 {
			continue // Skip header
		}
		t, err := time.Parse("02.01.2006", csvRecord[0])
		if err != nil {
			return nil, fmt.Errorf("invalid time on line %d: %q: %w", line, csvRecord[0], err)
		}
		amount, err := parseAmount(csvRecord[5])
		if err != nil {
			return nil, fmt.Errorf("invalid amount on line %d: %q: %w", line, amount, err)
		}
		text := strings.TrimSpace(csvRecord[2])
		rs = append(rs, record.Record{Time: t, Text: text, Amount: amount})
	}
	return rs, nil
}

func parseAmount(s string) (int64, error) {
	v := strings.ReplaceAll(s, ".", "")
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}
