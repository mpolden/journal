package journal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/sql"
)

type Account struct {
	Number string
	Name   string
}

type RecordGroup struct {
	Name    string
	Records []record.Record
}

type Group struct {
	Name     string
	Patterns []string
	patterns []*regexp.Regexp
	IDs      []string
}

type Config struct {
	Database string
	Accounts []Account
	Groups   []Group
}

type Journal struct {
	accounts []Account
	groups   []Group
	db       *sql.Client
}

type Writes struct {
	Account int64
	Record  int64
}

func (c *Config) load() error {
	if len(c.Database) == 0 {
		return fmt.Errorf("invalid path to database: %q", c.Database)
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
				return fmt.Errorf("invalid pattern: %q", pattern)
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

func New(conf Config) (*Journal, error) {
	if err := conf.load(); err != nil {
		return nil, err
	}
	db, err := sql.New(conf.Database)
	if err != nil {
		return nil, err
	}
	return &Journal{
		db:       db,
		accounts: conf.Accounts,
		groups:   conf.Groups,
	}, nil
}

func (j *Journal) writeAccounts() (int64, error) {
	as := make([]sql.Account, len(j.accounts))
	for i, a := range j.accounts {
		as[i] = sql.Account{Number: a.Number, Name: a.Name}
	}
	return j.db.AddAccounts(as)
}

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

func (j *Journal) Group(rs []record.Record) []RecordGroup {
	groups := make(map[string][]record.Record)
	for _, r := range rs {
		g := j.findGroup(r)
		groups[g.Name] = append(groups[g.Name], r)
	}
	var rgs []RecordGroup
	for name, rs := range groups {
		rgs = append(rgs, RecordGroup{Name: name, Records: rs})
	}
	sort.Slice(rgs, func(i, j int) bool { return rgs[i].Name < rgs[j].Name })
	return rgs
}

func (j *Journal) findGroup(r record.Record) *Group {
	for i, g := range j.groups {
		for _, id := range g.IDs {
			if r.ID() == id {
				return &j.groups[i]
			}
		}
	}
	for i, g := range j.groups {
		for _, p := range g.patterns {
			if p.MatchString(r.Text) {
				return &j.groups[i]
			}
		}
	}
	return &Group{Name: "*** UNMATCHED ***"}
}

func (rg *RecordGroup) Sum() int64 {
	var sum int64
	for _, r := range rg.Records {
		sum += r.Amount
	}
	return sum
}
