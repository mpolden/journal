package komplett

import (
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
)

type reader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

func NewReader(rd io.Reader) record.Reader {
	return &reader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

func (r *reader) parseAmount(s string) (int64, error) {
	v := r.replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	// Fix inverted sign. This bank records purchases as positive transactions
	return n * -1, nil
}

func (r *reader) Read() ([]record.Record, error) {
	doc, err := goquery.NewDocumentFromReader(r.rd)
	if err != nil {
		return nil, err
	}
	var parseErr error
	var rs []record.Record
	doc.Find("tr.smtxt12").EachWithBreak(func(i int, s *goquery.Selection) bool {
		vs := s.Find("td")
		timeText := strings.TrimSpace(vs.Eq(0).Text())
		time, err := time.Parse("02.01.2006", timeText)
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
		t := record.Record{
			Time:   time,
			Text:   text,
			Amount: amount,
		}
		rs = append(rs, t)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return rs, nil
}
