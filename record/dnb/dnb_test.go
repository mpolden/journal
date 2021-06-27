package dnb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRead(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(wd, "testdata", "test.xlsx")
	f, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := NewReader(f)
	rs, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		t      time.Time
		text   string
		amount int64
	}{
		{time.Date(2020, 6, 25, 0, 0, 0, 0, time.UTC), "Transaction 1", -119990},
		{time.Date(2020, 6, 26, 0, 0, 0, 0, time.UTC), "Transaction 2", -59995},
		{time.Date(2020, 6, 27, 0, 0, 0, 0, time.UTC), "Transaction 3", 70000},
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
