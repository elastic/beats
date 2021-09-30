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

package opentelemetry

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
)

const (
	opentelementryDefaultHost = "0.0.0.0"
	opentelementryDefaultPort = "4317"
)

type eventInfo struct {
	events []beat.Event
}

func TestOpenTelemetryPublish(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("opentelemetry"))
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())

	tests := []struct {
		title  string
		events []eventInfo
	}{
		{
			"publish single event to test",
			single(common.MapStr{
				"host":          "test-host",
				"message":       id,
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"publish single event with topic from type",
			single(common.MapStr{
				"host":          "test-host",
				"message":       id,
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"publish single event with formating to test topic",
			single(common.MapStr{
				"host":          "test-host",
				"message":       id,
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"batch publish to test topic",
			randMulti(5, 100, common.MapStr{
				"host":          "test-host",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"batch publish to test topic from type",
			randMulti(5, 100, common.MapStr{
				"host":          "test-host",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"batch publish with random partitioner",
			randMulti(1, 10, common.MapStr{
				"host":          "test-host",
				"type":          "log",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"batch publish with round robin partitioner",
			randMulti(1, 10, common.MapStr{
				"host":          "test-host",
				"type":          "log",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			"batch publish with hash partitioner without key (fallback to random)",

			randMulti(1, 10, common.MapStr{
				"host":          "test-host",
				"type":          "log",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with hash partitioner with key",
			randMulti(1, 10, common.MapStr{
				"host":          "test-host",
				"type":          "log",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with fields hash partitioner",
			randMulti(1, 10, common.MapStr{
				"host":          "test-host",
				"type":          "log",
				"cpu.nr":        2,
				"cpu.processor": 23.4545,
			}),
		},
	}

	defaultConfig := map[string]interface{}{
		"hosts":         []string{fmt.Sprintf("%s:%s", opentelementryDefaultHost, opentelementryDefaultPort)},
		"timeout":       "1s",
		"cpu.nr":        2,
		"cpu.processor": 23.4545,
	}

	for i, test := range tests {
		test := test
		name := fmt.Sprintf("run test(%v): %v", i, test.title)

		cfg := makeConfig(t, defaultConfig)

		t.Run(name, func(t *testing.T) {
			grp, err := makeOtel(nil, beat.Info{Beat: "libbeat", IndexPrefix: "testbeat"}, outputs.NewNilObserver(), cfg)
			if err != nil {
				t.Fatal(err)
			}

			output := grp.Clients[0].(*client)
			if err := output.Connect(); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, output.index, "testbeat")
			defer output.Close()

			// publish test events
			var wg sync.WaitGroup
			for i := range test.events {
				batch := outest.NewBatch(test.events[i].events...)
				batch.OnSignal = func(_ outest.BatchSignal) {
					wg.Done()
				}

				wg.Add(1)
				output.Publish(context.Background(), batch)
			}

			// wait for all published batches to be ACKed
			wg.Wait()

		})
	}
}

func makeConfig(t *testing.T, in map[string]interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(in)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func flatten(infos []eventInfo) []beat.Event {
	var out []beat.Event
	for _, info := range infos {
		out = append(out, info.events...)
	}
	return out
}

func single(fields common.MapStr) []eventInfo {
	return []eventInfo{
		{
			events: []beat.Event{
				{Timestamp: time.Now(), Fields: fields},
			},
		},
	}
}

func randMulti(batches, n int, event common.MapStr) []eventInfo {
	var out []eventInfo
	for i := 0; i < batches; i++ {
		var data []beat.Event
		for j := 0; j < n; j++ {
			tmp := common.MapStr{}
			for k, v := range event {
				tmp[k] = v
			}
			tmp["message"] = randString(100)
			data = append(data, beat.Event{Timestamp: time.Now(), Fields: tmp})
		}

		out = append(out, eventInfo{data})
	}
	return out
}

func randString(length int) string {
	return string(randASCIIBytes(length))
}

func randASCIIBytes(length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = randChar()
	}
	return b
}

func randChar() byte {
	start, end := 'a', 'z'
	if rand.Int31n(2) == 1 {
		start, end = 'A', 'Z'
	}
	return byte(rand.Int31n(end-start+1) + start)
}

func TestPublishOT(t *testing.T) {

	redisConfig := map[string]interface{}{
		"hosts": []string{"0.0.0.0:423"},
	}

	testPublishList(t, redisConfig)
}

func testPublishList(t *testing.T, cfg map[string]interface{}) {

	out := newOpenTelemetryTestingOutput(t, cfg)
	_ = out
	//err = sendTestEvents(out, batches, batchSize)
	//assert.NoError(t, err)

}

const (
	testBeatname    = "libbeat"
	testBeatversion = "1.2.3"
	testMetaValue   = "private"
)

func newOpenTelemetryTestingOutput(t *testing.T, cfg map[string]interface{}) outputs.Client {
	config, err := common.NewConfigFrom(cfg)
	if err != nil {
		t.Fatalf("Error reading config: %v", err)
	}

	plugin := outputs.FindFactory("opentelemetry")
	if plugin == nil {
		t.Fatalf("redis output module not registered")
	}

	out, err := plugin(nil, beat.Info{Beat: testBeatname, Version: testBeatversion}, outputs.NewNilObserver(), config)
	if err != nil {
		t.Fatalf("Failed to initialize redis output: %v", err)
	}

	client := out.Clients[0]

	return client
}
