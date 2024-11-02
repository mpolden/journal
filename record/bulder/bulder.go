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
	var (
		balanceIndex      = -1
		mainCategoryIndex = -1
		subCategoryIndex  = -1
		textIndex         = -1
		amountInIndex     = -1
		amountOutIndex    = -1
		typeIndex         = -1
	)
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
			for i, field := range csvRecord {
				switch field {
				case "Balanse":
					balanceIndex = i
				case "Hovedkategori":
					mainCategoryIndex = i
				case "Underkategori":
					subCategoryIndex = i
				case "Type":
					typeIndex = i
				case "Tekst", "Tekst/KID":
					textIndex = i
				case "Inn på konto":
					amountInIndex = i
				case "Ut fra konto":
					amountOutIndex = i
				}
			}
			continue // Skip header
		}
		t, err := time.Parse("2006-01-02", csvRecord[0])
		if err != nil {
			return nil, fmt.Errorf("invalid time on line %d: %q: %w", line, csvRecord[0], err)
		}
		amountValue := ""
		if amountIn := csvRecord[amountInIndex]; amountIn != "" {
			amountValue = amountIn
		} else if amountOut := csvRecord[amountOutIndex]; amountOut != "" {
			amountValue = amountOut
		}
		amount, err := parseAmount(amountValue)
		if err != nil {
			return nil, fmt.Errorf("invalid amount on line %d: %q: %w", line, amountValue, err)
		}
		var balance int64
		if balanceIndex > -1 {
			v := csvRecord[balanceIndex]
			balance, err = parseAmount(v)
			if err != nil {
				return nil, fmt.Errorf("invalid balance on line %d: %q: %w", line, v, err)
			}
		}
		var text strings.Builder
		paymentType := csvRecord[typeIndex]
		paymentText := csvRecord[textIndex]
		text.WriteString(paymentType)
		if paymentText != "" {
			text.WriteString(",")
			text.WriteString(paymentText)
		}
		category := csvRecord[mainCategoryIndex]
		subCategory := csvRecord[subCategoryIndex]
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
