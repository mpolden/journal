package cmd

import (
	"fmt"
	"time"
)

const timeLayout = "2006-01-02"

type clock struct{ now func() time.Time }

func newClock() *clock { return &clock{now: time.Now} }

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(timeLayout, s)
}

func (c *clock) monthRange(month int) (time.Time, time.Time, error) {
	now := c.now()
	if month < 1 || month > 12 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month: %d", month)
	}
	s := time.Date(now.Year(), time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	if month > int(now.Month()) {
		// Given month has not passed yet, so use previous year
		s = s.AddDate(-1, 0, 0)
	}
	u := s.AddDate(0, 1, -1)
	return s, u, nil
}

func (c *clock) timeRange(since, until string) (time.Time, time.Time, error) {
	now := c.now()
	s, err := parseTime(since)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	u, err := parseTime(until)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if s.IsZero() { // Default to start of month
		s = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if u.IsZero() {
		u = now
	}
	return s, u, nil
}
