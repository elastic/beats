// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"

	"github.com/stretchr/testify/require"
)

func TestSynthEventTimestamp(t *testing.T) {
	se := SynthEvent{TimestampEpochMicros: 1000} // 1ms
	require.Equal(t, time.Unix(0, int64(time.Millisecond)), se.Timestamp())
}

func TestRootFields(t *testing.T) {
	// Actually marshal to JSON and back to test the struct tags for deserialization from JSON
	source := common.MapStr{
		"type": "journey/start",
		"root_fields": map[string]interface{}{
			"synthetics": map[string]interface{}{
				"nested": "v1",
			},
			"truly_at_root": "v2",
		},
	}
	jsonBytes, err := json.Marshal(source)
	require.NoError(t, err)
	se := &SynthEvent{}
	err = json.Unmarshal(jsonBytes, se)
	require.NoError(t, err)

	m := se.ToMap()

	// Test that even deep maps merge correctly
	testslike.Test(t, lookslike.MustCompile(common.MapStr{
		"synthetics": common.MapStr{
			"type":   "journey/start",
			"nested": "v1",
		},
		"truly_at_root": "v2",
	}), m)
}
