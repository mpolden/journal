package sql

import (
	"database/sql"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const schema = `
CREATE TABLE IF NOT EXISTS account (
  id INTEGER PRIMARY KEY,
  number TEXT NOT NULL,
  description TEXT,
  CONSTRAINT number_unique UNIQUE (number)
);

CREATE TABLE IF NOT EXISTS record (
  id INTEGER PRIMARY KEY,
  account_id INTEGER NOT NULL,
  time INTEGER NOT NULL,
  text TEXT NOT NULL,
  amount INTEGER NOT NULL,
  FOREIGN KEY(account_id) REFERENCES account(id)
);

CREATE INDEX IF NOT EXISTS record_time_idx ON record (time);
`

type Client struct {
	db *sqlx.DB
	mu sync.RWMutex
}

type Account struct {
	Number      string `db:"number"`
	Description string `db:"description"`
}

type Record struct {
	Time   int64  `db:"time"`
	Text   string `db:"text"`
	Amount int64  `db:"amount"`
	Account
}

func New(filename string) (*Client, error) {
	db, err := sqlx.Connect("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	// Ensure foreign keys are enabled (defaults to off)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Client{db: db}, nil
}

func (c *Client) AddAccount(number, description string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	id := 0
	err = tx.Get(&id, "SELECT id FROM account WHERE number = $1 LIMIT 1", number)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("INSERT INTO account (number, description) VALUES ($1, $2)", number, description); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return tx.Commit()
}

func (c *Client) GetAccount(number string) (Account, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var account Account
	query := "SELECT number, description FROM account WHERE number = $1 LIMIT 1"
	if err := c.db.Get(&account, query, number); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (c *Client) AddRecords(accountNumber string, records []Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	accountID := 0
	if err := tx.Get(&accountID, "SELECT id FROM account WHERE number = $1 LIMIT 1", accountNumber); err != nil {
		return errors.Wrapf(err, "invalid account: %s", accountNumber)
	}

	query := `
SELECT COUNT(*)
FROM record
WHERE account_id = $1 AND time = $2 AND text = $3 AND amount = $4
LIMIT 1`

	insertQuery := `
INSERT INTO record (account_id, time, text, amount)
VALUES ($1, $2, $3, $4)
`

	for _, r := range records {
		count := 0
		if err := tx.Get(&count, query, accountID, r.Time, r.Text, r.Amount); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if _, err := tx.Exec(insertQuery, accountID, r.Time, r.Text, r.Amount); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (c *Client) SelectRecords(accountNumber string) ([]Record, error) {
	return c.SelectRecordsBetween(accountNumber, time.Time{}, time.Time{})
}

func (c *Client) SelectRecordsBetween(accountNumber string, since, until time.Time) ([]Record, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	query := `
SELECT number, time, text, amount FROM record
INNER JOIN account ON account_id = account.id
`
	args := []interface{}{}
	if accountNumber != "" {
		query += " WHERE number = ?"
		args = append(args, accountNumber)
	}
	if !since.IsZero() {
		query += " AND time >= ?"
		args = append(args, since.Unix())
	}
	if !until.IsZero() {
		query += " AND time <= ?"
		args = append(args, until.Unix())
	}
	query += " ORDER BY time DESC"
	var rs []Record
	if err := c.db.Select(&rs, query, args...); err != nil {
		return nil, err
	}
	return rs, nil
}
