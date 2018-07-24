package timeutil

import "time"

func MonthsBetween(t, u time.Time) int64 {
	var months int64
	month := t.Month()
	for t.Before(u) {
		t = t.AddDate(0, 1, 0)
		next := t.Month()
		if next != month {
			months++
		}
		month = next
	}
	return months
}
