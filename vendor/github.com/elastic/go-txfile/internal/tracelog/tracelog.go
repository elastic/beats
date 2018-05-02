package tracelog

import (
	"fmt"
	"os"
	"strings"
)

type Logger interface {
	Println(...interface{})
	Printf(string, ...interface{})
}

type stderrLogger struct{}

type nilLogger struct{}

func Get(selector string) Logger {
	if isEnabled(selector) {
		return (*stderrLogger)(nil)
	}
	return (*nilLogger)(nil)
}

func isEnabled(selector string) bool {
	v := os.Getenv("TRACE_SELECTOR")
	if v == "" {
		return true
	}

	selectors := strings.Split(v, ",")
	for _, sel := range selectors {
		if selector == strings.TrimSpace(sel) {
			return true
		}
	}
	return false
}

func (*nilLogger) Println(...interface{})        {}
func (*nilLogger) Printf(string, ...interface{}) {}

func (*stderrLogger) Println(vs ...interface{})          { fmt.Fprintln(os.Stderr, vs...) }
func (*stderrLogger) Printf(s string, vs ...interface{}) { fmt.Fprintf(os.Stderr, s, vs...) }
