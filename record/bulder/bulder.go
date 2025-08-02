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

const (
	balanceField     = "Balanse"
	categoryField    = "Hovedkategori"
	dateField        = "Dato"
	amountField      = "Beløp"
	inflowField      = "Inn på konto"
	outflowField     = "Ut fra konto"
	subCategoryField = "Underkategori"
	textField        = "Tekst"
	textFieldLegacy  = "Tekst/KID"
	typeField        = "Type"
)

var requiredFields = []string{
	categoryField,
	dateField,
	subCategoryField,
	textField,
	typeField,
}

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

func findAmount(indices map[string]int, record []string) (string, error) {
	for _, field := range []string{amountField, inflowField, outflowField} {
		i, ok := indices[field]
		if !ok {
			continue
		}
		if record[i] == "" {
			continue
		}
		return record[i], nil
	}
	return "", fmt.Errorf("no amount field found")
}

// Read all records from the underlying reader.
func (r *Reader) Read() ([]record.Record, error) {
	buf := bufio.NewReader(r.rd)
	c := csv.NewReader(buf)
	c.Comma = ';'
	var rs []record.Record
	line := 0
	indices := map[string]int{}
	for {
		cr, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if len(cr) < 10 {
			continue
		}
		// Determine field index from header
		if line == 1 {
			for i, field := range cr {
				if field == textFieldLegacy {
					field = textField
				}
				indices[field] = i
			}
			for _, field := range requiredFields {
				if _, ok := indices[field]; !ok {
					return nil, fmt.Errorf("required field %q not found in header", field)
				}
			}
			continue
		}
		date := cr[indices[dateField]]
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, fmt.Errorf("invalid time on line %d: %q: %w", line, date, err)
		}
		amountValue, err := findAmount(indices, cr)
		if err != nil {
			return nil, fmt.Errorf("no amount on line %d: %w", line, err)
		}
		amount, err := parseAmount(amountValue)
		if err != nil {
			return nil, fmt.Errorf("invalid amount on line %d: %q: %w", line, amountValue, err)
		}
		var balance int64
		if i, ok := indices[balanceField]; ok {
			v := cr[i]
			balance, err = parseAmount(cr[i])
			if err != nil {
				return nil, fmt.Errorf("invalid balance on line %d: %q: %w", line, v, err)
			}
		}
		var text strings.Builder
		paymentType := cr[indices[typeField]]
		paymentText := cr[indices[textField]]
		text.WriteString(paymentType)
		if paymentText != "" {
			text.WriteString(",")
			text.WriteString(paymentText)
		}
		category := cr[indices[categoryField]]
		subCategory := cr[indices[subCategoryField]]
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
	v := strings.Map(func(r rune) rune {
		if r == ',' || r == '.' {
			return -1
		}
		return r
	}, s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}
