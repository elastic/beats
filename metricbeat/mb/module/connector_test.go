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

package module

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestProcessorsForConfig(t *testing.T) {
	testCases := map[string]struct {
		beatInfo       beat.Info
		configStr      string
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
	}
	for description, test := range testCases {
		if test.event.Fields == nil {
			test.event.Fields = common.MapStr{}
		}
		config, err := connectorConfigFromString(test.configStr)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}
		processors, err := processorsForConfig(test.beatInfo, config)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}
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

type fakeMetricSetRegister struct {
	success bool
}

func (fmsr *fakeMetricSetRegister) ProcessorsForMetricSet(moduleName, metricSetName string) (*processors.Processors, error) {
	if !fmsr.success {
		return nil, errors.New("failure")
	}

	procs := new(processors.Processors)
	procs.List = []processors.Processor{nil, nil}
	return procs, nil
}

func TestUseMetricSetProcessors_ReadingProcessorsFailed(t *testing.T) {
	r := new(fakeMetricSetRegister)

	var connector Connector
	err := connector.UseMetricSetProcessors(r, "module", "metricset")
	require.Error(t, err)
	require.Nil(t, connector.processors)
}

func TestUseMetricSetProcessors_ReadingProcessorsSucceeded(t *testing.T) {
	r := &fakeMetricSetRegister{
		success: true,
	}

	connector := Connector{
		processors: &processors.Processors{
			List: []processors.Processor{},
		},
	}
	err := connector.UseMetricSetProcessors(r, "module", "metricset")
	require.NoError(t, err)
	require.Len(t, connector.processors.List, 2)
}

// Helper function to convert from YML input string to an unpacked
// connectorConfig
func connectorConfigFromString(s string) (connectorConfig, error) {
	config := connectorConfig{}
	cfg, err := conf.NewConfigFrom(s)
	if err != nil {
		return config, err
	}
	if err := cfg.Unpack(&config); err != nil {
		return config, err
	}
	return config, nil
}
