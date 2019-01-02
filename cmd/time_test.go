package cmd

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestParseTime(t *testing.T) {
	var tests = []struct {
		in  string
		out time.Time
		err bool
	}{
		{"", time.Time{}, false},
		{"foobar", time.Time{}, true},
		{"2018-01-01", date(2018, 1, 1), false},
	}
	for _, tt := range tests {
		p, err := parseTime(tt.in)
		if tt.err == (err == nil) {
			t.Errorf("got unexpected error for %q: %q", tt.in, err)
		}
		if !tt.out.Equal(p) {
			t.Errorf("got %q, want %q", p, tt.out)
		}
	}
}

func TestMonthRange(t *testing.T) {
	testClock := &clock{now: func() time.Time { return date(2019, 1, 1) }}
	var tests = []struct {
		in  int
		s   time.Time
		u   time.Time
		err bool
	}{
		{-1, time.Time{}, time.Time{}, true},
		{0, time.Time{}, time.Time{}, true},
		{13, time.Time{}, time.Time{}, true},
		{10, date(2018, 10, 1), date(2018, 10, 31), false},
		{11, date(2018, 11, 1), date(2018, 11, 30), false},
		{12, date(2018, 12, 1), date(2018, 12, 31), false},
	}
	for i, tt := range tests {
		s, u, err := testClock.monthRange(tt.in)
		if tt.err == (err == nil) {
			t.Errorf("#%d: got unexpected error for %q: %s", i, tt.in, err)
		}
		if !tt.s.Equal(s) {
			t.Errorf("#%d: got s=%s, want %s", i, s, tt.s)
		}
		if !tt.u.Equal(u) {
			t.Errorf("#%d: got u=%s, want %s", i, u, tt.u)
		}
		if got, want := s.Location(), time.UTC; want != got {
			t.Errorf("#%d: got s.Location=%s, want %s", i, got, want)
		}
		if got, want := u.Location(), time.UTC; want != got {
			t.Errorf("#%d: got u.Location=%s, want %s", i, got, want)
		}
	}
}

func TestTimeRange(t *testing.T) {
	testClock := &clock{now: func() time.Time { return date(2018, 12, 15) }}
	var tests = []struct {
		since string
		until string
		s     time.Time
		u     time.Time
	}{
		{"", "", date(2018, 12, 1), date(2018, 12, 15)},
		{"2018-01-01", "", date(2018, 1, 1), date(2018, 12, 15)},
		{"2018-02-10", "2018-02-15", date(2018, 2, 10), date(2018, 2, 15)},
	}
	for _, tt := range tests {
		s, u, err := testClock.timeRange(tt.since, tt.until)
		if err != nil {
			t.Errorf("got error for s=%s u=%s: %s", tt.since, tt.until, err)
		}
		if !tt.s.Equal(s) {
			t.Errorf("got s=%s, want %s", s, tt.s)
		}
		if !tt.u.Equal(u) {
			t.Errorf("got u=%s, want %s", u, tt.u)
		}
		if got, want := s.Location(), time.UTC; want != got {
			t.Errorf("got s.Location=%s, want %s", got, want)
		}
		if got, want := u.Location(), time.UTC; want != got {
			t.Errorf("got u.Location=%s, want %s", got, want)
		}
	}
}
