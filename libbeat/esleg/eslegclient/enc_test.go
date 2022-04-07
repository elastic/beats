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

package eslegclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/monitoring/report"
)

func TestJSONEncoderMarshalBeatEvent(t *testing.T) {
	encoder := NewJSONEncoder(nil, true)
	event := beat.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: common.MapStr{
			"field1": "value1",
		},
	}

	err := encoder.Marshal(event)
	if err != nil {
		t.Errorf("Error while marshaling beat.Event using JSONEncoder: %v", err)
	}
	assert.Equal(t, encoder.buf.String(), "{\"@timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n",
		"Unexpected marshaled format of beat.Event")
}

func TestJSONEncoderMarshalMonitoringEvent(t *testing.T) {
	encoder := NewJSONEncoder(nil, true)
	event := report.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: common.MapStr{
			"field1": "value1",
		},
	}

	err := encoder.Marshal(event)
	if err != nil {
		t.Errorf("Error while marshaling report.Event using JSONEncoder: %v", err)
	}
	assert.Equal(t, encoder.buf.String(), "{\"timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n",
		"Unexpected marshaled format of report.Event")
}
