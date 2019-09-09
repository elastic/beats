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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/script/javascript"

	_ "github.com/elastic/beats/libbeat/processors/script/javascript/module/require"
)

func testEvent() *beat.Event {
	return &beat.Event{
		Fields: common.MapStr{
			"source": common.MapStr{
				"ip": "192.0.2.1",
			},
			"destination": common.MapStr{
				"ip": "192.0.2.1",
			},
			"network": common.MapStr{
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

	RegisterPlugin("Mock", newMock)

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	s, err := evt.GetValue("added")
	assert.NoError(t, err)

	switch ss := s.(type) {
	case string:
		assert.Equal(t, ss, "new_value")
	default:
		t.Fatal("unexpected type")
	}
}

type mockProcessor struct {
	fields common.MapStr
}

func newMock(c *common.Config) (processors.Processor, error) {
	config := struct {
		Fields common.MapStr `config:"fields" validate:"required"`
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
