package komplett

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func testFile(t *testing.T, name string) *os.File {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(wd, "testdata", name)

	f, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestReadHTML(t *testing.T) {
	f := testFile(t, "test.html")
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
		{date(2017, 5, 20), "Transaction 4", -4230},
		{date(2017, 4, 20), "Transaction 3", 4233},
		{date(2017, 3, 10), "Transaction 2", -4233},
		{date(2017, 2, 1), "Transaction 1", 133700},
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

func TestReadJSON(t *testing.T) {
	f := testFile(t, "test.json")
	defer f.Close()

	r := NewReader(f)
	r.JSON = true
	rs, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		t      time.Time
		text   string
		amount int64
	}{
		{date(2017, 9, 1), "Innskudd / Ekstra avdrag", 4242},
		{date(2017, 8, 1), "Innskudd / Ekstra avdrag", 133700},
		{date(2018, 9, 1), "Varekjøp", -50000},
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
