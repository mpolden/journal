package record

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/timeutil"
	"github.com/pkg/errors"
)

const (
	decimalSeparator  = "."
	thousandSeparator = ","
	byteOrderMark     = '\uFEFF'
)

// Reader is the interface for record readers.
type Reader interface {
	Read() ([]Record, error)
}

// An Account identifies a finanical account.
type Account struct {
	Number string
	Name   string
}

// A Record is a record of a finanical transaction.
type Record struct {
	Account Account
	Time    time.Time
	Text    string
	Amount  int64
}

// A Group is a list of recordes grouped together under a common name.
type Group struct {
	Name          string
	Records       []Record
	MonthlyBudget int64
}

// A Period stores record groups for specific moment in time.
type Period struct {
	Time   time.Time
	Groups []Group
}

type reader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func NewReader(rd io.Reader) Reader {
	return &reader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

// ID returns a shortened SHA-1 hash of the fields in this record
func (r *Record) ID() string {
	var buf bytes.Buffer
	buf.WriteString(r.Account.Number)
	buf.WriteString(r.Time.Format("2006-01-02"))
	buf.WriteString(r.Text)
	buf.WriteString(strconv.FormatInt(r.Amount, 10))
	sum := sha1.Sum(buf.Bytes())
	return fmt.Sprintf("%x", sum)[:10]
}

// Sum returns the total sum of all records in the group
func (g *Group) Sum() int64 {
	var sum int64
	for _, r := range g.Records {
		sum += r.Amount
	}
	return sum
}

// Budget returns the budget for this group. The budget is adjusted to the record period
func (g *Group) Budget() int64 {
	var start, end time.Time
	for _, r := range g.Records {
		if start.IsZero() || r.Time.Before(start) {
			start = r.Time
		}
		if r.Time.After(end) {
			end = r.Time
		}
	}
	return g.MonthlyBudget * max(1, timeutil.MonthsBetween(start, end))
}

// Balance returns the difference between the budget of this group and its sum. The balance is adjusted to the record
// period.
func (g *Group) Balance() int64 {
	return g.Budget() - g.Sum()
}

// AssortFunc uses groupFn to assort records into groups.
func AssortFunc(records []Record, assortFn func(Record) (Group, bool)) []Group {
	m := make(map[string]Group)
	for _, r := range records {
		target, ok := assortFn(r)
		if !ok {
			continue
		}
		g, ok := m[target.Name]
		if !ok {
			g = target
		}
		g.Records = append(g.Records, r)
		m[target.Name] = g
	}
	var gs []Group
	for _, g := range m {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i].Name < gs[j].Name })
	return gs
}

// AssortPeriodFunc assorts records into groups grouped by timeFn.
func AssortPeriodFunc(records []Record, timeFn func(time.Time) time.Time, assortFn func(Record) (Group, bool)) []Period {
	m := make(map[time.Time][]Record)
	for _, r := range records {
		key := timeFn(r.Time)
		m[key] = append(m[key], r)
	}
	var ps []Period
	for t, rs := range m {
		ps = append(ps, Period{Time: t, Groups: AssortFunc(rs, assortFn)})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Time.After(ps[j].Time) })
	return ps
}

func (d *reader) parseAmount(s string) (int64, error) {
	v := d.replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Read all records from the reader.
func (r *reader) Read() ([]Record, error) {
	buf := bufio.NewReader(r.rd)
	// Peek at the first rune see if the file starts with a byte order mark
	rune, _, err := buf.ReadRune()
	if err != nil {
		return nil, err
	}
	if rune != byteOrderMark {
		if err := buf.UnreadRune(); err != nil {
			return nil, err
		}
	}
	c := csv.NewReader(buf)
	c.Comma = ';'
	var rs []Record
	line := 0
	for {
		record, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if len(record) < 4 {
			continue
		}
		t, err := time.Parse("02.01.2006", record[0])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid time found on line %d: %q", line, record[0])
		}
		text := record[2]
		amount, err := r.parseAmount(record[3])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount found on line %d: %q", line, record[3])
		}
		rs = append(rs, Record{Time: t, Text: text, Amount: amount})
	}
	return rs, nil
}
