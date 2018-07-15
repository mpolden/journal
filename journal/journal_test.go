package journal

import (
	"strings"
	"testing"
	"time"

	"github.com/mpolden/journal/record"
)

func TestJournal(t *testing.T) {
	jsonConfig := `
{
  "Database": ":memory:",
  "Accounts": [
    {
      "Number": "1234.56.78900",
      "Description": "foo"
    }
  ]
}
`
	cfg, err := readConfig(strings.NewReader(jsonConfig))
	if err != nil {
		t.Fatal(err)
	}

	j, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	rs := []record.Record{{Time: time.Now(), Text: "Transaction 1", Amount: 42}}
	if err := j.Write("1234.56.78900", rs); err != nil {
		t.Fatal(err)
	}
}
