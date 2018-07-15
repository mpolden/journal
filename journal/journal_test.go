package journal

import (
	"strings"
	"testing"
	"time"

	"github.com/mpolden/journal/record"
)

func TestJournal(t *testing.T) {
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

	rs := []record.Record{{Time: time.Now(), Text: "Transaction 1", Amount: 42}}
	if err := j.Write("1234.56.78900", rs); err != nil {
		t.Fatal(err)
	}
}
