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

package fmtstr

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatString(t *testing.T) {
	tests := []struct {
		title       string
		pattern     string
		dyn, lookup map[string]string
		expected    string
	}{
		{
			"no interpolations",
			"no interpolations",
			nil, nil,
			"no interpolations",
		},
		{
			"simple lookup standalone",
			"%{k}",
			nil, map[string]string{"k": "v"},
			"v",
		},
		{
			"simple lookup start of string",
			"%{k} test",
			nil, map[string]string{"k": "v"},
			"v test",
		},
		{
			"simple lookup end of string",
			"test %{k}",
			nil, map[string]string{"k": "v"},
			"test v",
		},
		{
			"simple lookup middle of string",
			"pre %{k} post",
			nil, map[string]string{"k": "v"},
			"pre v post",
		},
		{
			"compile lookup default",
			"%{unknown:default}",
			nil, nil,
			"default",
		},
		{
			"just with % symbol",
			"just with % symbol",
			nil, nil,
			"just with % symbol",
		},
		{
			"with escaped % symbol",
			`\%{abc}`,
			nil, nil,
			"%{abc}",
		},
		{
			"with dynamic evaluation",
			"my dynamic %{key}",
			map[string]string{"key": "value"}, nil,
			"my dynamic value",
		},
		{
			"test mixed",
			"pre %{c} abc %{d} def %{c} post",
			map[string]string{"d": "dynamic"},
			map[string]string{"c": "const"},
			"pre const abc dynamic def const post",
		},
	}

	for i, test := range tests {
		// stringElement wraps StringElement in order to disable
		// optimization and enforce evaluation of formatter.
		type stringElement struct {
			StringElement
		}

		t.Logf("run (%v): '%v'", i, test.title)

		// compile format string with test key lookup
		sf, err := Compile(test.pattern,
			func(key string, ops []VariableOp) (FormatEvaler, error) {
				if test.lookup != nil {
					if v, found := test.lookup[key]; found {
						return StringElement{v}, nil
					}
				}

				if test.dyn != nil {
					if v, found := test.dyn[key]; found {
						return stringElement{StringElement{v}}, nil
					}
				}

				if len(ops) == 0 {
					return nil, errors.New("no default operator")
				}

				op := ops[0]
				if op.op != ":" {
					return nil, fmt.Errorf("invalid op: '%v'", op.op)
				}

				return StringElement{ops[0].param}, nil
			},
		)

		// validate compile ok
		if err != nil {
			t.Error(err)
			continue
		}

		// run string formatter
		actual, err := sf.Run(nil)
		if err != nil {
			t.Error(err)
			continue
		}

		// test validation
		if test.dyn == nil {
			assert.True(t, sf.IsConst())
		} else {
			assert.False(t, sf.IsConst())
		}
		assert.Equal(t, test.expected, actual)
	}
}

func TestFormatStringErrors(t *testing.T) {
	tests := []struct {
		title  string
		format string
	}{
		{"missing close", "%{key"},
		{"nesting not allowed", "%{key %{nested}}"},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		_, err := Compile(test.format, nil)
		assert.Error(t, err)
	}
}

func TestParseRawTokens(t *testing.T) {

	testCases := []struct {
		name         string
		input        string
		expectedList []any
		err          error
	}{

		{
			name:         "empty string",
			input:        "",
			expectedList: nil,
		},
		{
			name:         `when two %%`,
			input:        `%%`,
			expectedList: []any{"%%"},
		},
		{
			name:  `when input is %%{}`,
			input: `%%{}`,
			err:   fmt.Errorf("empty format expansion"),
		},
		{
			name:         `when input is %\{}`,
			input:        `%\{}`,
			expectedList: []any{"%{}"},
		},
		{
			name:  `when input is %{}`,
			input: `%{}`,
			err:   fmt.Errorf("empty format expansion"),
		},
		{
			name:         `when input is %{key}\\`,
			input:        `%{key}\\`,
			expectedList: []any{VariableToken("key"), `\`},
		},
		{
			name:         `when input is %{a:b:c}`,
			input:        `%{a:b:c}`,
			expectedList: []any{VariableToken("a:b:c")},
		},
		{
			name:  `when input is %{a`,
			input: `%{a`,
			err:   fmt.Errorf(`missing closing '}'`),
		},
		{
			name:         "simple lookup start of string",
			input:        "%{k} test",
			expectedList: []any{VariableToken("k"), " test"},
		},
		{
			name:         "simple lookup end of string",
			input:        "test %{k}",
			expectedList: []any{"test ", VariableToken("k")},
		},
		{
			name:         "simple lookup middle of string",
			input:        "pre %{k} post",
			expectedList: []any{"pre ", VariableToken("k"), " post"},
		},
		{
			name:         "compile lookup default",
			input:        "%{unknown:default}",
			expectedList: []any{VariableToken("unknown:default")},
		},
		{
			name:         "with escaped % symbol",
			input:        `\%{abc}`,
			expectedList: []any{`%{abc}`},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			lexer := MakeLexer(test.input)
			defer lexer.Finish()
			got, err := ParseRawTokens(lexer)
			if test.err != nil {
				require.Equal(t, test.err, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedList, got)
		})
	}
}

func FuzzParseRawTokens(f *testing.F) {
	f.Add("%{k} test")
	f.Add(`pre %{k} post`)
	f.Add("%{unknown:default}")

	f.Fuzz(func(t *testing.T, a string) {
		lex := MakeLexer(a)
		defer lex.Finish()
		output, err := ParseRawTokens(lex)
		if err != nil {
			t.Logf("skipping input %s with error: %v", a, err)
			return // invalid input
		}

		if strings.Contains(a, `\`) {
			return // we cannot rebuild the input if it contains escape character
		}
		// stringify output and match it with original input
		var finalOutput string
		for _, out := range output {
			switch t := out.(type) {
			case string:
				finalOutput += t
			case VariableToken:
				finalOutput += "%{" + string(t) + "}"
			}
		}

		require.Equal(t, a, finalOutput)
	})
}
