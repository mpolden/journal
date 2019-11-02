package komplett

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/record"
)

const (
	decimalSeparator  = "."
	thousandSeparator = " "
	timeLayout        = "02.01.2006"
	withdrawalPrefix  = "kr -"
)

// Reader implements a reader for Komplett-encoded (JSON) records.
type Reader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

type jsonTime time.Time

type jsonAmount int64

type jsonRecord struct {
	Time            jsonTime   `json:"FormattedPostingDate"`
	BillingAmount   jsonAmount `json:"BillingAmount"`
	Amount          jsonAmount `json:"Amount"`
	FormattedAmount string     `json:"FormattedAmount"`
	Text            string     `json:"DisplayDescription"`
}

func (t *jsonTime) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	tt, err := time.Parse(timeLayout, s)
	if err != nil {
		return err
	}
	*t = jsonTime(tt)
	return nil
}

func (a *jsonAmount) UnmarshalJSON(data []byte) error {
	parts := strings.Split(string(data), decimalSeparator)
	s := parts[0]
	if len(parts) == 2 {
		s += parts[1]
		if len(parts[1]) == 1 {
			s += "0"
		}
	} else {
		s += "00"
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*a = jsonAmount(n)
	return nil
}

// NewReader returns a new reader for Komplett-encoded records.
func NewReader(rd io.Reader) *Reader {
	return &Reader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

func (r *Reader) Read() ([]record.Record, error) {
	var jrs []jsonRecord
	if err := json.NewDecoder(r.rd).Decode(&jrs); err != nil {
		return nil, err
	}
	var rs []record.Record
	for _, jr := range jrs {
		amount := jr.BillingAmount
		if amount == 0 { // New format
			amount = jr.Amount
			// New format does not indicate whether transaction amount is positive or negative, so we guess
			// based on the formatted field.
			if strings.HasPrefix(jr.FormattedAmount, withdrawalPrefix) {
				amount = -amount
			}
		}
		rs = append(rs, record.Record{
			Time:   time.Time(jr.Time),
			Text:   jr.Text,
			Amount: int64(amount),
		})
	}
	return rs, nil
}
