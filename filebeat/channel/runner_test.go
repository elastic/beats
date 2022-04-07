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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/beat/events"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/processors"
	"github.com/elastic/beats/v8/libbeat/processors/actions"
)

func TestProcessorsForConfig(t *testing.T) {
	testCases := map[string]struct {
		beatInfo       beat.Info
		configStr      string
		clientCfg      beat.ClientConfig
		event          beat.Event
		expectedFields map[string]string
	}{
		"Simple static index": {
			configStr: "index: 'test'",
			expectedFields: map[string]string{
				"@metadata.raw_index": "test",
			},
		},
		"Index with agent info + timestamp": {
			beatInfo:  beat.Info{Beat: "TestBeat", Version: "3.9.27"},
			configStr: "index: 'beat-%{[agent.name]}-%{[agent.version]}-%{+yyyy.MM.dd}'",
			event:     beat.Event{Timestamp: time.Date(1999, time.December, 31, 23, 0, 0, 0, time.UTC)},
			expectedFields: map[string]string{
				"@metadata.raw_index": "beat-TestBeat-3.9.27-1999.12.31",
			},
		},
		"Set index in ClientConfig": {
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw_index": "clientCfgIndex",
			},
		},
		"ClientConfig processor runs after beat input Index": {
			configStr: "index: 'test'",
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw_index": "clientCfgIndex",
			},
		},
		"Set field in input config": {
			configStr: `processors: [add_fields: {fields: {testField: inputConfig}}]`,
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
		"Set field in ClientConfig": {
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(actions.NewAddFields(common.MapStr{
						"fields": common.MapStr{"testField": "clientConfig"},
					}, false, true)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "clientConfig",
			},
		},
		"Input config processors run after ClientConfig": {
			configStr: `processors: [add_fields: {fields: {testField: inputConfig}}]`,
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(actions.NewAddFields(common.MapStr{
						"fields": common.MapStr{"testField": "clientConfig"},
					}, false, true)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
	}
	for description, test := range testCases {
		if test.event.Fields == nil {
			test.event.Fields = common.MapStr{}
		}
		config, err := common.NewConfigFrom(test.configStr)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}

		editor, err := newCommonConfigEditor(test.beatInfo, config)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}

		clientCfg, err := editor(test.clientCfg)
		require.NoError(t, err)

		processors := clientCfg.Processing.Processor
		processedEvent, err := processors.Run(&test.event)
		// We don't check if err != nil, because we are testing the final outcome
		// of running the processors, including when some of them fail.
		if processedEvent == nil {
			t.Errorf("[%s] Unexpected fatal error running processors: %v\n",
				description, err)
		}
		for key, value := range test.expectedFields {
			field, err := processedEvent.GetValue(key)
			if err != nil {
				t.Errorf("[%s] Couldn't get field %s from event: %v", description, key, err)
				continue
			}
			assert.Equal(t, field, value)
			fieldStr, ok := field.(string)
			if !ok {
				// Note that requiring a string here is just to simplify the test setup,
				// not a requirement of the underlying api.
				t.Errorf("[%s] Field [%s] should be a string", description, key)
				continue
			}
			if fieldStr != value {
				t.Errorf("[%s] Event field [%s]: expected [%s], got [%s]", description, key, value, fieldStr)
			}
		}
	}
}

func TestProcessorsForConfigIsFlat(t *testing.T) {
	// This test is regrettable, and exists because of inconsistencies in
	// processor handling between processors.Processors and processing.group
	// (which implements beat.ProcessorList) -- see processorsForConfig for
	// details. The upshot is that, for now, if the input configuration specifies
	// processors, they must be returned as direct children of the resulting
	// processors.Processors (rather than being collected in additional tree
	// structure).
	// This test should be removed once we have a more consistent mechanism for
	// collecting and running processors.
	configStr := `processors:
- add_fields: {fields: {testField: value}}
- add_fields: {fields: {testField2: stuff}}`
	config, err := common.NewConfigFrom(configStr)
	if err != nil {
		t.Fatal(err)
	}

	editor, err := newCommonConfigEditor(beat.Info{}, config)
	if err != nil {
		t.Fatal(err)
	}

	clientCfg, err := editor(beat.ClientConfig{})
	require.NoError(t, err)

	lst := clientCfg.Processing.Processor
	assert.Equal(t, 2, len(lst.(*processors.Processors).List))
}

// setRawIndex is a bare-bones processor to set the raw_index field to a
// constant string in the event metadata. It is used to test order of operations
// for processorsForConfig.
type setRawIndex struct {
	indexStr string
}

func (p *setRawIndex) Run(event *beat.Event) (*beat.Event, error) {
	if event.Meta == nil {
		event.Meta = common.MapStr{}
	}
	event.Meta[events.FieldMetaRawIndex] = p.indexStr
	return event, nil
}

func (p *setRawIndex) String() string {
	return fmt.Sprintf("set_raw_index=%v", p.indexStr)
}

// makeProcessors wraps one or more bare Processor objects in Processors.
func makeProcessors(procs ...processors.Processor) *processors.Processors {
	procList := processors.NewList(nil)
	procList.List = procs
	return procList
}
