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

//go:build stresstest

package stress_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline/stress"
	_ "github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// additional flags
var (
	duration time.Duration // -duration <dur>
)

func init() {
	flag.DurationVar(&duration, "duration", 0, "configure max run duration")
}

func TestPipeline(t *testing.T) {
	genConfigs := collectConfigs(t, "configs/gen/*.yml")
	pipelineConfigs := collectConfigs(t, "configs/pipeline/*.yml")
	outConfigs := collectConfigs(t, "configs/out/*.yml")

	info := beat.Info{
		Beat:     "stresser",
		Version:  "0",
		Name:     "stresser.test",
		Hostname: "stresser.test",
		Logger:   logptest.NewTestingLogger(t, ""),
	}

	if duration == 0 {
		duration = 15 * time.Second
	}

	configTest(t, "gen", genConfigs, func(t *testing.T, gen string) {
		configTest(t, "pipeline", pipelineConfigs, func(t *testing.T, pipeline string) {
			configTest(t, "out", outConfigs, func(t *testing.T, out string) {

				// Reset and snapshot the global ack counter so each
				// scenario reports its own throughput.
				stress.AckedEventCount.Store(0)
				start := time.Now()
				if testing.Verbose() {
					fmt.Printf("%v Start stress test %v\n", start.Format(time.RFC3339), t.Name())
				}
				defer func() {
					end := time.Now()
					elapsed := end.Sub(start)
					acked := stress.AckedEventCount.Load()
					rate := float64(acked) / elapsed.Seconds()
					fmt.Printf("STRESS %v: acked=%d duration=%v rate=%.0f events/s\n",
						t.Name(), acked, elapsed.Round(time.Millisecond), rate)
				}()

				config, err := common.LoadFiles(gen, pipeline, out)
				if err != nil {
					t.Fatal(err)
				}

				name := t.Name()
				name = strings.Replace(name, "/", "-", -1)
				name = strings.Replace(name, "\\", "-", -1)

				dir, err := ioutil.TempDir("", "")
				if err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(dir)

				// Merge test info into config object
				config.Merge(map[string]interface{}{
					"test": map[string]interface{}{
						"tmpdir": dir,
						"name":   name,
					},
				})

				// check if the pipeline configuration allows for parallel tests
				onErr := func(err error) {
					t.Error(err)
				}

				if err := stress.RunTests(info, duration, config, onErr); err != nil {
					t.Error("Test failed with:", err)
				}
			})
		})
	})
}

func configTest(t *testing.T, typ string, configs []string, fn func(t *testing.T, config string)) {
	for _, config := range configs {
		config := config
		t.Run(testName(typ, config), func(t *testing.T) {
			t.Parallel()
			fn(t, config)
		})
	}
}

func collectConfigs(t *testing.T, pattern string) []string {
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func testName(typ, path string) string {
	return fmt.Sprintf("%v=%v", typ, filepath.Base(path[:len(path)-4]))
}
