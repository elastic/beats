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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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
}

func TestParseNoYear(t *testing.T) {
	c := defaultConfig()
	c.Field = "ts"
	c.Layouts = append(c.Layouts, time.StampMilli)
	c.Timezone = "EST"

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
