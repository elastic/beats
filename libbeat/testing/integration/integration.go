// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"testing"
)

const (
	expectErrMsg = "cannot set expectations once the test started"
)

// BeatTest describes all operations involved
// in integration testing of a Beat
type BeatTest interface {
	// Start the integration test
	//
	// The test runs until all the expectations are met (unless `ExpectStop` is used) or context was canceled or the Beat exits on its own.
	Start(context.Context) BeatTest

	// Wait until the test is over.
	//
	// `PrintOutput` might be helpful for debugging after calling this function.
	Wait()

	// ExpectStart sets an expectation that the Beat will report that it started.
	ExpectStart() BeatTest

	// ExpectStop sets an expectation that the Beat will exit by itself.
	// The process exit code will be checked against the given value.
	//
	// User controls the timeout by passing the context in `Start`.
	//
	// All the output expectations would still work as usual, however,
	// satisfying all expectations would not stop the Beat.
	ExpectStop(exitCode int) BeatTest

	// ExpectOutput registers an output watch for the given substrings.
	//
	// Every future output line produced by the Beat will be checked
	// if it contains one of the given strings.
	//
	// If given multiple strings, they get checked in order:
	// The first substring must be found first, then second, etc.
	//
	// For `AND` behavior use this function multiple times.
	//
	// This function should be used before `Start` because it's
	// inspecting only the new output lines.
	ExpectOutput(...string) BeatTest

	// ExpectOutputRegex registers an output watch for the given regular expression..
	//
	// Every future output line produced by the Beat will be matched
	// against the given regular expression.
	//
	// If given multiple expressions, they get checked in order.
	// The first expression must match first, then second, etc.
	//
	// For `AND` behavior use this function multiple times.
	//
	// This function should be used before `Start` because it's
	// inspecting only new outputs.
	ExpectOutputRegex(...*regexp.Regexp) BeatTest

	// PrintOutput prints last `limit` lines of the output
	//
	// It might be handy for inspecting the output in case of a failure.
	// Use `limit=-1` to print the entire output (strongly discouraged).
	//
	// JSON lines of the output are formatted.
	PrintOutput(lineCount int)

	// PrintExpectations prints all currently set expectations
	PrintExpectations()

	// WithReportOptions sets the reporting options for the test.
	WithReportOptions(ReportOptions) BeatTest
}

// ReportOptions describes all reporting options
type ReportOptions struct {
	// PrintExpectationsBeforeStart if set to `true`, all the defined
	// expectations will be printed before the test starts.
	//
	// Use it only if you have a manageable amount of expectations that
	// would be readable in the output.
	PrintExpectationsBeforeStart bool

	// PrintLinesOnFail defines how many lines of the Beat output
	// the test should print in case of failure (default 0).
	//
	// It uses `PrintOutput`, see its documentation for details.
	PrintLinesOnFail int

	// PrintConfig defines if the test prints out the entire configuration file
	// in case of failure.
	PrintConfigOnFail bool
}

// BeatTestOptions describes all options to run the test
type BeatTestOptions = RunBeatOptions

// NewBeatTest creates a new integration test for a Beat.
func NewBeatTest(t *testing.T, opts BeatTestOptions) BeatTest {
	test := &beatTest{
		t:    t,
		opts: opts,
	}

	return test
}

type beatTest struct {
	t                *testing.T
	opts             BeatTestOptions
	reportOpts       ReportOptions
	expectations     []OutputWatcher
	expectedExitCode *int
	beat             *RunningBeat
	mtx              sync.Mutex
}

// Start implements the BeatTest interface.
func (b *beatTest) Start(ctx context.Context) BeatTest {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.beat != nil {
		b.t.Fatal("test cannot be startd multiple times")
		return b
	}
	watcher := NewOverallWatcher(b.expectations)
	b.t.Logf("running %s integration test...", b.opts.Beatname)
	if b.reportOpts.PrintExpectationsBeforeStart {
		b.printExpectations()
	}
	b.beat = RunBeat(ctx, b.t, b.opts, watcher)

	return b
}

// Wait implements the BeatTest interface.
func (b *beatTest) Wait() {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat == nil {
		b.t.Fatal("test must start first before calling wait on it")
		return
	}

	err := b.beat.Wait()
	exitErr := &exec.ExitError{}
	if !errors.As(err, &exitErr) {
		b.t.Fatalf("unexpected error when stopping %s: %s", b.opts.Beatname, err)
		return
	}

	exitCode := 0
	if err != nil {
		exitCode = exitErr.ExitCode()
	}
	b.t.Logf("%s stopped, exit code %d", b.opts.Beatname, exitCode)

	if b.expectedExitCode != nil && exitCode != *b.expectedExitCode {
		b.t.Cleanup(func() {
			b.t.Logf("expected exit code %d, actual %d", *b.expectedExitCode, exitCode)
		})

		b.t.Fail()
	}

	if b.beat.watcher != nil {
		b.t.Cleanup(func() {
			b.t.Logf("\n\nExpectations are not met:\n\n%s\n\n", b.beat.watcher.String())
			if b.reportOpts.PrintLinesOnFail != 0 {
				b.PrintOutput(b.reportOpts.PrintLinesOnFail)
			}
			if b.reportOpts.PrintConfigOnFail {
				b.PrintConfig()
			}
		})
		b.t.Fail()
	}
}

// ExpectOutput implements the BeatTest interface.
func (b *beatTest) ExpectOutput(lines ...string) BeatTest {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat != nil {
		b.t.Fatal(expectErrMsg)
		return b
	}

	if len(lines) == 0 {
		return b
	}

	if len(lines) == 1 {
		l := escapeJSONCharacters(lines[0])
		b.expectations = append(b.expectations, NewStringWatcher(l))
		return b
	}

	watchers := make([]OutputWatcher, 0, len(lines))
	for _, l := range lines {
		escaped := escapeJSONCharacters(l)
		watchers = append(watchers, NewStringWatcher(escaped))
	}
	b.expectations = append(b.expectations, NewInOrderWatcher(watchers))
	return b
}

// ExpectOutputRegex implements the BeatTest interface.
func (b *beatTest) ExpectOutputRegex(exprs ...*regexp.Regexp) BeatTest {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat != nil {
		b.t.Fatal(expectErrMsg)
		return b
	}

	if len(exprs) == 0 {
		return b
	}

	if len(exprs) == 1 {
		b.expectations = append(b.expectations, NewRegexpWatcher(exprs[0]))
		return b
	}

	watchers := make([]OutputWatcher, 0, len(exprs))
	for _, e := range exprs {
		watchers = append(watchers, NewRegexpWatcher(e))
	}
	b.expectations = append(b.expectations, NewInOrderWatcher(watchers))

	return b
}

// ExpectStart implements the BeatTest interface.
func (b *beatTest) ExpectStart() BeatTest {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat != nil {
		b.t.Fatal(expectErrMsg)
		return b
	}

	expectedLine := fmt.Sprintf("%s start running.", b.opts.Beatname)
	b.expectations = append(b.expectations, NewStringWatcher(expectedLine))
	return b
}

// ExpectStop implements the BeatTest interface.
func (b *beatTest) ExpectStop(exitCode int) BeatTest {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat != nil {
		b.t.Fatal(expectErrMsg)
		return b
	}

	b.opts.KeepRunning = true
	b.expectedExitCode = &exitCode
	return b
}

// PrintOutput implements the BeatTest interface.
func (b *beatTest) PrintOutput(lineCount int) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat == nil {
		return
	}

	b.t.Logf("\n\nLast %d lines of the output:\n\n%s\n\n", lineCount, b.beat.CollectOutput(lineCount))
}

// PrintConfig prints the entire configuration file the Beat test ran with
func (b *beatTest) PrintConfig() {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.beat == nil {
		return
	}

	b.t.Logf("\n\nConfig file %s ran with:\n\n%s\n\n", b.opts.Beatname, b.opts.Config)
}

// WithReportOptions implements the BeatTest interface.
func (b *beatTest) WithReportOptions(opts ReportOptions) BeatTest {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.reportOpts = opts
	return b
}

// PrintExpectations implements the BeatTest interface.
func (b *beatTest) PrintExpectations() {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.printExpectations()
}

// lock-free, so it can be used inside a lock
func (b *beatTest) printExpectations() {
	overall := NewOverallWatcher(b.expectations)
	b.t.Logf("set expectations:\n%s", overall)
	if b.expectedExitCode != nil {
		b.t.Logf("\nprocess is expected to exit with code %d\n\n", *b.expectedExitCode)
	} else {
		b.t.Log("\nprocess is expected to be killed once expectations are met\n\n")
	}
}

// we know that we're going to inpect the JSON output from the Beat
// so we must take care of the escaped characters,
// e.g. backslashes in paths on Windows.
func escapeJSONCharacters(s string) string {
	bytes, _ := json.Marshal(s)
	// trimming quote marks
	return string(bytes[1 : len(bytes)-1])
}
