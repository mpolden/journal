package komplett

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mpolden/journal/bank"
	"github.com/pkg/errors"
)

const (
	decimalSeparator  = "."
	thousandSeparator = " "
)

var replacer = strings.NewReplacer(decimalSeparator, "", thousandSeparator, "")

func parseAmount(s string) (int64, error) {
	v := replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	// Fix inverted sign. This bank records purchases as positive transactions
	return n * -1, nil
}

func Parse(r io.Reader) ([]bank.Transaction, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	var parseErr error
	var ts []bank.Transaction
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
		amount, err := parseAmount(amountText)
		if err != nil {
			parseErr = errors.Wrapf(err, "invalid amount: %q", amountText)
			return false
		}
		t := bank.Transaction{
			Time:   time,
			Text:   text,
			Amount: amount,
		}
		ts = append(ts, t)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return ts, nil
}
