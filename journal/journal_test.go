package journal

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/record/komplett"
	"github.com/mpolden/journal/record/norwegian"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func testJournal(t *testing.T) *Journal {
	tomlConf := `
Database = ":memory:"
DefaultGroup = "* no group *"

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

[[groups]]
name = "Unimportant"
patterns = ["^Spam"]
discard = true
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

func TestReaderFrom(t *testing.T) {
	r := strings.NewReader("")
	var tests = []struct {
		name     string
		filename string
		impl     string
	}{
		{"csv", "", "default"},
		{"norwegian", "", "norwegian"},
		{"komplett", "", "komplett"},
		{"komplettsparing", "", "komplett-sparing"},
		{"auto", "foo.csv", "default"},
		{"auto", "foo.xlsx", "norwegian"},
		{"auto", "foo.html", "komplett"},
		{"auto", "foo.json", "komplett-sparing"},
	}
	for i, tt := range tests {
		rr, err := readerFrom(r, tt.name, tt.filename)
		if err != nil {
			t.Fatal(err)
		}
		switch tt.impl {
		case "default":
			if _, ok := rr.(record.Reader); !ok {
				t.Errorf("#%d: want record.Reader, got %T", i, rr)
			}
		case "norwegian":
			if _, ok := rr.(*norwegian.Reader); !ok {
				t.Errorf("#%d: want norwegian.Reader, got %T", i, rr)
			}
		case "komplett", "komplett-sparing":
			kr, ok := rr.(*komplett.Reader)
			if !ok {
				t.Errorf("#%d: want komplett.Reader, got %T", i, rr)
			}
			want := tt.impl == "komplett-sparing"
			if kr.JSON != want {
				t.Errorf("#%d: want JSON = %t, got %t", i, want, kr.JSON)
			}
		}
	}
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

func TestFormatAmount(t *testing.T) {
	j := testJournal(t)
	var tests = []struct {
		amount int64
		comma  string
		out    string
	}{
		{-1053, ",", "-10,53"},
		{-153, ",", "-1,53"},
		{-15, ",", "-0,15"},
		{-1, ",", "-0,01"},
		{0, ",", "0,00"},
		{1, ",", "0,01"},
		{15, ",", "0,15"},
		{153, ",", "1,53"},
		{1053, ",", "10,53"},
		{1053, ".", "10.53"},
	}
	for i, tt := range tests {
		j.Comma = tt.comma
		got := j.FormatAmount(tt.amount)
		if got != tt.out {
			t.Errorf("#%d: want %q, got %q", i, tt.out, got)
		}
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
		{Account: a1, Time: date(2018, 1, 1), Text: "Spam", Amount: 42},  // No group (discarded)
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
	rgs := j.Assort(records)
	var tests = []record.Group{
		{Name: "* no group *", Records: rs[5:6]},
		{Name: "Groceries", Records: rs[2:4]},
		{Name: "Misc", Records: rs[4:5]},
		{Name: "Other", Records: rs[6:7]},
		{Name: "Travel", Records: rs[:2]},
	}
	if want, got := len(tests), len(rgs); want != got {
		t.Errorf("want %d groups, got %d", want, got)
	}
	for i, tt := range tests {
		if rgs[i].Name != tt.Name {
			t.Errorf("#%d: want Name = %q, got %q", i, tt.Name, rgs[i].Name)
		}
		if !reflect.DeepEqual(rgs[i].Records, tt.Records) {
			t.Errorf("#%d: want Records = %+v, got %+v", i, tt.Records, rgs[i].Records)
		}
	}
}
