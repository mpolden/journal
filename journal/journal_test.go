package journal

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mpolden/journal/record"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func testJournal(t *testing.T) *Journal {
	tomlConf := `
Database = ":memory:"
[[accounts]]
number = "1234.56.78900"
name = "My account"

[[groups]]
name = "Travel"
patterns = ["^Foo"]

[[groups]]
name = "Groceries"
patterns = ["^Bar", "^Baz"]
`
	conf, err := readConfig(strings.NewReader(tomlConf))
	if err != nil {
		t.Fatal(err)
	}

	j, err := New(conf)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestWrite(t *testing.T) {
	j := testJournal(t)
	rs := []record.Record{{Time: time.Now(), Text: "Transaction 1", Amount: 42}}
	writes, err := j.Write("1234.56.78900", rs)
	if err != nil {
		t.Fatal(err)
	}
	if want, got := int64(1), writes.Account; want != got {
		t.Errorf("want %d account writes, got %d", want, got)
	}
	if want, got := int64(1), writes.Record; want != got {
		t.Errorf("want %d record writes, got %d", want, got)
	}
}

func TestGroup(t *testing.T) {
	j := testJournal(t)
	account := record.Account{Number: "1234.56.78900", Name: "My account"}
	rs := []record.Record{
		{Account: account, Time: date(2018, 2, 2), Text: "Foo 1", Amount: 42},
		{Account: account, Time: date(2018, 1, 1), Text: "Foo 2", Amount: 42},
		{Account: account, Time: date(2018, 2, 2), Text: "Bar 1", Amount: 42},
		{Account: account, Time: date(2018, 1, 1), Text: "Baz 1", Amount: 42},
	}
	_, err := j.Write(account.Number, rs)
	if err != nil {
		t.Fatal(err)
	}
	records, err := j.Read(account.Number, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	rg := j.Group(records)

	var tests = []RecordGroup{
		{"Groceries", rs[2:]},
		{"Travel", rs[:2]},
	}
	for i, tt := range tests {
		if rg[i].Name != tt.Name {
			t.Errorf("want Name = %q, got %q", tt.Name, rg[i].Name)
		}
		if !reflect.DeepEqual(rg[i].Records, tt.Records) {
			t.Errorf("want Records = %+v, got %+v", tt.Records, rg[i].Records)
		}
	}
}
