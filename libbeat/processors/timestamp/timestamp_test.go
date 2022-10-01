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

package timestamp

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var expected = time.Date(2015, 3, 7, 11, 6, 39, 0, time.UTC)

func TestParsePatterns(t *testing.T) {
	logp.TestingSetup()

	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = append(c.Layouts, time.ANSIC, time.RFC3339Nano, time.RFC3339)

	p, err := newFromConfig(c)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{Fields: common.MapStr{}}

	for name, format := range map[string]string{
		"ANSIC":       time.ANSIC,
		"RFC3339Nano": time.RFC3339Nano,
		"RFC3339":     time.RFC3339,
	} {
		t.Run(name, func(t *testing.T) {
			evt.Timestamp = time.Time{}
			evt.PutValue("ts", expected.Format(format))

			evt, err = p.Run(evt)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, expected, evt.Timestamp)
		})
	}

	t.Run("UNIX", func(t *testing.T) {
		p.Layouts = []string{"UNIX"}

		epochSec := expected.Unix()
		times := []interface{}{
			epochSec,
			float64(epochSec),
			strconv.FormatInt(epochSec, 10),
			strconv.FormatInt(epochSec, 10) + ".0",
		}

		for _, timeValue := range times {
			evt.Timestamp = time.Time{}
			evt.PutValue("ts", timeValue)

			evt, err = p.Run(evt)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, expected, evt.Timestamp)
		}
	})

	t.Run("UNIX_MS", func(t *testing.T) {
		p.Layouts = []string{"UNIX_MS"}

		epochMs := int64(expected.UnixNano()) / int64(time.Millisecond)
		times := []interface{}{
			epochMs,
			float64(epochMs),
			strconv.FormatInt(epochMs, 10),
			strconv.FormatInt(epochMs, 10) + ".0",
		}

		for _, timeValue := range times {
			evt.Timestamp = time.Time{}
			evt.PutValue("ts", timeValue)

			evt, err = p.Run(evt)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, expected, evt.Timestamp)
		}
	})
}

func TestParseNoYear(t *testing.T) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = append(c.Layouts, time.StampMilli)
	c.Timezone = cfgtype.MustNewTimezone("EST")

	p, err := newFromConfig(c)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{Fields: common.MapStr{
		"ts": "Mar  7 11:06:39.002",
	}}

	evt, err = p.Run(evt)
	if err != nil {
		t.Fatal(err)
	}

	// The current year in the EST timezone is returned since ts does not contain a year.
	EST := p.tz
	yearEST := time.Now().In(EST).Year()
	expected := time.Date(yearEST, 3, 7, 11, 6, 39, int(2*time.Millisecond), EST)

	// The timestamp was parsed as EST but the processor always writes a UTC time value.
	assert.Equal(t, expected.UTC(), evt.Timestamp)
}

func TestIgnoreMissing(t *testing.T) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = append(c.Layouts, time.RFC3339)

	p, err := newFromConfig(c)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{Fields: common.MapStr{}}

	_, err = p.Run(evt)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed to get time field")
	}

	p.IgnoreMissing = true
	_, err = p.Run(evt)
	assert.NoError(t, err)
}

func TestIgnoreFailure(t *testing.T) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = append(c.Layouts, time.RFC3339)

	p, err := newFromConfig(c)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{Fields: common.MapStr{"ts": expected.Format(time.Kitchen)}}

	_, err = p.Run(evt)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed parsing time field")
	}

	p.IgnoreFailure = true
	_, err = p.Run(evt)
	assert.NoError(t, err)
	assert.Zero(t, evt.Timestamp)
}

func TestBuiltInTest(t *testing.T) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = append(c.Layouts, "2006-01-89T15:04:05Z07:00") // Bad format.
	c.TestTimestamps = []string{
		"2015-03-07T11:06:39Z",
	}

	_, err := newFromConfig(c)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed to parse test timestamp")
	}
}

func TestTimezone(t *testing.T) {
	cases := map[string]struct {
		Timezone string
		Expected time.Time
		Error    bool
	}{
		"no timezone": {
			Expected: expected,
		},
		"location label": {
			// Use a location without DST to avoid surprises
			Timezone: "America/Panama",
			Expected: expected.Add(5 * time.Hour),
		},
		"UTC label": {
			Timezone: "Etc/UTC",
			Expected: expected,
		},
		"GMT label": {
			Timezone: "Etc/GMT+2",
			Expected: expected.Add(2 * time.Hour),
		},
		"UTC as standard offset": {
			Timezone: "+0000",
			Expected: expected,
		},
		"standard offset": {
			Timezone: "+0430",
			Expected: expected.Add(-4*time.Hour - 30*time.Minute),
		},
		"hour and minute offset": {
			Timezone: "+03:00",
			Expected: expected.Add(-3 * time.Hour),
		},
		"minute offset": {
			Timezone: "+00:30",
			Expected: expected.Add(-30 * time.Minute),
		},
		"abbreviated hour offset": {
			Timezone: "+04",
			Expected: expected.Add(-4 * time.Hour),
		},
		"negative hour and minute offset": {
			Timezone: "-03:30",
			Expected: expected.Add(3*time.Hour + 30*time.Minute),
		},
		"negative minute offset": {
			Timezone: "-00:30",
			Expected: expected.Add(30 * time.Minute),
		},
		"negative abbreviated hour offset": {
			Timezone: "-04",
			Expected: expected.Add(4 * time.Hour),
		},

		"unsupported UTC representation": {
			Timezone: "Z",
			Error:    true,
		},
		"non-existing location": {
			Timezone: "Equatorial/Kundu",
			Error:    true,
		},
		"incomplete offset": {
			Timezone: "-400",
			Error:    true,
		},
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			config := common.MustNewConfigFrom(map[string]interface{}{
				"field":    "ts",
				"timezone": c.Timezone,
				"layouts":  []string{time.ANSIC},
			})

			processor, err := New(config)
			if c.Error {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			originalTimestamp := expected.Format(time.ANSIC)
			t.Logf("Original timestamp: %+v", originalTimestamp)
			t.Logf("Timezone: %s", c.Timezone)

			event := &beat.Event{
				Fields: common.MapStr{
					"ts": originalTimestamp,
				},
			}

			event, err = processor.Run(event)
			assert.NoError(t, err)
			assert.Equal(t, c.Expected, event.Timestamp)
		})
	}
}

func TestMetadataTarget(t *testing.T) {
	datetime := "2006-01-02T15:04:05Z"
	c := defaultConfig()
	c.Field = "@metadata.time"
	c.TargetField = "@metadata.ts"
	c.Layouts = append(c.Layouts, time.RFC3339)
	c.Timezone = cfgtype.MustNewTimezone("EST")

	p, err := newFromConfig(c)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{
		Meta: common.MapStr{
			"time": datetime,
		},
	}

	newEvt, err := p.Run(evt)
	assert.NoError(t, err)

	expTs, err := time.Parse(time.RFC3339, datetime)
	assert.NoError(t, err)

	expMeta := common.MapStr{
		"time": datetime,
		"ts":   expTs.UTC(),
	}

	assert.Equal(t, expMeta, newEvt.Meta)
	assert.Equal(t, evt.Fields, newEvt.Fields)
	assert.Equal(t, evt.Timestamp, newEvt.Timestamp)
}
