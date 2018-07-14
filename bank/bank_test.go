package bank

import (
	"strings"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestParse(t *testing.T) {
	lines := `"01.02.2017";"01.02.2017";"Transaction 1";"1.337,00";"1.337,00";"";""
"10.03.2017";"10.03.2017";"Transaction 2";"-42,00";"1.295,00";"";""
"20.04.2017";"20.04.2017";"Transaction 3";"42,00";"1.337,00";"";""
`
	ts, err := Parse(strings.NewReader(lines))
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		t      time.Time
		text   string
		amount int64
	}{
		{date(2017, 2, 1), "Transaction 1", 133700},
		{date(2017, 3, 10), "Transaction 2", -4200},
		{date(2017, 4, 20), "Transaction 3", 4200},
	}
	if len(ts) != len(tests) {
		t.Fatalf("want %d records, got %d", len(tests), len(ts))
	}
	for i, tt := range tests {
		if !ts[i].Time.Equal(tt.t) {
			t.Errorf("#%d: want Time = %s, got %s", i, tt.t, ts[i].Time)
		}
		if ts[i].Text != tt.text {
			t.Errorf("#%d: want Text = %s, got %s", i, tt.text, ts[i].Text)
		}
		if ts[i].Amount != tt.amount {
			t.Errorf("#%d: want Amount = %d, got %d", i, tt.amount, ts[i].Amount)
		}
	}
}
