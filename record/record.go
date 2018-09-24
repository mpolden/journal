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

	"github.com/pkg/errors"
)

// Field identifies a record field.
type Field int

const (
	decimalSeparator  = "."
	thousandSeparator = ","
	byteOrderMark     = '\uFEFF'

	// NameField is the text field of a record.
	NameField Field = iota

	// GroupField is the name field of a record group.
	GroupField

	// TimeField is the time field of a record.
	TimeField

	// SumField is the sum field of a record or record group.
	SumField
)

// Reader is the interface for record readers.
type Reader interface {
	Read() ([]Record, error)
}

// A Budget represents a budget for a group of records.
type Budget struct {
	Default int64
	Months  [12]int64
}

// An Account identifies a finanical account.
type Account struct {
	Number  string
	Name    string
	Records int64
}

// A Record is a record of a finanical transaction.
type Record struct {
	Account Account
	Time    time.Time
	Text    string
	Amount  int64
	Balance int64
}

// A Group is a list of records grouped together under a common name.
type Group struct {
	Name    string
	Records []Record
	budget  Budget
}

// A Range represents a record time range.
type Range struct {
	Since time.Time
	Until time.Time
}

// A Period is a list of record groups occurring at a specific time.
type Period struct {
	Time   time.Time
	Groups []Group
}

type reader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

// NewGroup returns a new group with name and budget.
func NewGroup(name string, budget Budget) Group { return Group{Name: name, budget: budget} }

// NewReader returns a new reader for CSV-encoded records.
func NewReader(rd io.Reader) Reader {
	return &reader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

// Month returns the budget for month.
func (b *Budget) Month(m time.Month) int64 {
	monthly := false
	for _, n := range b.Months {
		if n != 0 {
			monthly = true
			break
		}
	}
	if monthly {
		return b.Months[m-1]
	}
	return b.Default
}

// ID returns a shortened SHA-1 hash of the fields in this record.
func (r *Record) ID() string {
	var buf bytes.Buffer
	buf.WriteString(r.Account.Number)
	buf.WriteString(r.Time.Format("2006-01-02"))
	buf.WriteString(r.Text)
	buf.WriteString(strconv.FormatInt(r.Amount, 10))
	// Balance is considered optional, only include it in the hash if non-zero
	if r.Balance != 0 {
		buf.WriteString(strconv.FormatInt(r.Balance, 10))
	}
	sum := sha1.Sum(buf.Bytes())
	return fmt.Sprintf("%x", sum)[:10]
}

func (r *Range) months() []time.Month {
	var months []time.Month
	t := r.Since
	for !t.After(r.Until) {
		months = append(months, t.Month())
		t = t.AddDate(0, 1, 0)
	}
	return months
}

// Sum returns the total sum of all records in the group.
func (g *Group) Sum() int64 {
	var sum int64
	for _, r := range g.Records {
		sum += r.Amount
	}
	return sum
}

// Budget returns the budget for this group. The budget is adjusted to the number of months in range r.
func (g *Group) Budget(r Range) int64 {
	var budget int64
	for _, m := range r.months() {
		budget += g.budget.Month(m)
	}
	return budget
}

// Balance returns the difference between the budget of this group and its sum. Balance adjusts the budget using
// range r in the same way that Budget does.
func (g *Group) Balance(r Range) int64 { return g.Budget(r) - g.Sum() }

// MaxBalance returns the highest balance of the groups in gs. MaxBalance adjusts the budget using range r in the
// same way that Budget does.
func MaxBalance(gs []Group, r Range) int64 {
	var max int64
	for _, rg := range gs {
		if b := rg.Balance(r); b > max {
			max = b
		}
	}
	return max
}

// MinBalance returns the lowest balance of the groups in gs. MinBalance adjusts the budget using range r in the
// same way that Budget does.
func MinBalance(gs []Group, r Range) int64 {
	min := MaxBalance(gs, r)
	for _, rg := range gs {
		if b := rg.Balance(r); b < min {
			min = b
		}
	}
	return min
}

// AssortFunc uses groupFn to assort records into groups.
func AssortFunc(records []Record, assortFn func(Record) *Group) []Group {
	m := make(map[string]Group)
	for _, r := range records {
		target := assortFn(r)
		if target == nil {
			continue
		}
		g, ok := m[target.Name]
		if !ok {
			g = *target
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
func AssortPeriodFunc(records []Record, timeFn func(time.Time) time.Time, assortFn func(Record) *Group) []Period {
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

// Sort sorts a list of records by field.
func Sort(rs []Record, field Field) {
	sort.Slice(rs, func(i, j int) bool {
		switch field {
		case NameField:
			return rs[i].Text < (rs[j].Text)
		case TimeField:
			return rs[i].Time.Before(rs[j].Time)
		case SumField:
			return rs[i].Amount < rs[j].Amount
		}
		return false
	})
}

// SortGroup sorts a list of record groups by field.
func SortGroup(gs []Group, field Field) {
	sort.Slice(gs, func(i, j int) bool {
		switch field {
		case GroupField:
			return gs[i].Name < gs[j].Name
		case SumField:
			return gs[i].Sum() < gs[j].Sum()
		}
		return false
	})
}

func (r *reader) parseAmount(s string) (int64, error) {
	v := r.replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Read all records from the underlying reader.
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
		if len(record) < 5 {
			continue
		}
		t, err := time.Parse("02.01.2006", record[0])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid time on line %d: %q", line, record[0])
		}
		text := record[2]
		amount, err := r.parseAmount(record[3])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount on line %d: %q", line, record[3])
		}
		var balance int64
		if record[4] != "" {
			balance, err = r.parseAmount(record[4])
			if err != nil {
				return nil, errors.Wrapf(err, "invalid balance on line %d: %q", line, record[4])
			}
		}
		rs = append(rs, Record{Time: t, Text: text, Amount: amount, Balance: balance})
	}
	return rs, nil
}
