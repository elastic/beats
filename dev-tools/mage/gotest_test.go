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
	"time"

	"github.com/magefile/mage/mg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const envGoTestHelper = "GOTEST_WANT_HELPER"

var gotestHelperMode = os.Getenv(envGoTestHelper) == "1"

// TestGoTest_CaptureOutput runs different `go test` scenarios via `GoTest` and
// captures the stderr and stdout output of the test run. The output is then
// validated using a regular expression.
//
// For each scenario a GoTest helper test is defined and a regular expression
// that the test output must match. The naming convention for scenario X is:
//   - TestGoTest_Helper_<X>: the test function to be executed
//   - wantTest<X>: regular expression the output must match.
//
// TestGoTest_CaptureOutput sets the `GOTEST_WANT_HELPER` environment variable when it executes the tests.
// each test helper must check if it is driven by this function or not:
//
//         func TestGoTest_Helper_X(t *testing.T) {
//           if !gotestHelperMode {
//             return
//           }
//
//           // sample test
//         }
//
func TestGoTest_CaptureOutput(t *testing.T) {
	errNonZero := "go test returned a non-zero value"
	makeArgs := func(test string) GoTestArgs {
		return GoTestArgs{
			TestName:   "asserts",
			Packages:   []string{"."},
			Env:        map[string]string{envGoTestHelper: "1"},
			ExtraFlags: []string{"-test.run", test},
		}
	}

	tests := map[string]struct {
		args    GoTestArgs
		verbose bool
		wantErr string
		want    string
	}{
		"passing test without output": {
			args:    makeArgs("TestGoTest_Helper_OK"),
			verbose: true,
			want:    wantTestOK,
		},
		"capture output from assert failures": {
			args:    makeArgs("TestGoTest_Helper_AssertOutput"),
			wantErr: errNonZero,
			want:    wantTestAssertOutput,
		},
		"capture test log output": {
			args:    makeArgs("TestGoTest_Helper_LogOutput"),
			wantErr: errNonZero,
			want:    wantTestLogOutput,
		},
		"capture panic": {
			args:    makeArgs("TestGoTest_Helper_WithPanic"),
			wantErr: errNonZero,
			want:    wantTestWithPanic,
		},
		"capture wrong panic": {
			args:    makeArgs("TestGoTest_Helper_WithWrongPanic"),
			wantErr: errNonZero,
			want:    wantTestWithWrongPanic,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			oldVerboseArg := os.Getenv(mg.VerboseEnv)
			defer func() {
				os.Setenv(mg.VerboseEnv, oldVerboseArg)
			}()

			if test.verbose {
				os.Setenv(mg.VerboseEnv, "true")
			} else {
				os.Setenv(mg.VerboseEnv, "false")
			}

			var buf strings.Builder
			args := test.args
			args.Output = &buf
			err := GoTest(context.TODO(), args)

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
				t.Fatalf("GoTest output mismatch:\nwant:\n%v\n\ngot:\n%v", test.want, output)
			}
		})
	}
}

func TestGoTest_Helper_OK(t *testing.T) {
	if !gotestHelperMode {
		return
	}

	// Succeeding test without any additional output or test logs.
}

var wantTestOK = `--- PASS: TestGoTest_Helper_OK.*`

func TestGoTest_Helper_AssertOutput(t *testing.T) {
	if !gotestHelperMode {
		return
	}

	t.Run("assert fails", func(t *testing.T) {
		assert.True(t, false)
	})

	t.Run("assert with message", func(t *testing.T) {
		assert.True(t, false, "My message")
	})

	t.Run("assert with messagef", func(t *testing.T) {
		assert.True(t, false, "My message with arguments: %v", 42)
	})

	t.Run("require fails", func(t *testing.T) {
		require.True(t, false)
	})

	t.Run("require with message", func(t *testing.T) {
		require.True(t, false, "My message")
	})

	t.Run("require with messagef", func(t *testing.T) {
		require.True(t, false, "My message with arguments: %v", 42)
	})

	t.Run("equals map", func(t *testing.T) {
		want := map[string]interface{}{
			"a": 1,
			"b": true,
			"c": "test",
			"e": map[string]interface{}{
				"x": "y",
			},
		}

		got := map[string]interface{}{
			"a": 42,
			"b": false,
			"c": "test",
		}

		assert.Equal(t, want, got)
	})
}

var wantTestAssertOutput = `(?sm:
=== Failed
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/assert_fails.*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestGoTest_Helper_AssertOutput/assert_fails.*
    --- FAIL: TestGoTest_Helper_AssertOutput/assert_fails .*
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/assert_with_message .*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestGoTest_Helper_AssertOutput/assert_with_message.*
        	Messages:   	My message.*
    --- FAIL: TestGoTest_Helper_AssertOutput/assert_with_message .*
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/assert_with_messagef .*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestGoTest_Helper_AssertOutput/assert_with_messagef.*
        	Messages:   	My message with arguments: 42.*
    --- FAIL: TestGoTest_Helper_AssertOutput/assert_with_messagef .*
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/require_fails .*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestGoTest_Helper_AssertOutput/require_fails.*
    --- FAIL: TestGoTest_Helper_AssertOutput/require_fails .*
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/require_with_message .*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestGoTest_Helper_AssertOutput/require_with_message.*
        	Messages:   	My message.*
    --- FAIL: TestGoTest_Helper_AssertOutput/require_with_message .*
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/require_with_messagef .*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
        	Error:      	Should be true.*
        	Test:       	TestGoTest_Helper_AssertOutput/require_with_messagef.*
        	Messages:   	My message with arguments: 42.*
    --- FAIL: TestGoTest_Helper_AssertOutput/require_with_messagef .*
=== FAIL: dev-tools/mage TestGoTest_Helper_AssertOutput/equals_map .*
    gotest_test.go:\d+:.*
        	Error Trace:	gotest_test.go:\d+.*
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

func TestGoTest_Helper_LogOutput(t *testing.T) {
	if !gotestHelperMode {
		return
	}

	t.Run("on error", func(t *testing.T) {
		t.Log("Log message should be printed")
		t.Logf("printf style log message: %v", 42)
		t.Error("Log should fail")
		t.Errorf("Log should fail with printf style log: %v", 23)
	})

	t.Run("on fatal", func(t *testing.T) {
		t.Log("Log message should be printed")
		t.Logf("printf style log message: %v", 42)
		t.Fatal("Log should fail")
	})

	t.Run("on fatalf", func(t *testing.T) {
		t.Log("Log message should be printed")
		t.Logf("printf style log message: %v", 42)
		t.Fatalf("Log should fail with printf style log: %v", 42)
	})

	t.Run("with newlines", func(t *testing.T) {
		t.Log("Log\nmessage\nshould\nbe\nprinted")
		t.Logf("printf\nstyle\nlog\nmessage:\n%v", 42)
		t.Fatalf("Log\nshould\nfail\nwith\nprintf\nstyle\nlog:\n%v", 42)
	})
}

var wantTestLogOutput = `(?sm:
=== Failed.*
=== FAIL: dev-tools/mage TestGoTest_Helper_LogOutput/on_error.*
    gotest_test.go:\d+: Log message should be printed.*
    gotest_test.go:\d+: printf style log message: 42.*
    gotest_test.go:\d+: Log should fail.*
    gotest_test.go:\d+: Log should fail with printf style log: 23.*
    --- FAIL: TestGoTest_Helper_LogOutput/on_error.*
=== FAIL: dev-tools/mage TestGoTest_Helper_LogOutput/on_fatal.*
    gotest_test.go:\d+: Log message should be printed.*
    gotest_test.go:\d+: printf style log message: 42.*
    gotest_test.go:\d+: Log should fail.*
    --- FAIL: TestGoTest_Helper_LogOutput/on_fatal.*
=== FAIL: dev-tools/mage TestGoTest_Helper_LogOutput/on_fatalf.*
    gotest_test.go:\d+: Log message should be printed.*
    gotest_test.go:\d+: printf style log message: 42.*
    gotest_test.go:\d+: Log should fail with printf style log: 42.*
    --- FAIL: TestGoTest_Helper_LogOutput/on_fatalf.*
=== FAIL: dev-tools/mage TestGoTest_Helper_LogOutput/with_newlines.*
    gotest_test.go:\d+: Log.*
        message.*
        should.*
        be.*
        printed.*
    gotest_test.go:\d+: printf.*
        style.*
        log.*
        message:.*
        42.*
    gotest_test.go:\d+: Log.*
        should.*
        fail.*
        with.*
        printf.*
        style.*
        log:.*
        42.*
    --- FAIL: TestGoTest_Helper_LogOutput/with_newlines.*
=== FAIL: dev-tools/mage TestGoTest_Helper_LogOutput.*
DONE 5 tests, 5 failures in.*
)`

func TestGoTest_Helper_WithPanic(t *testing.T) {
	if !gotestHelperMode {
		return
	}

	panic("Kaputt.")
}

var wantTestWithPanic = `(?sm:
=== FAIL: dev-tools/mage TestGoTest_Helper_WithPanic.*
panic: Kaputt. \[recovered\].*
	panic: Kaputt.*
)`

func TestGoTest_Helper_WithWrongPanic(t *testing.T) {
	if !gotestHelperMode {
		return
	}

	t.Run("setup failing go-routine", func(t *testing.T) {
		go func() {
			time.Sleep(1 * time.Second)
			t.Error("oops")
		}()
	})

	t.Run("false positive failure", func(t *testing.T) {
		time.Sleep(10 * time.Second)
	})
}

// The regular expression must very forgiving. Unfortunately the order of the
// tests and log lines can differ per run.
var wantTestWithWrongPanic = `(?sm:
=== FAIL: dev-tools/mage TestGoTest_Helper_WithWrongPanic.*
.*
panic: Fail in goroutine after TestGoTest_Helper_WithWrongPanic/setup_failing_go-routine has completed.*
)`
