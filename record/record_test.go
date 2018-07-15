package record

import (
	"strings"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func testReadFrom(lines string, t *testing.T) {
	r := NewReader(strings.NewReader(lines))
	rs, err := r.Read()
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
	if len(rs) != len(tests) {
		t.Fatalf("want %d records, got %d", len(tests), len(rs))
	}
	for i, tt := range tests {
		if !rs[i].Time.Equal(tt.t) {
			t.Errorf("#%d: want Time = %s, got %s", i, tt.t, rs[i].Time)
		}
		if rs[i].Text != tt.text {
			t.Errorf("#%d: want Text = %s, got %s", i, tt.text, rs[i].Text)
		}
		if rs[i].Amount != tt.amount {
			t.Errorf("#%d: want Amount = %d, got %d", i, tt.amount, rs[i].Amount)
		}
	}
}

func TestReadFrom(t *testing.T) {
	lines := `"01.02.2017";"01.02.2017";"Transaction 1";"1.337,00";"1.337,00";"";""
"10.03.2017";"10.03.2017";"Transaction 2";"-42,00";"1.295,00";"";""
"20.04.2017";"20.04.2017";"Transaction 3";"42,00";"1.337,00";"";""
`
	testReadFrom(lines, t)
	testReadFrom(string(byteOrderMark)+lines, t)
}
