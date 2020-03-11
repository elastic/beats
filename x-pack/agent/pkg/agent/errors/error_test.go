// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestErrorsWrap(t *testing.T) {
	ce := New("custom error", TypePath, M("k", "v"))
	ew := errors.Wrap(ce, "wrapper")
	outer := New(ew)

	outerCustom, ok := outer.(Error)
	if !ok {
		t.Error("expected Error")
		return
	}

	if tt := outerCustom.Type(); tt != TypePath {
		t.Errorf("expected type Path got %v", tt)
	}

	meta := outerCustom.Meta()
	if _, found := meta["k"]; !found {
		t.Errorf("expected meta with key 'k' but not found")
	}
}

func TestErrors(t *testing.T) {
	type testCase struct {
		id                   string
		expectedType         ErrorType
		expectedReadableType string
		expectedError        string
		expectedMeta         map[string]interface{}
		args                 []interface{}
	}

	cases := []testCase{
		testCase{"custom message", TypeUnexpected, "UNEXPECTED", "msg1: err1", nil, []interface{}{fmt.Errorf("err1"), "msg1"}},
		testCase{"no message", TypeUnexpected, "UNEXPECTED", "err1", nil, []interface{}{fmt.Errorf("err1")}},

		testCase{"custom type (crash)", TypeApplicationCrash, "CRASH", "msg1: err1", nil, []interface{}{fmt.Errorf("err1"), "msg1", TypeApplicationCrash}},
		testCase{"custom type (config)", TypeConfig, "CONFIG", "msg1: err1", nil, []interface{}{fmt.Errorf("err1"), "msg1", TypeConfig}},
		testCase{"custom type (path)", TypePath, "PATH", "msg1: err1", nil, []interface{}{fmt.Errorf("err1"), "msg1", TypePath}},

		testCase{"meta simple", TypeUnexpected, "UNEXPECTED", "msg1: err1", map[string]interface{}{"a": 1}, []interface{}{fmt.Errorf("err1"), "msg1", M("a", 1)}},
		testCase{"meta two keys", TypeUnexpected, "UNEXPECTED", "msg1: err1", map[string]interface{}{"a": 1, "b": 21}, []interface{}{fmt.Errorf("err1"), "msg1", M("a", 1), M("b", 21)}},
		testCase{"meta overriding key", TypeUnexpected, "UNEXPECTED", "msg1: err1", map[string]interface{}{"a": 21}, []interface{}{fmt.Errorf("err1"), "msg1", M("a", 1), M("a", 21)}},

		testCase{"overriding custom message", TypeUnexpected, "UNEXPECTED", "msg2: err1", nil, []interface{}{fmt.Errorf("err1"), "msg1", "msg2"}},
		testCase{"overriding custom type (crash)", TypeApplicationCrash, "CRASH", "msg1: err1", nil, []interface{}{fmt.Errorf("err1"), "msg1", TypeConfig, TypeApplicationCrash}},
		testCase{"overriding error", TypeUnexpected, "UNEXPECTED", "err2", nil, []interface{}{fmt.Errorf("err1"), fmt.Errorf("err2")}},
	}

	for _, tc := range cases {
		actualErr := New(tc.args...)
		agentErr, ok := actualErr.(Error)
		if !ok {
			t.Errorf("[%s] expected Error", tc.id)
			continue
		}

		if e := agentErr.Error(); e != tc.expectedError {
			t.Errorf("[%s] expected error: '%s', got '%s'", tc.id, tc.expectedError, e)
		}
		if e := agentErr.Type(); e != tc.expectedType {
			t.Errorf("[%s] expected error type: '%v', got '%v'", tc.id, tc.expectedType, e)
		}
		if e := agentErr.ReadableType(); e != tc.expectedReadableType {
			t.Errorf("[%s] expected error readable type: '%v', got '%v'", tc.id, tc.expectedReadableType, e)
		}

		if e := agentErr.Meta(); len(e) != len(tc.expectedMeta) {
			t.Errorf("[%s] expected meta length: '%v', got '%v'", tc.id, len(tc.expectedReadableType), len(e))
		}

		if len(tc.expectedMeta) != 0 {
			e := agentErr.Meta()
			for ek, ev := range tc.expectedMeta {
				v, found := e[ek]
				if !found {
					t.Errorf("[%s] expected meta key: '%v' not found", tc.id, ek)
				}

				if ev != v {
					t.Errorf("[%s] expected meta value for key: '%v' not equal. Expected: '%v', got: '%v'", tc.id, ek, ev, v)
				}
			}
		}
	}
}

func TestNoErrorNoMsg(t *testing.T) {
	actualErr := New()
	agentErr, ok := actualErr.(Error)
	if !ok {
		t.Error("expected Error")
		return
	}

	e := agentErr.Error()
	if !strings.Contains(e, "error_test.go[") {
		t.Errorf("Error does not contain source file: %v", e)
	}

	if !strings.HasSuffix(e, ": unknown error") {
		t.Errorf("Error does not contain default error: %v", e)
	}
}

func TestNoError(t *testing.T) {
	// test with message
	msg := "msg2"
	actualErr := New(msg)
	agentErr, ok := actualErr.(Error)
	if !ok {
		t.Error("expected Error")
		return
	}

	e := agentErr.Error()
	if !strings.Contains(e, "error_test.go[") {
		t.Errorf("Error does not contain source file: %v", e)
	}

	if !strings.HasSuffix(e, ": unknown error") {
		t.Errorf("Error does not contain default error: %v", e)
	}

	if !strings.HasPrefix(e, msg) {
		t.Errorf("Error does not contain provided message: %v", e)
	}
}

func TestMetaFold(t *testing.T) {
	err1 := fmt.Errorf("level1")
	err2 := New("level2", err1, M("key1", "level2"), M("key2", "level2"))
	err3 := New("level3", err2, M("key1", "level3"), M("key3", "level3"))
	err4 := New("level4", err3)

	resultingErr, ok := err4.(Error)
	if !ok {
		t.Fatal("error is not Error")
	}

	meta := resultingErr.Meta()
	expectedMeta := map[string]interface{}{
		"key1": "level3",
		"key2": "level2",
		"key3": "level3",
	}

	if len(expectedMeta) != len(meta) {
		t.Fatalf("Metadata do not match expected '%v' got '%v'", expectedMeta, meta)
	}

	for ek, ev := range expectedMeta {
		v, found := meta[ek]
		if !found {
			t.Errorf("Key '%s' not found in a meta collection", ek)
			continue
		}

		if v != ev {
			t.Errorf("Values for key '%s' don't match. Expected: '%v', got '%v'", ek, ev, v)
		}
	}
}

func TestMetaCallDoesNotModifyCollection(t *testing.T) {
	err1 := fmt.Errorf("level1")
	err2 := New("level2", err1, M("key1", "level2"), M("key2", "level2"))
	err3 := New("level3", err2, M("key1", "level3"), M("key3", "level3"))
	err4 := New("level4", err3)

	resultingErr, ok := err4.(agentError)
	if !ok {
		t.Fatal("error is not Error")
	}

	resultingErr.Meta()

	if len(resultingErr.meta) != 0 {
		t.Fatalf("err4.meta modified by calling Meta(): %v", resultingErr.meta)
	}
}
