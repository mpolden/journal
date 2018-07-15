package sql

import (
	"testing"
	"time"
)

func testClient() *Client {
	c, err := New(":memory:")
	if err != nil {
		panic(err)
	}
	return c
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestAddAccounts(t *testing.T) {
	c := testClient()
	accounts := []Account{
		{Number: "1.2.3", Description: "Account 1"},
		{Number: "4.5.6", Description: "Account 2"},
		{Number: "7.8.9", Description: "Account 3"},
		{Number: "1.2.3", Description: "Account 1"}, // Duplicate
	}
	n, err := c.AddAccounts(accounts)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(3); n != want {
		t.Errorf("want %d accounts, got %d", want, n)
	}
	as, err := c.SelectAccounts("")
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range as {
		if accounts[i].Number != a.Number {
			t.Errorf("want Number = %s, got %s", accounts[i].Number, a.Number)
		}
		if accounts[i].Description != a.Description {
			t.Errorf("want Number = %s, got %s", accounts[i].Number, a.Number)
		}
	}
}

func TestAddRecords(t *testing.T) {
	c := testClient()
	number := "1.2.3"
	as := []Account{{Number: number, Description: "Savings"}}
	if _, err := c.AddAccounts(as); err != nil {
		t.Fatal(err)
	}
	records := []Record{
		{Time: date(2017, 4, 20).Unix(), Text: "Transaction 4", Amount: 5678},
		{Time: date(2017, 3, 15).Unix(), Text: "Transaction 3", Amount: 24},
		{Time: date(2017, 2, 10).Unix(), Text: "Transaction 2", Amount: 1234},
		{Time: date(2017, 1, 1).Unix(), Text: "Transaction 1", Amount: 42},
		{Time: date(2017, 1, 1).Unix(), Text: "Transaction 1", Amount: 42}, // Duplicate, ignored
	}
	n, err := c.AddRecords(number, records)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(4); n != want {
		t.Errorf("want %d records, got %d", want, n)
	}
	rs, err := c.SelectRecords(number)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rs), len(records)-1; got != want {
		t.Errorf("want len = %d, got %d", want, got)
	}
	for i, r := range rs {
		if records[i].Time != r.Time {
			t.Errorf("want Time = %d, got %d", records[i].Time, r.Time)
		}
		if records[i].Text != r.Text {
			t.Errorf("want Text = %s, got %s", records[i].Text, r.Text)
		}
		if records[i].Amount != r.Amount {
			t.Errorf("want Amount = %d, got %d", records[i].Amount, r.Amount)
		}
	}

	// Select records in date range
	since := date(2017, 2, 10)
	until := date(2017, 3, 15)
	rs, err = c.SelectRecordsBetween(number, since, until)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rs), 2; got != want {
		t.Errorf("want len = %d, got %d", want, got)
	}
	if got, want := rs[0].Time, until.Unix(); got != want {
		t.Errorf("want Time = %d, got %d", want, got)
	}
	if got, want := rs[1].Time, since.Unix(); got != want {
		t.Errorf("want Time = %d, got %d", want, got)
	}

	// Select all records
	rs, err = c.SelectRecords("")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rs), len(records)-1; got != want {
		t.Errorf("want len = %d, got %d", want, got)
	}
}
