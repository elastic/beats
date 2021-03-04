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

package add_data_stream_index

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestAddDataStreamIndex(t *testing.T) {
	simpleDs := DataStream{
		"myns",
		"myds",
		"mytype",
	}
	tests := []struct {
		name    string
		ds      DataStream
		event   *beat.Event
		want    string
		wantErr bool
	}{
		{
			"simple",
			simpleDs,
			&beat.Event{},
			"mytype-myds-myns",
			false,
		},
		{
			"existing meta",
			simpleDs,
			&beat.Event{Meta: common.MapStr{}},
			"mytype-myds-myns",
			false,
		},
		{
			"custom ds",
			simpleDs,
			&beat.Event{Meta: common.MapStr{
				FieldMetaCustomDataset: "custom-ds",
			}},
			"mytype-custom-ds-myns",
			false,
		},
		{
			"defaults ds/ns",
			DataStream{
				Type: "mytype",
			},
			&beat.Event{},
			"mytype-generic-default",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.ds)
			got, err := p.Run(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got.Meta[events.FieldMetaRawIndex])
		})
	}
}
