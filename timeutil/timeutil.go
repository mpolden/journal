package timeutil

import "time"

func MonthsBetween(a, b time.Time) int64 {
	var months int64
	month := a.Month()
	for a.Before(b) {
		a = a.AddDate(0, 1, 0)
		next := a.Month()
		if next != month {
			months++
		}
		month = next
	}
	return months
}
