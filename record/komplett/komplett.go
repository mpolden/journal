package komplett

import (
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/mpolden/journal/record"
)

const timeLayout = "02.01.2006"

var (
	separatorPattern = regexp.MustCompile("[.,]")
	cleanPattern     = regexp.MustCompile(`kr|NOK|"|\s+|\p{Z}+`)
)

// Reader implements a reader for Komplett-encoded (JSON) records.
type Reader struct{ rd io.Reader }

type jsonTime time.Time

type jsonAmount int64

type jsonRecord struct {
	// The JSON from their API keeps shuffling field names. Each number corresponds to a version of the format
	Time1   jsonTime   `json:"FormattedPostingDate"`
	Time2   jsonTime   `json:"TransactionDate"`
	Amount1 jsonAmount `json:"BillingAmount"`
	Amount2 jsonAmount `json:"FormattedAmount"`
	Text1   string     `json:"DisplayDescription"`
	Text2   string     `json:"MerchantName"`
	Text3   string     `json:"Description"`
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
	text := cleanPattern.ReplaceAllString(string(data), "")
	parts := separatorPattern.Split(text, -1)
	firstPart := parts[0]
	if len(parts) == 2 {
		firstPart += parts[1]
		if len(parts[1]) == 1 {
			firstPart += "0"
		}
	} else {
		firstPart += "00"
	}
	n, err := strconv.ParseInt(firstPart, 10, 64)
	if err != nil {
		return err
	}
	*a = jsonAmount(n)
	return nil
}

// NewReader returns a new reader for Komplett-encoded records.
func NewReader(rd io.Reader) *Reader {
	return &Reader{rd: rd}
}

func (r *Reader) Read() ([]record.Record, error) {
	var jrs []jsonRecord
	if err := json.NewDecoder(r.rd).Decode(&jrs); err != nil {
		return nil, err
	}
	var rs []record.Record
	for _, jr := range jrs {
		amount := jr.Amount1
		if amount == 0 {
			amount = jr.Amount2
		}
		txTime := time.Time(jr.Time1)
		if txTime.IsZero() {
			txTime = time.Time(jr.Time2)
		}
		text := jr.Text1
		if text == "" {
			text = jr.Text2
		}
		if text == "" {
			text = jr.Text3
		}
		rs = append(rs, record.Record{
			Time:   txTime,
			Text:   text,
			Amount: int64(amount),
		})
	}
	return rs, nil
}
