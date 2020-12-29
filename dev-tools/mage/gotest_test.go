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

package mage

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/magefile/mage/mg"
)

func TestGoTest_CaptureOutput(t *testing.T) {
	errNonZero := "go test returned a non-zero value"
	makeArgs := func(test string) GoTestArgs {
		return GoTestArgs{
			TestName:   "asserts",
			Tags:       []string{"gotestsample"},
			Packages:   []string{"./testdata"},
			ExtraFlags: []string{"-test.run", test},
		}
	}

	tests := map[string]struct {
		args    GoTestArgs
		verbose bool
		wantErr string
		want    string
	}{
		"capture output from assert failures": {
			args:    makeArgs("TestAssertOutput"),
			wantErr: errNonZero,
			want:    wantTestAssertOutput,
		},
		"capture test log output": {
			args:    makeArgs("TestLogOutput"),
			wantErr: errNonZero,
			want:    wantTestLogOutput,
		},
		"capture panic": {
			args:    makeArgs("TestWithPanic"),
			wantErr: errNonZero,
			want:    wantTestWithPanic,
		},
		"capture wrong panic": {
			args:    makeArgs("TestWithWrongPanic"),
			wantErr: errNonZero,
			want:    wantTestWithWrongPanic,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var buf strings.Builder
			args := test.args
			args.Output = &buf
			err := GoTest(context.TODO(), args)

			oldVerboseArg := os.Getenv(mg.VerboseEnv)
			defer func() {
				os.Setenv(mg.VerboseEnv, oldVerboseArg)
			}()

			if test.verbose {
				os.Setenv(mg.VerboseEnv, "true")
			} else {
				os.Setenv(mg.VerboseEnv, "false")
			}

			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("GoTest did return an unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("GoTest was expected to return an error saying '%v'", test.wantErr)
				}

				errString := err.Error()
				if !strings.Contains(errString, test.wantErr) {
					t.Fatalf("GoTest error does not match expected error message:\nwant: '%v'\ngot: '%v'",
						test.wantErr, errString)
				}
			}

			re, err := regexp.Compile(test.want)
			if err != nil {
				t.Fatalf("Failed to compile test match regex: %v", err)
			}

			output := buf.String()
			if !re.MatchString(output) {
				t.Fatalf("GoTest output missmatch:\nwant:\n%v\n\ngot:\n%v", test.want, output)
			}
		})
	}
}

var wantTestAssertOutput = `(?sm:
=== Failed
=== FAIL: dev-tools/mage/testdata TestAssertOutput/assert_fails.*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestAssertOutput/assert_fails.*
    --- FAIL: TestAssertOutput/assert_fails .*
=== FAIL: dev-tools/mage/testdata TestAssertOutput/assert_with_message .*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestAssertOutput/assert_with_message.*
        	Messages:   	My message.*
    --- FAIL: TestAssertOutput/assert_with_message .*
=== FAIL: dev-tools/mage/testdata TestAssertOutput/assert_with_messagef .*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestAssertOutput/assert_with_messagef.*
        	Messages:   	My message with arguments: 42.*
    --- FAIL: TestAssertOutput/assert_with_messagef .*
=== FAIL: dev-tools/mage/testdata TestAssertOutput/require_fails .*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestAssertOutput/require_fails.*
    --- FAIL: TestAssertOutput/require_fails .*
=== FAIL: dev-tools/mage/testdata TestAssertOutput/require_with_message .*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestAssertOutput/require_with_message.*
        	Messages:   	My message.*
    --- FAIL: TestAssertOutput/require_with_message .*
=== FAIL: dev-tools/mage/testdata TestAssertOutput/require_with_messagef .*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestAssertOutput/require_with_messagef.*
        	Messages:   	My message with arguments: 42.*
    --- FAIL: TestAssertOutput/require_with_messagef .*
=== FAIL: dev-tools/mage/testdata TestAssertOutput/equals_map .*
    gotest_sample_test.go:\d+:.*
        	Error Trace:	gotest_sample_test.go:\d+.*
        	Error:      	Not equal:.*
\s+expected: map\[string\]interface \{\}\{"a":1, "b":true, "c":"test", "e":map\[string\]interface \{\}\{"x":"y"\}\}.*
\s+actual  : map\[string\]interface \{\}\{"a":42, "b":false, "c":"test"\}.*
\s+Diff:.*
\s+--- Expected.*
\s+\+\+\+ Actual.*
\s+-\(map\[string\]interface \{\}\) \(len=4\) \{.*
\s+- \(string\) \(len=1\) "a": \(int\) 1,.*
\s+- \(string\) \(len=1\) "b": \(bool\) true,.*
\s+- \(string\) \(len=1\) "c": \(string\) \(len=4\) "test",.*
\s+- \(string\) \(len=1\) "e": \(map\[string\]interface \{\}\) \(len=1\) \{.*
\s+-  \(string\) \(len=1\) "x": \(string\) \(len=1\) "y".*
\s+- }.*
\s+\+\(map\[string\]interface \{\}\) \(len=3\) \{.*
\s+\+ \(string\) \(len=1\) "a": \(int\) 42,.*
\s+\+ \(string\) \(len=1\) "b": \(bool\) false,.*
\s+\+ \(string\) \(len=1\) "c": \(string\) \(len=4\) "test".*
\s+\}.*
)`

var wantTestLogOutput = `(?sm:
=== Failed.*
=== FAIL: dev-tools/mage/testdata TestLogOutput/on_error.*
    gotest_sample_test.go:\d+: Log message should be printed.*
    gotest_sample_test.go:\d+: printf style log message: 42.*
    gotest_sample_test.go:\d+: Log should fail.*
    gotest_sample_test.go:\d+: Log should fail with printf style log: 23.*
    --- FAIL: TestLogOutput/on_error.*
=== FAIL: dev-tools/mage/testdata TestLogOutput/on_fatal.*
    gotest_sample_test.go:\d+: Log message should be printed.*
    gotest_sample_test.go:\d+: printf style log message: 42.*
    gotest_sample_test.go:\d+: Log should fail.*
    --- FAIL: TestLogOutput/on_fatal.*
=== FAIL: dev-tools/mage/testdata TestLogOutput/on_fatalf.*
    gotest_sample_test.go:\d+: Log message should be printed.*
    gotest_sample_test.go:\d+: printf style log message: 42.*
    gotest_sample_test.go:\d+: Log should fail with printf style log: 42.*
    --- FAIL: TestLogOutput/on_fatalf.*
=== FAIL: dev-tools/mage/testdata TestLogOutput/with_newlines.*
    gotest_sample_test.go:\d+: Log.*
        message.*
        should.*
        be.*
        printed.*
    gotest_sample_test.go:\d+: printf.*
        style.*
        log.*
        message:.*
        42.*
    gotest_sample_test.go:\d+: Log.*
        should.*
        fail.*
        with.*
        printf.*
        style.*
        log:.*
        42.*
    --- FAIL: TestLogOutput/with_newlines.*
=== FAIL: dev-tools/mage/testdata TestLogOutput.*
DONE 5 tests, 5 failures in.*
)`

var wantTestWithPanic = `(?sm:
=== FAIL: dev-tools/mage/testdata TestWithPanic.*
panic: Kaputt. \[recovered\].*
	panic: Kaputt.*
)`

var wantTestWithWrongPanic = `(?sm:
=== FAIL: dev-tools/mage/testdata TestWithWrongPanic.*
panic: Fail in goroutine after TestWithWrongPanic/setup_failing_go-routine has completed.*
.*
=== FAIL: dev-tools/mage/testdata TestWithWrongPanic/false_positive_failure.*
panic: Fail in goroutine after TestWithWrongPanic/setup_failing_go-routine has completed.*
)`
