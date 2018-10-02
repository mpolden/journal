package komplett

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mpolden/journal/record"
	"github.com/pkg/errors"
)

const (
	decimalSeparator  = "."
	thousandSeparator = " "
	timeLayout        = "02.01.2006"
)

// Reader implements a reader for Komplett-encoded (HTML or JSON) records.
type Reader struct {
	rd       io.Reader
	replacer *strings.Replacer
	JSON     bool
}

type jsonTime time.Time

type jsonAmount int64

type jsonRecord struct {
	Time   jsonTime   `json:"FormattedPostingDate"`
	Amount jsonAmount `json:"BillingAmount"`
	Text   string     `json:"DisplayDescription"`
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
	s := string(data)
	if strings.Contains(s, decimalSeparator) {
		s = strings.Replace(s, decimalSeparator, "", -1)
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

func (r *Reader) parseAmount(s string) (int64, error) {
	v := r.replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	// Fix inverted sign. This bank records purchases as positive transactions
	return n * -1, nil
}

func (r *Reader) Read() ([]record.Record, error) {
	if r.JSON {
		return r.readJSON()
	}
	return r.readHTML()
}

func (r *Reader) readHTML() ([]record.Record, error) {
	doc, err := goquery.NewDocumentFromReader(r.rd)
	if err != nil {
		return nil, err
	}
	var parseErr error
	var rs []record.Record
	doc.Find("tr.smtxt12").EachWithBreak(func(i int, s *goquery.Selection) bool {
		vs := s.Find("td")
		timeText := strings.TrimSpace(vs.Eq(0).Text())
		time, err := time.Parse(timeLayout, timeText)
		if err != nil {
			parseErr = errors.Wrapf(err, "invalid time: %q", timeText)
			return false
		}
		text := strings.TrimSpace(vs.Eq(1).Text())
		amountText := strings.TrimSpace(s.Find("td span.credit-amount").Text())
		amount, err := r.parseAmount(amountText)
		if err != nil {
			parseErr = errors.Wrapf(err, "invalid amount: %q", amountText)
			return false
		}
		rs = append(rs, record.Record{
			Time:   time,
			Text:   text,
			Amount: amount,
		})
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return rs, nil
}

func (r *Reader) readJSON() ([]record.Record, error) {
	var jrs []jsonRecord
	if err := json.NewDecoder(r.rd).Decode(&jrs); err != nil {
		return nil, err
	}
	var rs []record.Record
	for _, jr := range jrs {
		rs = append(rs, record.Record{
			Time:   time.Time(jr.Time),
			Text:   jr.Text,
			Amount: int64(jr.Amount),
		})
	}
	return rs, nil
}
