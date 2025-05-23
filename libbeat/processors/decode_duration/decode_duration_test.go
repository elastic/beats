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

package decode_duration

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestDecodeDuration(t *testing.T) {
	cases := []struct {
		Duration time.Duration
		Format   string
		Result   float64
	}{
		{time.Second + time.Millisecond, "", 1001},
		{time.Second + time.Millisecond, "milliseconds", 1001},
		{time.Second + time.Millisecond, "seconds", 1.001},
		{3 * time.Second, "minutes", 0.05},
		{3 * time.Minute, "hours", 0.05},
	}

	for _, testCase := range cases {
		t.Run(fmt.Sprintf("%s format as %s", testCase.Duration, testCase.Format), func(t *testing.T) {
			evt := &beat.Event{Fields: mapstr.M{}}
			c := &decodeDuration{
				config: decodeDurationConfig{
					Field:  "duration",
					Format: testCase.Format,
				},
			}
			if _, err := evt.PutValue("duration", testCase.Duration.String()); err != nil {
				t.Fatal(err)
			}
			evt, err := c.Run(evt)
			if err != nil {
				t.Fatal(err)
			}
			d, err := evt.GetValue("duration")
			if err != nil {
				t.Fatal(err)
			}
			floatD, ok := d.(float64)
			if !ok {
				t.Fatal("result value is not duration")
			}
			floatD = math.Round(floatD*math.Pow10(6)) / math.Pow10(6)
			if floatD != testCase.Result {
				t.Fatalf("test case except: %f, actual: %f", testCase.Result, floatD)
			}
		})
	}
}
