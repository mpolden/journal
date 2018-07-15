package journal

import (
	"strings"
	"testing"
	"time"

	"github.com/mpolden/journal/record"
)

func testJournal(t *testing.T) *Journal {
	tomlConf := `
Database = ":memory:"
[[accounts]]
number = "1234.56.78900"
description = "My account"

[[groups]]
name = "Travel"
patterns = ["^Foo"]
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
