package komplett

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mpolden/journal/bank"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestReadFrom(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(wd, "testdata", "test.html")

	f, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var readFrom bank.ReadFromFunc = ReadFrom

	ts, err := readFrom(f)
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		t      time.Time
		text   string
		amount int64
	}{
		{date(2017, 5, 20), "Transaction 4", -4230},
		{date(2017, 4, 20), "Transaction 3", 4233},
		{date(2017, 3, 10), "Transaction 2", -4233},
		{date(2017, 2, 1), "Transaction 1", 133700},
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
