// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package boolexp

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/fleet/x-pack/pkg/boolexp/parser"
)

var showDebug = lookupEnvOrDefault("DEBUG", "0")

type testVarStore struct {
	vars map[string]interface{}
}

func (s *testVarStore) Lookup(v string) (interface{}, bool) {
	val, ok := s.vars[v]
	return val, ok
}

func TestBoolexp(t *testing.T) {
	testcases := []struct {
		expression string
		result     bool
		err        bool
	}{
		// Variables
		{expression: "%{[hello.var]} == 'hello'", result: true},
		{expression: "contains(%{[hello.var]}, 'hell')", result: true},

		{expression: "true", result: true},
		{expression: "false", result: false},
		{expression: "!false", result: true},
		{expression: "!true", result: false},
		{expression: "!(1 == 1)", result: false},
		{expression: "NOT false", result: true},
		{expression: "NOT true", result: false},
		{expression: "not false", result: true},
		{expression: "not true", result: false},
		{expression: "NOT (1 == 1)", result: false},

		{expression: "1 == 1", result: true},
		{expression: "1 == 2", result: false},
		{expression: "1 != 2", result: true},
		{expression: "1 != 1", result: false},
		{expression: "'hello' == 'hello'", result: true},
		{expression: "'hello' == 'hola'", result: false},

		// and
		{expression: "(1 == 1) AND (2 == 2)", result: true},
		{expression: "(1 == 4) AND (2 == 2)", result: false},
		{expression: "(1 == 1) AND (2 == 3)", result: false},
		{expression: "(1 == 5) AND (2 == 3)", result: false},

		{expression: "1 == 1 AND 2 == 2", result: true},
		{expression: "1 == 4 AND 2 == 2", result: false},
		{expression: "1 == 1 AND 2 == 3", result: false},
		{expression: "1 == 5 AND 2 == 3", result: false},

		{expression: "(1 == 1) and (2 == 2)", result: true},
		{expression: "(1 == 4) and (2 == 2)", result: false},
		{expression: "(1 == 1) and (2 == 3)", result: false},
		{expression: "(1 == 5) and (2 == 3)", result: false},

		{expression: "(1 == 1) && (2 == 2)", result: true},
		{expression: "(1 == 4) && (2 == 2)", result: false},
		{expression: "(1 == 1) && (2 == 3)", result: false},
		{expression: "(1 == 5) && (2 == 3)", result: false},

		// or
		{expression: "(1 == 1) OR (2 == 2)", result: true},
		{expression: "(1 == 1) OR (3 == 2)", result: true},
		{expression: "(1 == 2) OR (2 == 2)", result: true},
		{expression: "(1 == 2) OR (2 == 2)", result: true},
		{expression: "(1 == 2) OR (1 == 2)", result: false},

		{expression: "(1 == 1) or (2 == 2)", result: true},
		{expression: "(1 == 1) or (3 == 2)", result: true},
		{expression: "(1 == 2) or (2 == 2)", result: true},
		{expression: "(1 == 2) or (2 == 2)", result: true},
		{expression: "(1 == 2) or (1 == 2)", result: false},

		{expression: "(1 == 1) || (2 == 2)", result: true},
		{expression: "(1 == 1) || (3 == 2)", result: true},
		{expression: "(1 == 2) || (2 == 2)", result: true},
		{expression: "(1 == 2) || (2 == 2)", result: true},
		{expression: "(1 == 2) || (1 == 2)", result: false},

		// mixed
		{expression: "((1 == 1) AND (2 == 2)) OR (2 != 3)", result: true},
		{expression: "(1 == 1 OR 2 == 2) AND 2 != 3", result: true},
		{expression: "((1 == 1) AND (2 == 2)) OR (2 != 3)", result: true},
		{expression: "1 == 1 OR 2 == 2 AND 2 != 3", result: true},

		// functions
		{expression: "len('hello') == 5", result: true},
		{expression: "len('hello') != 1", result: true},
		{expression: "len('hello') == 1", result: false},
		{expression: "(len('hello') == 5) AND (len('Hi') == 2)", result: true},
		{expression: "len('hello') == size('hello')", result: true},
		{expression: "len('hello') == size('hi')", result: false},
		{expression: "contains('hello', 'eial')", result: false},
		{expression: "contains('hello', 'hel')", result: true},
		{expression: "!contains('hello', 'hel')", result: false},
		{expression: "contains('hello', 'hel') == true", result: true},
		{expression: "contains('hello', 'hel') == false", result: false},
		{expression: "countArgs('A', 'B', 'C', 'D', 'E', 'F') == 6", result: true},
		{expression: "countArgs('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J') == 10", result: true},

		// integers
		{expression: "1 < 5", result: true},
		{expression: "10 < 5", result: false},
		{expression: "1 > 5", result: false},
		{expression: "10 > 5", result: true},
		{expression: "1 <= 5", result: true},
		{expression: "5 <= 5", result: true},
		{expression: "10 <= 5", result: false},
		{expression: "10 >= 5", result: true},
		{expression: "5 >= 5", result: true},
		{expression: "4 >= 5", result: false},

		// Floats
		{expression: "1 == 1.0", result: true},
		{expression: "1.0 == 1.0", result: true},
		{expression: "1.0 == 1", result: true},
		{expression: "1 != 2.0", result: true},
		{expression: "1.0 != 2.0", result: true},
		{expression: "1.0 != 2", result: true},
		{expression: "1 < 5.0", result: true},
		{expression: "10 < 5.0", result: false},
		{expression: "1 > 5.0", result: false},
		{expression: "10 > 5.0", result: true},
		{expression: "1 <= 5.0", result: true},
		{expression: "10 <= 5.0", result: false},
		{expression: "1 >= 5.0", result: false},
		{expression: "10 >= 5.0", result: true},
		{expression: "10 >= 10.0", result: true},
		{expression: "10 <= 10.0", result: true},

		// Bad expression and malformed expression
		{expression: "contains('hello')", err: true},
		{expression: "contains()", err: true},
		{expression: "contains()", err: true},
		{expression: "donotexist()", err: true},
	}

	store := &testVarStore{
		vars: map[string]interface{}{
			"hello.var": "hello",
		},
	}

	fn := func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expecting 1 argument received %d", len(args))
		}
		val, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("expecting a string received %T", args[0])
		}
		return len(val), nil
	}

	methods := NewMethodsReg()
	methods.Register("len", fn)
	// test function aliasing
	methods.Register("size", fn)
	// test multiples arguments function.
	methods.Register("contains", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting 2 arguments received %d", len(args))
		}

		haystack, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("args 1 must be a string and received %T", args[0])
		}

		needle, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("args 2 must be a string and received %T", args[0])
		}

		return strings.Contains(haystack, needle), nil
	},
	)

	methods.Register("countArgs", func(args []interface{}) (interface{}, error) {
		return len(args), nil
	})

	for _, test := range testcases {
		test := test
		var title string
		if test.err {
			title = fmt.Sprintf("%s failed parsing", test.expression)
		} else {
			title = fmt.Sprintf("%s => return %v", test.expression, test.result)
		}
		t.Run(title, func(t *testing.T) {
			if showDebug == "1" {
				debug(test.expression)
			}

			r, err := Eval(test.expression, methods, store)

			if test.err {
				require.Error(t, err)
				return
			}

			assert.Equal(t, test.result, r)
		})
	}
}

func debug(expression string) {
	raw := antlr.NewInputStream(expression)

	lexer := parser.NewBoolexpLexer(raw)
	for {
		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			break
		}
		fmt.Printf("%s (%q)\n",
			lexer.SymbolicNames[t.GetTokenType()], t.GetText())
	}
}

var result bool

func BenchmarkEval(b *testing.B) {
	fn := func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expecting 1 argument received %d", len(args))
		}
		val, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("expecting a string received %T", args[0])
		}
		return len(val), nil
	}

	methods := NewMethodsReg()
	methods.Register("len", fn)

	expression, _ := New("(len('hello') == 5) AND (len('Hi') == 2)", methods)

	var r bool
	for n := 0; n < b.N; n++ {
		r, _ = expression.Eval(nil)
	}
	result = r
}

func lookupEnvOrDefault(name, d string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return d
}
