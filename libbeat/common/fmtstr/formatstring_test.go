package fmtstr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
