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

package processors

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type countFilter struct {
	N int
}

func (c *countFilter) Run(e *beat.Event) (*beat.Event, error) {
	c.N++
	return e, nil
}

func (c *countFilter) String() string { return "count" }

func TestWhenProcessor(t *testing.T) {
	type config map[string]interface{}

	tests := []struct {
		title    string
		filter   config
		events   []common.MapStr
		expected int
	}{
		{
			"condition_matches",
			config{"when.equals.i": 10},
			[]common.MapStr{{"i": 10}},
			1,
		},
		{
			"condition_fails",
			config{"when.equals.i": 11},
			[]common.MapStr{{"i": 10}},
			0,
		},
		{
			"no_condition",
			config{},
			[]common.MapStr{{"i": 10}},
			1,
		},
		{
			"condition_matches",
			config{"when.has_fields": []string{"i"}},
			[]common.MapStr{{"i": 10}},
			1,
		},
		{
			"condition_fails",
			config{"when.has_fields": []string{"j"}},
			[]common.MapStr{{"i": 10}},
			0,
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.title)

		config, err := common.NewConfigFrom(test.filter)
		if err != nil {
			t.Error(err)
			continue
		}

		cf := &countFilter{}
		filter, err := NewConditional(func(_ *common.Config) (Processor, error) {
			return cf, nil
		})(config)
		if err != nil {
			t.Error(err)
			continue
		}

		for _, fields := range test.events {
			event := &beat.Event{
				Timestamp: time.Now(),
				Fields:    fields,
			}
			_, err := filter.Run(event)
			if err != nil {
				t.Error(err)
			}
		}

		assert.Equal(t, test.expected, cf.N)
	}
}

func TestConditionRuleInitErrorPropagates(t *testing.T) {
	testErr := errors.New("test")
	filter, err := NewConditional(func(_ *common.Config) (Processor, error) {
		return nil, testErr
	})(common.NewConfig())

	assert.Equal(t, testErr, err)
	assert.Nil(t, filter)
}

type testCase struct {
	event common.MapStr
	want  common.MapStr
	cfg   string
}

func testProcessors(t *testing.T, cases map[string]testCase) {
	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			c, err := common.NewConfigWithYAML([]byte(test.cfg), "test "+name)
			if err != nil {
				t.Fatal(err)
			}

			var pluginConfig PluginConfig
			if err = c.Unpack(&pluginConfig); err != nil {
				t.Fatal(err)
			}

			processor, err := New(pluginConfig)
			if err != nil {
				t.Fatal(err)
			}

			result, err := processor.Run(&beat.Event{Fields: test.event.Clone()})
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.want, result.Fields)
		})
	}
}

func TestIfElseThenProcessor(t *testing.T) {
	const ifThen = `
- if:
    range.uid.lt: 500
  then:
    - add_fields: {target: "", fields: {uid_type: reserved}}
`

	const ifThenElse = `
- if:
    range.uid.lt: 500
  then:
    - add_fields: {target: "", fields: {uid_type: reserved}}
  else:
    - add_fields: {target: "", fields: {uid_type: user}}
`

	const ifThenElseSingleProcessor = `
- if:
    range.uid.lt: 500
  then:
    add_fields: {target: "", fields: {uid_type: reserved}}
  else:
    add_fields: {target: "", fields: {uid_type: user}}
`

	const ifThenElseIf = `
- if:
    range.uid.lt: 500
  then:
    - add_fields: {target: "", fields: {uid_type: reserved}}
  else:
    if:
      equals.uid: 500
    then:
      add_fields: {target: "", fields: {uid_type: "eq_500"}}
    else:
      add_fields: {target: "", fields: {uid_type: "gt_500"}}
`

	testProcessors(t, map[string]testCase{
		"if-then-true": {
			event: common.MapStr{"uid": 411},
			want:  common.MapStr{"uid": 411, "uid_type": "reserved"},
			cfg:   ifThen,
		},
		"if-then-false": {
			event: common.MapStr{"uid": 500},
			want:  common.MapStr{"uid": 500},
			cfg:   ifThen,
		},
		"if-then-else-true": {
			event: common.MapStr{"uid": 411},
			want:  common.MapStr{"uid": 411, "uid_type": "reserved"},
			cfg:   ifThenElse,
		},
		"if-then-else-false": {
			event: common.MapStr{"uid": 500},
			want:  common.MapStr{"uid": 500, "uid_type": "user"},
			cfg:   ifThenElse,
		},
		"if-then-else-false-single-processor": {
			event: common.MapStr{"uid": 500},
			want:  common.MapStr{"uid": 500, "uid_type": "user"},
			cfg:   ifThenElseSingleProcessor,
		},
		"if-then-else-if": {
			event: common.MapStr{"uid": 500},
			want:  common.MapStr{"uid": 500, "uid_type": "eq_500"},
			cfg:   ifThenElseIf,
		},
	})
}
