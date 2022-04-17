// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/menderesk/beats/v7/heartbeat/monitors/wrappers"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/go-lookslike"
	"github.com/menderesk/go-lookslike/testslike"

	"github.com/stretchr/testify/require"
)

func TestSynthEventTimestamp(t *testing.T) {
	se := SynthEvent{TimestampEpochMicros: 1000} // 1ms
	require.Equal(t, time.Unix(0, int64(time.Millisecond)), se.Timestamp())
}

func TestToMap(t *testing.T) {
	testUrl, _ := url.Parse("http://testurl")

	type testCase struct {
		name     string
		source   common.MapStr
		expected common.MapStr
	}

	testCases := []testCase{
		{
			"root fields with URL",
			common.MapStr{
				"type":            "journey/start",
				"package_version": "1.2.3",
				"root_fields": map[string]interface{}{
					"synthetics": map[string]interface{}{
						"nested": "v1",
					},
					"truly_at_root": "v2",
				},
				"url": testUrl.String(),
			},
			common.MapStr{
				"synthetics": common.MapStr{
					"type":            "journey/start",
					"package_version": "1.2.3",
					"nested":          "v1",
				},
				"url":           wrappers.URLFields(testUrl),
				"truly_at_root": "v2",
			},
		},
		{
			"root fields, step metadata",
			common.MapStr{
				"type":            "step/start",
				"package_version": "1.2.3",
				"journey":         common.MapStr{"name": "MyJourney", "id": "MyJourney", "tags": []string{"foo"}},
				"step":            common.MapStr{"name": "MyStep", "status": "success", "index": 42, "duration": common.MapStr{"us": int64(1232131)}},
				"root_fields": map[string]interface{}{
					"synthetics": map[string]interface{}{
						"nested": "v1",
					},
					"truly_at_root": "v2",
				},
			},
			common.MapStr{
				"synthetics": common.MapStr{
					"type":            "step/start",
					"package_version": "1.2.3",
					"nested":          "v1",
					"journey":         common.MapStr{"name": "MyJourney", "id": "MyJourney", "tags": []string{"foo"}},
					"step":            common.MapStr{"name": "MyStep", "status": "success", "index": 42, "duration": common.MapStr{"us": int64(1232131)}},
				},
				"truly_at_root": "v2",
			},
		},
		{
			"weird error, and blob, no URL",
			common.MapStr{
				"type":            "someType",
				"package_version": "1.2.3",
				"journey":         common.MapStr{"name": "MyJourney", "id": "MyJourney"},
				"step":            common.MapStr{"name": "MyStep", "index": 42, "status": "down", "duration": common.MapStr{"us": int64(1000)}},
				"error": common.MapStr{
					"name":    "MyErrorName",
					"message": "MyErrorMessage",
					"stack":   "MyErrorStack",
				},
				"blob":      "ablob",
				"blob_mime": "application/weird",
			},
			common.MapStr{
				"synthetics": common.MapStr{
					"type":            "someType",
					"package_version": "1.2.3",
					"journey":         common.MapStr{"name": "MyJourney", "id": "MyJourney"},
					"step":            common.MapStr{"name": "MyStep", "index": 42, "status": "down", "duration": common.MapStr{"us": int64(1000)}},
					"error": common.MapStr{
						"name":    "MyErrorName",
						"message": "MyErrorMessage",
						"stack":   "MyErrorStack",
					},
					"blob":      "ablob",
					"blob_mime": "application/weird",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Actually marshal to JSON and back to test the struct tags for deserialization from JSON
			jsonBytes, err := json.Marshal(tc.source)
			require.NoError(t, err)
			se := &SynthEvent{}
			err = json.Unmarshal(jsonBytes, se)
			require.NoError(t, err)

			m := se.ToMap()

			// Index will always be zero in thee tests, so helpfully include it
			llvalidator := lookslike.Strict(lookslike.Compose(
				lookslike.MustCompile(tc.expected),
				lookslike.MustCompile(common.MapStr{"synthetics": common.MapStr{"index": 0}}),
			))

			// Test that even deep maps merge correctly
			testslike.Test(t, llvalidator, m)
		})
	}
}
