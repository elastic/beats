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
	"fmt"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/conditions"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	type config map[string]any

	tests := []struct {
		title    string
		filter   config
		events   []mapstr.M
		expected int
	}{
		{
			"condition_matches",
			config{"when.equals.i": 10},
			[]mapstr.M{{"i": 10}},
			1,
		},
		{
			"condition_fails",
			config{"when.equals.i": 11},
			[]mapstr.M{{"i": 10}},
			0,
		},
		{
			"no_condition",
			config{},
			[]mapstr.M{{"i": 10}},
			1,
		},
		{
			"condition_matches",
			config{"when.has_fields": []string{"i"}},
			[]mapstr.M{{"i": 10}},
			1,
		},
		{
			"condition_fails",
			config{"when.has_fields": []string{"j"}},
			[]mapstr.M{{"i": 10}},
			0,
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.title)

		config, err := conf.NewConfigFrom(test.filter)
		if err != nil {
			t.Error(err)
			continue
		}

		cf := &countFilter{}
		filter, err := NewConditional(func(_ *conf.C, log *logp.Logger) (beat.Processor, error) {
			return cf, nil
		})(config, logptest.NewTestingLogger(t, ""))
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
	filter, err := NewConditional(func(_ *conf.C, log *logp.Logger) (beat.Processor, error) {
		return nil, testErr
	})(conf.NewConfig(), logptest.NewTestingLogger(t, ""))

	assert.Equal(t, testErr, err)
	assert.Nil(t, filter)
}

type testCase struct {
	event mapstr.M
	want  mapstr.M
	cfg   string
}

func testProcessors(t *testing.T, cases map[string]testCase) {
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			c, err := conf.NewConfigWithYAML([]byte(test.cfg), "test "+name)
			if err != nil {
				t.Fatal(err)
			}

			var pluginConfig PluginConfig
			if err = c.Unpack(&pluginConfig); err != nil {
				t.Fatal(err)
			}

			processor, err := New(pluginConfig, logptest.NewTestingLogger(t, ""))
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
			event: mapstr.M{"uid": 411},
			want:  mapstr.M{"uid": 411, "uid_type": "reserved"},
			cfg:   ifThen,
		},
		"if-then-false": {
			event: mapstr.M{"uid": 500},
			want:  mapstr.M{"uid": 500},
			cfg:   ifThen,
		},
		"if-then-else-true": {
			event: mapstr.M{"uid": 411},
			want:  mapstr.M{"uid": 411, "uid_type": "reserved"},
			cfg:   ifThenElse,
		},
		"if-then-else-false": {
			event: mapstr.M{"uid": 500},
			want:  mapstr.M{"uid": 500, "uid_type": "user"},
			cfg:   ifThenElse,
		},
		"if-then-else-false-single-processor": {
			event: mapstr.M{"uid": 500},
			want:  mapstr.M{"uid": 500, "uid_type": "user"},
			cfg:   ifThenElseSingleProcessor,
		},
		"if-then-else-if": {
			event: mapstr.M{"uid": 500},
			want:  mapstr.M{"uid": 500, "uid_type": "eq_500"},
			cfg:   ifThenElseIf,
		},
	})
}

var ErrProcessorClose = fmt.Errorf("error processor close error")

type errorProcessor struct{}

func (c *errorProcessor) Run(e *beat.Event) (*beat.Event, error) {
	return e, nil
}
func (c *errorProcessor) String() string { return "error_processor" }
func (c *errorProcessor) Close() error {
	return ErrProcessorClose
}

var ErrSetPathsProcessor = fmt.Errorf("error processor set paths error")

type setPathsProcessor struct{}

func (c *setPathsProcessor) Run(e *beat.Event) (*beat.Event, error) {
	return e, nil
}
func (c *setPathsProcessor) String() string { return "error_processor" }
func (c *setPathsProcessor) SetPaths(p *paths.Path) error {
	return fmt.Errorf("error_processor set paths error: %s", p)
}

func TestConditionRuleClose(t *testing.T) {
	const whenCondition = `
contains.a: b
`
	c, err := conf.NewConfigWithYAML([]byte(whenCondition), "when config")
	require.NoError(t, err)

	condConfig := conditions.Config{}
	err = c.Unpack(&condConfig)
	require.NoError(t, err)

	ep := &errorProcessor{}
	condRule, err := NewConditionRule(condConfig, ep, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{"a": "b"},
	}
	result, err := condRule.Run(event)
	require.NoError(t, err)
	require.Equal(t, event, result)
	err = Close(condRule)
	require.ErrorIs(t, err, ErrProcessorClose)
}

func TestIfThenElseProcessorClose(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	thenProcessors := &Processors{
		List: []beat.Processor{&errorProcessor{}},
		log:  logger,
	}
	elsProcessors := &Processors{
		List: []beat.Processor{&errorProcessor{}},
		log:  logger,
	}
	proc := &ClosingIfThenElseProcessor{
		IfThenElseProcessor{
			then: thenProcessors,
			els:  elsProcessors,
		},
	}
	err := Close(proc)
	require.ErrorIs(t, err, ErrProcessorClose)
	require.Equal(t, ErrProcessorClose.Error()+"\n"+ErrProcessorClose.Error(), err.Error())
}

func TestIfThenElseProcessorSetPaths(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	thenProcessors := &Processors{
		List: []beat.Processor{&setPathsProcessor{}},
		log:  logger,
	}
	elsProcessors := &Processors{
		List: []beat.Processor{&setPathsProcessor{}},
		log:  logger,
	}
	proc := &IfThenElseProcessor{
		cond: nil,
		then: thenProcessors,
		els:  elsProcessors,
	}

	// SetPaths should not panic when then is nil
	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
	err := proc.SetPaths(beatPaths)
	require.ErrorAs(t, err, &ErrSetPathsProcessor)
	require.ErrorContains(t, err, ErrSetPathsProcessor.Error())
	require.ErrorContains(t, err, beatPaths.String())
}

func TestIfThenElseProcessorSetPathsNil(t *testing.T) {
	const cfg = `
if:
  equals.test: value
then:
  - add_fields: {target: "", fields: {test_field: test_value}}
`
	c, err := conf.NewConfigWithYAML([]byte(cfg), "if-then config")
	require.NoError(t, err)

	beatProcessor, err := NewIfElseThenProcessor(c, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	proc, ok := beatProcessor.(SetPather)
	require.True(t, ok)

	// SetPaths should not panic when then is nil
	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
	err = proc.SetPaths(beatPaths)
	require.NoError(t, err)
}
