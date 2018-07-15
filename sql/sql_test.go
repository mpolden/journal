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

func TestAddAccount(t *testing.T) {
	c := testClient()
	number := "1.33.7"
	for i := 0; i < 2; i++ {
		if err := c.AddAccount(number, "Savings"); err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	if err := c.db.Get(&count, "SELECT COUNT(*) FROM account LIMIT 1"); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("want %d accounts, got %d", 1, count)
	}
	account, err := c.GetAccount(number)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := account.Number, "1.33.7"; got != want {
		t.Errorf("want Number = %s, got %s", want, got)
	}
}

func TestAddRecords(t *testing.T) {
	c := testClient()
	number := "1.33.7"
	if err := c.AddAccount(number, "Savings"); err != nil {
		t.Fatal(err)
	}
	records := []Record{
		{Time: date(2017, 4, 20).Unix(), Text: "Transaction 4", Amount: 5678},
		{Time: date(2017, 3, 15).Unix(), Text: "Transaction 3", Amount: 24},
		{Time: date(2017, 2, 10).Unix(), Text: "Transaction 2", Amount: 1234},
		{Time: date(2017, 1, 1).Unix(), Text: "Transaction 1", Amount: 42},
		{Time: date(2017, 1, 1).Unix(), Text: "Transaction 1", Amount: 42}, // Duplicate, ignored
	}
	if err := c.AddRecords(number, records); err != nil {
		t.Fatal(err)
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
}
