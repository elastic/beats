package queuetest

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

var debug bool
var printLog bool

type TestLogger struct {
	t *testing.T
}

func init() {
	flag.BoolVar(&debug, "debug", false, "enable test debug log")
	flag.BoolVar(&printLog, "debug-print", false, "print test log messages right away")
}

type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func withLogOutput(fn func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {

		stderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer r.Close()

			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Text()
				t.Log(line)
				if printLog {
					stderr.WriteString(line)
					stderr.WriteString("\n")
				}
			}
		}()

		os.Stderr = w
		defer func() {
			os.Stderr = stderr
			w.Close()
			wg.Wait()
		}()

		level := logp.InfoLevel
		if debug {
			level = logp.DebugLevel
		}
		logp.DevelopmentSetup(logp.WithLevel(level))
		fn(t)
	}
}

// NewTestLogger creates a new logger interface,
// logging via t.Log/t.Logf. If `-debug` is given on command
// line, debug logs will be included.
// Run tests with `-debug-print`, to print log output to console right away.
// This guarantees logs are still written if the test logs are not printed due
// to a panic in the test itself.
//
// Capturing log output using the TestLogger, will make the
// log output correctly display with test test being run.
func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t}
}

func (l *TestLogger) Debug(vs ...interface{}) {
	if debug {
		l.t.Log(vs...)
		print(vs)
	}
}

func (l *TestLogger) Info(vs ...interface{}) {
	l.t.Log(vs...)
	print(vs)
}

func (l *TestLogger) Err(vs ...interface{}) {
	l.t.Error(vs...)
	print(vs)
}

func (l *TestLogger) Debugf(format string, v ...interface{}) {
	if debug {
		l.t.Logf(format, v...)
		printf(format, v)
	}
}

func (l *TestLogger) Infof(format string, v ...interface{}) {
	l.t.Logf(format, v...)
	printf(format, v)
}
func (l *TestLogger) Errf(format string, v ...interface{}) {
	l.t.Errorf(format, v...)
	printf(format, v)
}

func print(vs []interface{}) {
	if printLog {
		fmt.Println(vs...)
	}
}

func printf(format string, vs []interface{}) {
	if printLog {
		fmt.Printf(format, vs...)
		if format[len(format)-1] != '\n' {
			fmt.Println("")
		}
	}
}
