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

package channel

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions"
	"github.com/stretchr/testify/assert"
)

func TestBuildProcessorList(t *testing.T) {
	testCases := []struct {
		description    string
		beatInfo       beat.Info
		configStr      string
		clientCfg      beat.ClientConfig
		event          beat.Event
		expectedFields map[string]string
	}{
		{
			description: "Simple static index",
			configStr:   "index: 'test'",
			expectedFields: map[string]string{
				"@metadata.raw-index": "test",
			},
		},
		{
			description: "Index with agent info + timestamp",
			beatInfo:    beat.Info{Beat: "TestBeat", Version: "3.9.27"},
			configStr:   "index: 'beat-%{[agent.name]}-%{[agent.version]}-%{+yyyy.MM.dd}'",
			event:       beat.Event{Timestamp: time.Date(1999, time.December, 31, 23, 0, 0, 0, time.UTC)},
			expectedFields: map[string]string{
				"@metadata.raw-index": "beat-TestBeat-3.9.27-1999.12.31",
			},
		},
		{
			description: "Set index in ClientConfig",
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw-index": "clientCfgIndex",
			},
		},
		{
			description: "ClientConfig processor runs after beat input Index",
			configStr:   "index: 'test'",
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw-index": "clientCfgIndex",
			},
		},
		{
			description: "Set field in input config",
			configStr: `processors:
- add_fields: {fields: {testField: inputConfig}}`,
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
		{
			description: "Set field in ClientConfig",
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(actions.NewAddFields(common.MapStr{
						"fields": common.MapStr{"testField": "clientConfig"},
					}, false)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "clientConfig",
			},
		},
		{
			description: "Input config processors run after ClientConfig",
			configStr: `processors:
- add_fields: {fields: {testField: inputConfig}}`,
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(actions.NewAddFields(common.MapStr{
						"fields": common.MapStr{"testField": "clientConfig"},
					}, false)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
	}
	for _, test := range testCases {
		if test.event.Fields == nil {
			test.event.Fields = common.MapStr{}
		}
		config, err := outletConfigFromString(test.configStr)
		if err != nil {
			t.Errorf("[%s] %v", test.description, err)
			continue
		}
		processors, err := buildProcessorList(test.beatInfo, config, test.clientCfg)
		if err != nil {
			t.Errorf("[%s] %v", test.description, err)
			continue
		}
		processedEvent, err := processors.Run(&test.event)
		if err != nil {
			t.Error(err)
			continue
		}
		for key, value := range test.expectedFields {
			field, err := processedEvent.GetValue(key)
			if err != nil {
				t.Errorf("[%s] Couldn't get field %s from event: %v", test.description, key, err)
				continue
			}
			assert.Equal(t, field, value)
			fieldStr, ok := field.(string)
			if !ok {
				// Note that requiring a string here is just to simplify the test setup,
				// not a requirement of the underlying api.
				t.Errorf("[%s] Field [%s] should be a string", test.description, key)
				continue
			}
			if fieldStr != value {
				t.Errorf("[%s] Event field [%s]: expected [%s], got [%s]", test.description, key, value, fieldStr)
			}
		}
	}
}

// setRawIndex is a bare-bones processor to set the raw-index field to a
// constant string in the event metadata. It is used to test order of operations
// for buildProcessorList.
type setRawIndex struct {
	indexStr string
}

func (p *setRawIndex) Run(event *beat.Event) (*beat.Event, error) {
	if event.Meta == nil {
		event.Meta = common.MapStr{}
	}
	event.Meta["raw-index"] = p.indexStr
	return event, nil
}

func (p *setRawIndex) String() string {
	return fmt.Sprintf("set_raw_index=%v", p.indexStr)
}

// Helper function to convert from YML input string to an unpacked
// inputOutletConfig
func outletConfigFromString(s string) (inputOutletConfig, error) {
	config := inputOutletConfig{}
	cfg, err := common.NewConfigFrom(s)
	if err != nil {
		return config, err
	}
	if err := cfg.Unpack(&config); err != nil {
		return config, err
	}
	return config, nil
}

// makeProcessors wraps one or more bare Processor objects in Processors.
func makeProcessors(procs ...processors.Processor) *processors.Processors {
	procList := processors.NewList(nil)
	procList.List = procs
	return procList
}
