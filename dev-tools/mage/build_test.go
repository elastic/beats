package mage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BuildArgs_ParseExtraFlags(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect []string
	}{{
		name:   "no flags",
		input:  nil,
		expect: []string{},
	}, {
		name:   "multiple flags with no tags",
		input:  []string{"-a", "-b", "-key=value"},
		expect: []string{"-a", "-b", "-key=value"},
	}, {
		name:   "one build tag",
		input:  []string{"-tags=example"},
		expect: []string{"-tags=example"},
	}, {
		name:   "multiple build tags",
		input:  []string{"-tags=example", "-tags=test"},
		expect: []string{"-tags=example,test"},
	}, {
		name:   "joined build tags",
		input:  []string{"-tags=example,test"},
		expect: []string{"-tags=example,test"},
	}, {
		name:   "multiple build tags with other flags",
		input:  []string{"-tags=example", "-tags=test", "-key=value", "-a"},
		expect: []string{"-key=value", "-a", "-tags=example,test"},
	}, {
		name:   "incorrectly formatted tag",
		input:  []string{"-tags= example"},
		expect: []string{},
	}, {
		name:   "incorrectly formatted tag with valid tag",
		input:  []string{"-tags= example", "-tags=test"},
		expect: []string{"-tags=test"},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := BuildArgs{ExtraFlags: tc.input}
			flags := args.ParseExtraFlags()
			assert.EqualValues(t, tc.expect, flags)
		})
	}
}
