package sql

import (
	"database/sql"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite database driver
	"github.com/pkg/errors"
)

const schema = `
CREATE TABLE IF NOT EXISTS account (
  id INTEGER PRIMARY KEY,
  number TEXT NOT NULL,
  name TEXT NOT NULL,
  CONSTRAINT number_unique UNIQUE (number)
);

CREATE TABLE IF NOT EXISTS record (
  id INTEGER PRIMARY KEY,
  account_id INTEGER NOT NULL,
  time INTEGER NOT NULL,
  text TEXT NOT NULL,
  amount INTEGER NOT NULL,
  CONSTRAINT record_unique UNIQUE(account_id, time, text, amount),
  FOREIGN KEY(account_id) REFERENCES account(id)
);

CREATE INDEX IF NOT EXISTS record_time_idx ON record (time);
`

// Client implements a client for a SQLite database.
type Client struct {
	db *sqlx.DB
	mu sync.RWMutex
}

// Account represents a financial account.
type Account struct {
	Number  string `db:"number"`
	Name    string `db:"name"`
	Records int64  `db:"records"`
}

// Record represents a single financial record.
type Record struct {
	Time   int64  `db:"time"`
	Text   string `db:"text"`
	Amount int64  `db:"amount"`
	Account
}

// New creates a new database client for given filename.
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

func rowsAffected(result sql.Result) int64 {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// SQLite implements RowsAffected
		panic(err)
	}
	return rowsAffected
}

// AddAccounts writes accounts to the database and returns the number of changed rows.
func (c *Client) AddAccounts(accounts []Account) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var rows int64
	for _, a := range accounts {
		count := 0
		if err := tx.Get(&count, "SELECT COUNT(*) FROM account WHERE number = ? LIMIT 1", a.Number); err != nil {
			return 0, err
		}
		if count > 0 {
			continue
		}
		res, err := tx.Exec("INSERT INTO account (number, name) VALUES ($1, $2)", a.Number, a.Name)
		if err != nil {
			return 0, err
		}
		rows += rowsAffected(res)
	}
	return rows, tx.Commit()
}

// SelectAccounts reads accounts from the database matching accountNumber. If accountNumber is an empty string, all
// accounts are returned.
func (c *Client) SelectAccounts(accountNumber string) ([]Account, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var as []Account
	query := `
SELECT number, name, COUNT(record.id) AS records
FROM account
LEFT JOIN record ON account.id = record.account_id
`
	args := []interface{}{}
	if accountNumber != "" {
		query += " WHERE number = ?"
		args = append(args, accountNumber)
	}
	query += " GROUP BY number ORDER BY number ASC"
	err := c.db.Select(&as, query, args...)
	return as, err
}

// AddRecords writes new records to belonging to accountNumber to the database, and returns the number of changed rows.
// Any duplicate records are ignored.
func (c *Client) AddRecords(accountNumber string, records []Record) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	accountID := 0
	if err := tx.Get(&accountID, "SELECT id FROM account WHERE number = $1 LIMIT 1", accountNumber); err != nil {
		return 0, errors.Wrapf(err, "invalid account: %s", accountNumber)
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

	var rows int64
	for _, r := range records {
		count := 0
		if err := tx.Get(&count, query, accountID, r.Time, r.Text, r.Amount); err != nil {
			return 0, err
		}
		if count > 0 {
			continue
		}
		res, err := tx.Exec(insertQuery, accountID, r.Time, r.Text, r.Amount)
		if err != nil {
			return 0, err
		}
		rows += rowsAffected(res)
	}

	return rows, tx.Commit()
}

// SelectRecords reads all records belonging to given accountNumber.
func (c *Client) SelectRecords(accountNumber string) ([]Record, error) {
	return c.SelectRecordsBetween(accountNumber, time.Time{}, time.Time{})
}

// SelectRecordsBetween reads all records belonging to given accountNumber, and occurring between the times since and
// until.
func (c *Client) SelectRecordsBetween(accountNumber string, since, until time.Time) ([]Record, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	query := `
SELECT name, number, time, text, amount
FROM record
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
