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

package buffer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/monitoring"
)

// reporter is a struct that will fill a ring buffer for each monitored registry.
type reporter struct {
	config
	wg         sync.WaitGroup
	done       chan struct{}
	registries map[string]*monitoring.Registry

	// ring buffers for namespaces
	entries map[string]*ringBuffer
}

// MakeReporter creates and starts a reporter with the given config.
func MakeReporter(beat beat.Info, cfg *common.Config) (*reporter, error) {
	config := defaultConfig()
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	r := &reporter{
		config:     config,
		done:       make(chan struct{}),
		registries: map[string]*monitoring.Registry{},
		entries:    map[string]*ringBuffer{},
	}

	for _, ns := range r.config.Namespaces {
		reg := monitoring.GetNamespace(ns).GetRegistry()
		r.registries[ns] = reg
		r.entries[ns] = newBuffer(r.config.Size)
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.snapshotLoop()
	}()
	return r, nil
}

// Stop will stop the reporter from collecting new information.
// It will not clear any previously collected data.
func (r *reporter) Stop() {
	close(r.done)
	r.wg.Wait()
}

// snapshotLoop will collect a snapshot for each monitored registry for the configured period and store them in the correct buffer.
func (r *reporter) snapshotLoop() {
	ticker := time.NewTicker(r.config.Period)
	defer ticker.Stop()

	for {
		var ts time.Time
		select {
		case <-r.done:
			return
		case ts = <-ticker.C:
		}

		for name, reg := range r.registries {
			snap := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)
			if _, ok := snap["@timestamp"]; !ok {
				snap["@timestamp"] = ts.UTC()
			}
			r.entries[name].add(snap)
		}
	}
}

// ServeHTTP is an http.Handler that will respond with the monitored registries buffer's contents in JSON.
func (r *reporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	resp := make(map[string][]interface{}, len(r.entries))
	for name, entries := range r.entries {
		resp[name] = entries.getAll()
	}

	p, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to encode JSON response: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(p)
}
