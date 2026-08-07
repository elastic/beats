// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCapacityFromFlushEventsCountsHistogramsAndBuckets(t *testing.T) {
	events := map[string]mb.Event{
		"k1": {ModuleFields: mapstr.M{
			"labels": mapstr.M{"a": "b"},
			"http_request_duration_seconds": mapstr.M{
				"histogram": mapstr.M{
					"values": []float64{0.1, 0.2, 0.3},
					"counts": []uint64{1, 2, 3},
				},
			},
			"http_request_bytes": mapstr.M{
				"histogram": mapstr.M{
					"values": []float64{1.0},
					"counts": []uint64{1},
				},
			},
		}},
	}
	cap := capacityFromFlushEvents(events)
	assert.Equal(t, 2, cap.histograms)
	assert.Equal(t, 4, cap.buckets)
}
