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

package add_locale

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestExportTimezone(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"format": "abbreviation",
	})
	if err != nil {
		t.Fatal(err)
	}

	input := mapstr.M{}

	zone, _ := time.Now().In(time.Local).Zone()

	actual := getActualValue(t, testConfig, input)

	expected := mapstr.M{
		"event": map[string]string{
			"timezone": zone,
		},
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestTimezoneFormat(t *testing.T) {
	// Test positive format

	posLoc, err := time.LoadLocation("Africa/Asmara")
	if err != nil {
		t.Fatal(err)
	}

	posZone, posOffset := time.Now().In(posLoc).Zone()

	posAddLocal := &addLocale{TimezoneFormat: Offset}

	posVal := posAddLocal.Format(posZone, posOffset)

	assert.Regexp(t, `\+[\d]{2}\:[\d]{2}`, posVal)

	// Test negative format

	negLoc, err := time.LoadLocation("America/Curacao")
	if err != nil {
		t.Fatal(err)
	}

	negZone, negOffset := time.Now().In(negLoc).Zone()

	negAddLocal := &addLocale{TimezoneFormat: Offset}

	negVal := negAddLocal.Format(negZone, negOffset)

	assert.Regexp(t, `\-[\d]{2}\:[\d]{2}`, negVal)
}

func TestTimezoneCacheRefreshOnChange(t *testing.T) {
	loc := &addLocale{TimezoneFormat: Offset}

	// Pre-populate the cache with the current zone/offset so Run won't
	// refresh it, then confirm the cache pointer is preserved across calls.
	zone, offset := time.Now().Zone()
	loc.cache.Store(&tzEntry{zone: zone, offset: offset, boxedFormat: loc.Format(zone, offset)})
	first := loc.cache.Load()

	for i := 0; i < 5; i++ {
		_, err := loc.Run(&beat.Event{Fields: mapstr.M{}})
		assert.NoError(t, err, "Run should not error")
	}
	assert.Same(t, first, loc.cache.Load(), "cache entry must be reused when zone/offset are unchanged")

	// A change in zone/offset must invalidate the cache.
	loc.cache.Store(&tzEntry{zone: "STALE", offset: offset + 1, boxedFormat: "stale"})
	_, err := loc.Run(&beat.Event{Fields: mapstr.M{}})
	assert.NoError(t, err, "Run should not error")
	refreshed := loc.cache.Load()
	assert.NotEqual(t, "stale", refreshed.boxedFormat, "cache must be refreshed when zone/offset change")
	assert.Equal(t, zone, refreshed.zone, "cache should reflect current zone")
	assert.Equal(t, offset, refreshed.offset, "cache should reflect current offset")
}

func getActualValue(t *testing.T, config *config.C, input mapstr.M) mapstr.M {
	log := logptest.NewTestingLogger(t, "add_locale_test")
	p, err := New(config, log)
	if err != nil {
		log.Error("Error initializing add_locale")
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: input})
	if err != nil {
		log.Error("Error running add_locale processor")
		t.Fatal(err)
	}

	return actual.Fields
}

func BenchmarkConstruct(b *testing.B) {
	var testConfig = config.NewConfig()

	input := mapstr.M{}

	p, err := New(testConfig, logp.NewNopLogger())
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if _, err = p.Run(&beat.Event{Fields: input}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRunParallel exercises the cache hot path from many goroutines so
// the cost of the cache synchronization (atomic load vs. mutex) is visible.
func BenchmarkRunParallel(b *testing.B) {
	p, err := New(config.NewConfig(), logp.NewNopLogger())
	if err != nil {
		b.Fatal(err)
	}
	// Prime the cache so every iteration takes the hit path.
	if _, err := p.Run(&beat.Event{Fields: mapstr.M{}}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := p.Run(&beat.Event{Fields: mapstr.M{}}); err != nil {
				b.Fatal(err)
			}
		}
	})
}
