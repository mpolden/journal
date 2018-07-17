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
name = "My account 1"

[[accounts]]
number = "1234.56.78901"
name = "My account 2"

[[groups]]
name = "Travel"
patterns = ["^Foo"]

[[groups]]
name = "Groceries"
patterns = ["^Bar", "^Baz"]

[[groups]]
name = "Misc"
ids = ["45defdf469"]

[[groups]]
name = "Other"
account = "1234.56.78901"
patterns = ["^Boo"]
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
	if want, got := int64(2), writes.Account; want != got {
		t.Errorf("want %d account writes, got %d", want, got)
	}
	if want, got := int64(1), writes.Record; want != got {
		t.Errorf("want %d record writes, got %d", want, got)
	}
}

func TestAssort(t *testing.T) {
	j := testJournal(t)
	a1 := record.Account{Number: "1234.56.78900", Name: "My account 1"}
	a2 := record.Account{Number: "1234.56.78901", Name: "My account 2"}
	rs := []record.Record{
		{Account: a1, Time: date(2018, 2, 2), Text: "Foo 1", Amount: 42}, // Travel
		{Account: a1, Time: date(2018, 1, 1), Text: "Foo 2", Amount: 42}, // Travel
		{Account: a1, Time: date(2018, 2, 2), Text: "Bar 1", Amount: 42}, // Groceries
		{Account: a1, Time: date(2018, 1, 1), Text: "Baz 1", Amount: 42}, // Groceries
		{Account: a1, Time: date(2018, 1, 1), Text: "Bar 2", Amount: 42}, // Misc (pinned)
		{Account: a1, Time: date(2018, 1, 1), Text: "Boo 1", Amount: 42}, // Unmatched (wrong account)
		{Account: a2, Time: date(2018, 1, 1), Text: "Boo 2", Amount: 42}, // Other
	}
	if _, err := j.Write(a1.Number, rs[:6]); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Write(a2.Number, rs[6:]); err != nil {
		t.Fatal(err)
	}
	records, err := j.Read("", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	rg := j.Assort(records)
	var tests = []record.Group{
		{Name: "*** UNMATCHED ***", Records: rs[5:6]},
		{Name: "Groceries", Records: rs[2:4]},
		{Name: "Misc", Records: rs[4:5]},
		{Name: "Other", Records: rs[6:]},
		{Name: "Travel", Records: rs[:2]},
	}
	for i, tt := range tests {
		if rg[i].Name != tt.Name {
			t.Errorf("#%d: want Name = %q, got %q", i, tt.Name, rg[i].Name)
		}
		if !reflect.DeepEqual(rg[i].Records, tt.Records) {
			t.Errorf("#%d: want Records = %+v, got %+v", i, tt.Records, rg[i].Records)
		}
	}
}
