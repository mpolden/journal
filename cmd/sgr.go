package cmd

import "strings"

const (
	darkGray   = "\033[1;30m"
	lightRed   = "\033[1;31m"
	lightGreen = "\033[1;32m"
	reverse    = "\033[7m"
	reset      = "\033[0m"
)

var sgrTrim = strings.NewReplacer(darkGray, "", lightRed, "", lightGreen, "", reverse, "", reset, "")

type sgr struct {
	min     int64
	max     int64
	enabled bool
}

func (s *sgr) bar(n int64) string {
	var (
		bars    int64 = 30
		barSize       = (s.max - s.min) / bars
	)
	var pos int64
	if barSize > 0 {
		pos = n / barSize
	}
	var sb strings.Builder
	fill := ' '
	symbol := func(sym rune, cs ...string) {
		if s.enabled {
			for _, c := range cs {
				sb.WriteString(c)
			}
		} else {
			fill = sym
		}
	}
	for i, j, r := -bars/2, bars/2, false; i < j; i++ {
		if !r && i < 0 && i >= pos {
			symbol('-', reverse, lightGreen)
			r = true
		} else if i > 0 {
			if !r && i <= pos {
				symbol('+', reverse, lightRed)
				r = true
			} else if r && i > pos {
				symbol(' ', reset)
				r = false
			}
		}
		sb.WriteRune(fill)
		if r && (i == 0 || i == j-1) {
			symbol(' ', reset)
			r = false
		}
	}
	return sb.String()
}

func (s *sgr) color(n int64) (string, string) {
	if !s.enabled {
		return "", ""
	}
	if n == 0 {
		return darkGray, reset
	} else if n < 0 {
		return lightGreen, reset
	}
	return lightRed, reset
}
