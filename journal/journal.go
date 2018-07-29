package journal

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/sql"
)

// Account represents a financial account.
type Account struct {
	Number string
	Name   string
}

// Group represents a group configuration which decides how records should be assorted into groups.
type Group struct {
	Name     string
	Account  string
	Budget   int64
	Budgets  [12]int64
	Patterns []string
	patterns []*regexp.Regexp
	IDs      []string
	Discard  bool
}

// Config represents a journal's configuration.
type Config struct {
	Database     string
	Comma        string
	DefaultGroup string
	Accounts     []Account
	Groups       []Group
}

// Journal implements a journal of financial records.
type Journal struct {
	accounts     []Account
	groups       []Group
	db           *sql.Client
	Comma        string
	DefaultGroup string
}

// Writes represents statistics of a journal's updates.
type Writes struct {
	Account int64
	Record  int64
}

func (c *Config) load() error {
	if len(c.Database) == 0 {
		return fmt.Errorf("invalid database path: %q", c.Database)
	}
	if c.Database[0] == '~' {
		if len(c.Database) > 1 && c.Database[1] != '/' {
			return fmt.Errorf("invalid database path: %q", c.Database)
		}
		user, err := user.Current()
		if err != nil {
			return err
		}
		c.Database = filepath.Join(user.HomeDir, c.Database[1:])
	}
	for _, a := range c.Accounts {
		if len(a.Number) == 0 {
			return fmt.Errorf("invalid account number: %q", a.Number)
		}
	}
	for i, g := range c.Groups {
		if len(g.Name) == 0 {
			return fmt.Errorf("invalid group name: %q", g.Name)
		}
		for _, pattern := range g.Patterns {
			if len(pattern) == 0 {
				return fmt.Errorf("group: %q: invalid pattern: %q", g.Name, pattern)
			}
			p, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			c.Groups[i].patterns = append(c.Groups[i].patterns, p)
		}

	}
	return nil
}

func readConfig(r io.Reader) (Config, error) {
	var conf Config
	_, err := toml.DecodeReader(r, &conf)
	return conf, err
}

// FromConfig creates a new journal from a configuration file located at name.
func FromConfig(name string) (*Journal, error) {
	if name == "~/.journalrc" {
		home := os.Getenv("HOME")
		name = filepath.Join(home, ".journalrc")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	conf, err := readConfig(f)
	if err != nil {
		return nil, err
	}
	return New(conf)
}

// New creates a new journal from the given configuration.
func New(conf Config) (*Journal, error) {
	if err := conf.load(); err != nil {
		return nil, err
	}
	db, err := sql.New(conf.Database)
	if err != nil {
		return nil, err
	}
	comma := conf.Comma
	if comma == "" {
		comma = "."
	}
	defaultGroup := conf.DefaultGroup
	if defaultGroup == "" {
		defaultGroup = "*** UNMATCHED ***"
	}
	return &Journal{
		db:           db,
		accounts:     conf.Accounts,
		groups:       conf.Groups,
		Comma:        comma,
		DefaultGroup: defaultGroup,
	}, nil
}

// FormatAmount formats number n as a financial amount.
func (j *Journal) FormatAmount(n int64) string {
	i := n / 100
	f := n % 100
	var buf bytes.Buffer
	if f < 0 {
		f *= -1
		if i == 0 {
			buf.WriteRune('-')
		}
	}
	buf.WriteString(strconv.FormatInt(i, 10))
	buf.WriteString(j.Comma)
	buf.WriteString(fmt.Sprintf("%02d", f))
	return buf.String()
}

// Export writes periods to writer w using CSV-encoding. The timeLayout defines the format of time fields.
func (j *Journal) Export(w io.Writer, periods []record.Period, timeLayout string) error {
	csv := csv.NewWriter(w)
	for _, p := range periods {
		for _, rg := range p.Groups {
			r := []string{p.Time.Format(timeLayout), rg.Name, j.FormatAmount(rg.Sum())}
			if err := csv.Write(r); err != nil {
				return err
			}
		}
	}
	csv.Flush()
	return csv.Error()
}

func (j *Journal) writeAccounts() (int64, error) {
	as := make([]sql.Account, len(j.accounts))
	for i, a := range j.accounts {
		as[i] = sql.Account{Number: a.Number, Name: a.Name}
	}
	return j.db.AddAccounts(as)
}

// Write writes records for accountNumber into the journal.
func (j *Journal) Write(accountNumber string, records []record.Record) (Writes, error) {
	var writes Writes
	n, err := j.writeAccounts()
	if err != nil {
		return writes, err
	}
	writes.Account = n
	rs := make([]sql.Record, len(records))
	for i, r := range records {
		rs[i] = sql.Record{Time: r.Time.Unix(), Text: r.Text, Amount: r.Amount}
	}
	n, err = j.db.AddRecords(accountNumber, rs)
	writes.Record = n
	return writes, err
}

// Read reads records for accountNumber between the times since and until from the journal.
func (j *Journal) Read(accountNumber string, since, until time.Time) ([]record.Record, error) {
	rs, err := j.db.SelectRecordsBetween(accountNumber, since, until)
	if err != nil {
		return nil, err
	}
	records := make([]record.Record, len(rs))
	for i, r := range rs {
		records[i] = record.Record{
			Account: record.Account{Number: r.Account.Number, Name: r.Account.Name},
			Time:    time.Unix(r.Time, 0).UTC(),
			Text:    r.Text,
			Amount:  r.Amount,
		}
	}
	return records, nil
}

// Assort assorts records into groups using this journal's configuration.
func (j *Journal) Assort(records []record.Record) []record.Group {
	return record.AssortFunc(records, j.findGroup)
}

// AssortPeriod assorts record groups into time periods using timeFn.
func (j *Journal) AssortPeriod(records []record.Record, timeFn func(time.Time) time.Time) []record.Period {
	return record.AssortPeriodFunc(records, timeFn, j.findGroup)
}

func (j *Journal) findGroup(r record.Record) *record.Group {
	for _, g := range j.groups {
		if g.Account != "" && g.Account != r.Account.Number {
			continue
		}
		for _, id := range g.IDs {
			if r.ID() == id {
				if g.Discard {
					return nil
				}
				rg := record.NewGroup(g.Name, record.Budget{
					Default: g.Budget,
					Months:  g.Budgets,
				})
				return &rg
			}
		}
	}
	for _, g := range j.groups {
		if g.Account != "" && g.Account != r.Account.Number {
			continue
		}
		for _, p := range g.patterns {
			if p.MatchString(r.Text) {
				if g.Discard {
					return nil
				}
				rg := record.NewGroup(g.Name, record.Budget{
					Default: g.Budget,
					Months:  g.Budgets,
				})
				return &rg
			}
		}
	}
	return &record.Group{Name: j.DefaultGroup}
}
