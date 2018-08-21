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

package dns

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

func TestDNSProcessorRun(t *testing.T) {
	p := &processor{
		Config:   defaultConfig,
		resolver: &stubResolver{},
		log:      logp.NewLogger(logName),
	}
	p.Config.reverseFlat = map[string]string{
		"source.ip": "source.domain",
	}
	t.Log(p.String())

	t.Run("default", func(t *testing.T) {
		event, err := p.Run(&beat.Event{
			Fields: common.MapStr{
				"source.ip": gatewayIP,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		v, _ := event.GetValue("source.domain")
		assert.Equal(t, gatewayName, v)
	})

	const forwardDomain = "www." + gatewayName
	t.Run("append", func(t *testing.T) {
		p.Config.Action = ActionAppend

		event, err := p.Run(&beat.Event{
			Fields: common.MapStr{
				"source.ip":     gatewayIP,
				"source.domain": forwardDomain,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		v, _ := event.GetValue("source.domain")
		assert.ElementsMatch(t,
			[]string{gatewayName, forwardDomain},
			v)
	})

	t.Run("replace", func(t *testing.T) {
		p.Config.Action = ActionReplace

		event, err := p.Run(&beat.Event{
			Fields: common.MapStr{
				"source.ip":     gatewayIP,
				"source.domain": forwardDomain,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		v, _ := event.GetValue("source.domain")
		assert.Equal(t, gatewayName, v)
	})
}

func TestDNSProcessorTagOnFailure(t *testing.T) {
	p := &processor{
		Config:   defaultConfig,
		resolver: &stubResolver{},
		log:      logp.NewLogger(logName),
	}
	p.Config.TagOnFailure = []string{"_lookup_failed"}
	p.Config.reverseFlat = map[string]string{
		"source.ip":      "source.domain",
		"destination.ip": "destination.domain",
	}
	t.Log(p.String())

	event, err := p.Run(&beat.Event{
		Fields: common.MapStr{
			"source.ip":      "192.0.2.1",
			"destination.ip": "192.0.2.2",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	v, _ := event.GetValue("tags")
	if assert.Len(t, v, 1) {
		assert.ElementsMatch(t, v, p.Config.TagOnFailure)
	}
}

func TestDNSProcessorRunInParallel(t *testing.T) {
	// This is a simple smoke test to make sure that there are no concurrency
	// issues. It is most effective when run with the race detector.

	conf := defaultConfig
	reg := monitoring.NewRegistry()
	cache, err := NewPTRLookupCache(reg, conf.CacheConfig, &stubResolver{})
	if err != nil {
		t.Fatal(err)
	}
	p := &processor{Config: conf, resolver: cache, log: logp.NewLogger(logName)}
	p.Config.reverseFlat = map[string]string{"source.ip": "source.domain"}

	const numGoroutines = 10
	const numEvents = 500
	var wg sync.WaitGroup

	// Start several goroutines.
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			// Execute processor.
			for i := 0; i < numEvents; i++ {
				_, err := p.Run(&beat.Event{
					Fields: common.MapStr{
						"source.ip": "192.168.0." + strconv.Itoa(i%256),
					},
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	wg.Wait()
}
