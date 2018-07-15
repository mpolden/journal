package journal

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/sql"
)

type Account struct {
	Number      string
	Description string
}

type Config struct {
	Database string
	Accounts []Account
}

type Journal struct {
	accounts []Account
	db       *sql.Client
}

func (c *Config) validate() error {
	if len(c.Database) == 0 {
		return fmt.Errorf("invalid path to database: %q", c.Database)
	}
	for _, a := range c.Accounts {
		if len(a.Number) == 0 {
			return fmt.Errorf("invalid account number: %q", a.Number)
		}
	}
	return nil
}

func readConfig(r io.Reader) (Config, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	err = json.Unmarshal(data, &cfg)
	return cfg, err
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
	cfg, err := readConfig(f)
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

func New(cfg Config) (*Journal, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	db, err := sql.New(cfg.Database)
	if err != nil {
		return nil, err
	}
	return &Journal{db: db, accounts: cfg.Accounts}, nil
}

func (j *Journal) writeAccounts() error {
	for _, a := range j.accounts {
		if err := j.db.AddAccount(a.Number, a.Description); err != nil {
			return err
		}
	}
	return nil
}

func (j *Journal) Write(accountNumber string, records []record.Record) error {
	if err := j.writeAccounts(); err != nil {
		return err
	}
	rs := make([]sql.Record, len(records))
	for i, r := range records {
		rs[i] = sql.Record{Time: r.Time.Unix(), Text: r.Text, Amount: r.Amount}
	}
	return j.db.AddRecords(accountNumber, rs)
}
