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

package processor

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/script/javascript"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/require"
)

func init() {
	RegisterPlugin("Mock", newMock)
	RegisterPlugin("MockWithCloser", newMockWithCloser)
}

func testEvent() *beat.Event {
	return &beat.Event{
		Fields: mapstr.M{
			"source": mapstr.M{
				"ip": "192.0.2.1",
			},
			"destination": mapstr.M{
				"ip": "192.0.2.1",
			},
			"network": mapstr.M{
				"transport": "igmp",
			},
			"message": "key=hello",
		},
	}
}

func TestNewProcessorDummyProcessor(t *testing.T) {
	const script = `
var processor = require('processor');

var mock = new processor.Mock({"fields": {"added": "new_value"}});

function process(evt) {
    mock.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	require.NoError(t, err)

	evt, err := p.Run(testEvent())
	require.NoError(t, err)

	checkEvent(t, evt, "added", "new_value")
}

func TestChainOfDummyProcessors(t *testing.T) {
	const script = `
var processor = require('processor');

var hungarianHello = new processor.Mock({"fields": {"helló": "világ"}});
var germanHello = new processor.Mock({"fields": {"hallo": "Welt"}});

var chain = new processor.Chain()
    .Add(hungarianHello)
    .Mock({
        fields: { "hola": "mundo" },
    })
    .Add(function(evt) {
        evt.Put("hello", "world");
    })
    .Build();

var chainOfChains = new processor.Chain()
    .Add(chain)
	.Add(germanHello)
    .Build();
function process(evt) {
    chainOfChains.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	require.NoError(t, err)

	evt, err := p.Run(testEvent())
	require.NoError(t, err)

	// checking if hello world is added to the event in different languages
	checkEvent(t, evt, "helló", "világ")
	checkEvent(t, evt, "hola", "mundo")
	checkEvent(t, evt, "hello", "world")
	checkEvent(t, evt, "hallo", "Welt")
}

func TestProcessorWithCloser(t *testing.T) {
	const script = `
var processor = require('processor');

var processorWithCloser = new processor.MockWithCloser().Build()

function process(evt) {
    processorWithCloser.Run(evt);
}
`

	logp.TestingSetup()
	_, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	require.Error(t, err, "processor that implements Closer() shouldn't be allowed")
}

func checkEvent(t *testing.T, evt *beat.Event, key, value string) {
	s, err := evt.GetValue(key)
	assert.NoError(t, err)

	switch ss := s.(type) {
	case string:
		assert.Equal(t, ss, value)
	default:
		t.Fatal("unexpected type")
	}
}

type mockProcessor struct {
	fields mapstr.M
}

func newMock(c *config.C) (processors.Processor, error) {
	config := struct {
		Fields mapstr.M `config:"fields" validate:"required"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the mock processor configuration: %s", err)
	}

	return &mockProcessor{
		fields: config.Fields,
	}, nil
}

func (m *mockProcessor) Run(event *beat.Event) (*beat.Event, error) {
	event.Fields.DeepUpdate(m.fields)
	return event, nil
}

func (m *mockProcessor) String() string {
	s, _ := json.Marshal(m.fields)
	return fmt.Sprintf("mock=%s", s)
}

type mockProcessorWithCloser struct{}

func newMockWithCloser(c *config.C) (processors.Processor, error) {
	return &mockProcessorWithCloser{}, nil
}

func (m *mockProcessorWithCloser) Run(event *beat.Event) (*beat.Event, error) {
	// Nothing to do, we only want this struct to implement processors.Closer
	return event, nil
}

func (m *mockProcessorWithCloser) Close() error {
	// Nothing to do, we only want this struct to implement processors.Closer
	return nil
}

func (m *mockProcessorWithCloser) String() string {
	return "mockWithCloser"
}
