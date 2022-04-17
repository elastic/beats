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

package remote_write

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// TestGenerateEventsCounter tests counter simple cases
func TestGenerateEventsCounter(t *testing.T) {
	g := remoteWriteEventGenerator{}

	timestamp := model.Time(424242)
	timestamp1 := model.Time(424243)
	labels := common.MapStr{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp1,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := common.MapStr{
		"metrics": common.MapStr{
			"net_conntrack_listener_conn_closed_total": float64(42),
		},
		"labels": labels,
	}
	expected1 := common.MapStr{
		"metrics": common.MapStr{
			"net_conntrack_listener_conn_closed_total": float64(43),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 2)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	assert.EqualValues(t, e.Timestamp, timestamp.Time())
	e = events[labels.String()+timestamp1.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	assert.EqualValues(t, e.Timestamp, timestamp1.Time())
}
