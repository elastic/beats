package errx

import (
	"bufio"
	"fmt"
	"strings"
)

func pad(buf *strings.Builder, pattern string) bool {
	if buf.Len() == 0 {
		return false
	}

	buf.WriteString(pattern)
	return true
}

func putStr(buf *strings.Builder, s string) bool {
	if s == "" {
		return false
	}
	pad(buf, ": ")
	buf.WriteString(s)
	return true
}

func putSubErr(b *strings.Builder, sep string, err error, verbose bool) bool {
	if err == nil {
		return false
	}

	var s string
	if verbose {
		s = fmt.Sprintf("%+v", err)
	} else {
		s = fmt.Sprintf("%v", err)
	}

	if s == "" {
		return false
	}

	pad(b, sep)

	// iterate lines
	r := strings.NewReader(s)
	scanner := bufio.NewScanner(r)
	first := true
	for scanner.Scan() {
		if !first {
			pad(b, sep)
		} else {
			first = false
		}

		b.WriteString(scanner.Text())
	}
	return true
}
