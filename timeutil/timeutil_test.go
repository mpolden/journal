package timeutil

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestMonthsBetween(t *testing.T) {
	var tests = []struct {
		a      time.Time
		b      time.Time
		months int64
	}{
		{date(2018, 1, 1), date(2017, 1, 1), 0}, // Start date is before end date
		{date(2018, 1, 1), date(2018, 1, 1), 0},
		{date(2018, 1, 31), date(2018, 2, 1), 1}, // Days are ignored
		{date(2018, 1, 1), date(2018, 3, 1), 2},
		{date(2017, 1, 1), date(2018, 3, 1), 14}, // Overlapping year
	}
	for i, tt := range tests {
		if want, got := tt.months, MonthsBetween(tt.a, tt.b); want != got {
			t.Errorf("#%d: want %d, got %d", i, want, got)
		}
	}
}
