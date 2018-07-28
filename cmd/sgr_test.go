package cmd

import "testing"

func TestSGRBar(t *testing.T) {
	var tests = []struct {
		n     int64
		min   int64
		max   int64
		color bool
		out   string
	}{
		{4000, 0, 10000, true, "                \x1b[7m\x1b[1;31m            \x1b[0m  "},
		// Number is out of bounds
		{10000, 0, 5000, true, "                \x1b[7m\x1b[1;31m              \x1b[0m"},
		{-5000, -10000, 10000, true, "        \x1b[7m\x1b[1;32m        \x1b[0m              "},
		// Max > min
		{-2000, -5000, -10000, true, "                              "},
		{0, 0, 1000, true, "                              "},
		{4000, 0, 10000, false, "                ++++++++++++  "},
		{-5000, -10000, 10000, false, "        --------              "},
		{0, 0, 1000, false, "                              "},
	}
	for i, tt := range tests {
		b := sgr{min: tt.min, max: tt.max, enabled: tt.color}
		if got := b.bar(tt.n); got != tt.out {
			t.Errorf("#%d: want '%q', got '%q'", i, tt.out, got)
		}
	}
}
