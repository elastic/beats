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

package instance_test

import (
	"encoding/json"
	"flag"
	"net/http"
	"sync"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/mock"
)

func TestMonitoringNameFromConfig(t *testing.T) {
	mockBeat, _ := mock.New(nil, nil)
	wg := sync.WaitGroup{}
	wg.Add(1)

	// Make sure the beat has stopped before finishing the test
	t.Cleanup(wg.Wait)

	go func() {
		defer wg.Done()

		// Set the configuration file path flag so the beat can read it
		flag.Set("c", "testdata/mockbeat.yml")
		instance.Run(mock.Settings, func(_ *beat.Beat, _ *common.Config) (beat.Beater, error) {
			return mockBeat, nil
		})
	}()

	t.Cleanup(func() {
		mockBeat.Stop()
	})

	resp, err := http.Get("http://localhost:5066/state")
	if err != nil {
		t.Fatal("calling state endpoint: ", err.Error())
	}
	defer resp.Body.Close()

	beatName := struct {
		Beat struct {
			Name string
		}
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&beatName); err != nil {
		t.Fatalf("could not decode response body: %s", err.Error())
	}

	if got, want := beatName.Beat.Name, "TestMonitoringNameFromConfig"; got != want {
		t.Fatalf("expecting '%s', got '%s'", want, got)
	}
}
