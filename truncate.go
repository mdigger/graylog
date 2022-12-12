package graylog

import (
	"strings"
	"unicode"
)

func truncate(s string, max, min int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n\t"); i > 0 {
		s = s[:i]
	}

	runes := []rune(s)
	if len(runes) <= max {
		return s
	}

	// skip for first space or punctuation
	for i := max - 1; i >= min; i-- {
		if unicode.In(runes[i],
			unicode.Z, unicode.Pd, unicode.Pe, unicode.Pf, unicode.Po) {
			break
		}

		max = i
	}

	// skip by first no spaces or punctuations
	for i := max - 1; i >= min; i-- {
		if r := runes[i]; (r == '!' || r == '?' || r == '⁈' || r == ';') ||
			!unicode.In(r,
				unicode.Z, unicode.Pd, unicode.Pi, unicode.Ps, unicode.Po) {
			break
		}

		max = i
	}

	return string(append(runes[:max], '…')) // FIXME: escapes to heap
}
