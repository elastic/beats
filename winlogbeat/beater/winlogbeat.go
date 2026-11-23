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

/*
Package beater provides the implementation of the libbeat Beater interface for
Winlogbeat. It runs the Filebeat beater with the winlog input, translating
Winlogbeat's configuration and starting from Winlogbeat's legacy state file.
*/
package beater

import (
	"context"
	"fmt"
	"sync"

	fbbeater "github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/winlog"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/paths"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
	"github.com/elastic/beats/v7/winlogbeat/module"
)

const pipelinesWarning = "Winlogbeat is unable to load the ingest pipelines" +
	" because the Elasticsearch output is not configured/enabled. If you have" +
	" already loaded the ingest pipelines, you can ignore this warning."

// winlogbeat wraps the Filebeat beater, adding Winlogbeat's ingest pipeline
// loading.
type winlogbeat struct {
	inner    beat.Beater // The Filebeat beater.
	settings settings
	log      *logp.Logger

	// registryFile is the resolved path of the legacy state file. It is
	// read in Run so that commands like setup and export do not touch it.
	registryFile string

	mu     sync.Mutex
	legacy map[string]checkpoint.EventLogState
}

// New returns a new Winlogbeat.
func New(b *beat.Beat, _ *conf.C) (beat.Beater, error) {
	s, err := translateConfig(b.BeatConfig)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	if !b.InSetupCmd {
		// Create the event logs only to validate their configuration early.
		for _, cfg := range s.EventLogs {
			_, err := eventlog.New(cfg, b.Info.Logger)
			if err != nil {
				return nil, fmt.Errorf("failed to create new event log: %w", err)
			}
		}
	}

	w := &winlogbeat{
		settings:     s,
		log:          b.Info.Logger,
		registryFile: b.Info.Paths.Resolve(paths.Data, s.RegistryFile),
	}
	tracker := newStopTracker(len(s.EventLogs), func() { go w.Stop() })
	factory := func(info beat.Info, store statestore.States) []v2.Plugin {
		return []v2.Plugin{winlog.PluginWithOptions(info.Logger, store, winlog.Options{
			InitialState: w.initialState,
			OnStopped:    tracker.stopped,
		})}
	}

	b.BeatConfig = s.Filebeat
	w.inner, err = fbbeater.New(factory)(b, s.Filebeat)
	if err != nil {
		return nil, err
	}

	// Set after the Filebeat beater, which installs its own callback.
	b.OverwritePipelinesCallback = func(esConfig *conf.C) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		esClient, err := eslegclient.NewConnectedClient(ctx, esConfig, b.Info)
		if err != nil {
			return err
		}
		_, err = module.UploadPipelines(b.Info, esClient, s.OverwritePipelines)
		return err
	}

	return w, nil
}

// Run loads the legacy state, registers Winlogbeat's ingest pipeline loading
// and runs the Filebeat beater.
func (w *winlogbeat) Run(b *beat.Beat) error {
	err := w.loadLegacyState()
	if err != nil {
		return err
	}

	if beat.SetupPipelinesEnabled(b.BeatConfig) {
		if b.Config.Output.Name() == "elasticsearch" {
			callback := func(esClient *eslegclient.Connection, _ *logp.Logger) error {
				_, err := module.UploadPipelines(b.Info, esClient, w.settings.OverwritePipelines)
				return err
			}
			_, err := elasticsearch.RegisterConnectCallback(callback)
			if err != nil {
				return err
			}
		} else {
			w.log.Warn(pipelinesWarning)
		}
	}
	return w.inner.Run(b)
}

// loadLegacyState reads the state file written by Winlogbeat versions that
// kept their own registry.
func (w *winlogbeat) loadLegacyState() error {
	states, err := checkpoint.Read(w.registryFile)
	if err != nil {
		// The error already names the file.
		return err
	}
	if len(states) != 0 {
		w.log.Infof("Loaded start positions for %d event logs from legacy state file %s", len(states), w.registryFile)
	}
	w.mu.Lock()
	w.legacy = states
	w.mu.Unlock()
	return nil
}

// initialState returns the legacy start position for the named event log.
// It is called from input goroutines.
func (w *winlogbeat) initialState(name string) (checkpoint.EventLogState, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	st, ok := w.legacy[name]
	return st, ok
}

// Stop stops the Filebeat beater.
func (w *winlogbeat) Stop() {
	w.inner.Stop()
}

// stopTracker calls stop once, after stopped has been called for n event logs.
type stopTracker struct {
	once sync.Once
	stop func()

	mu        sync.Mutex
	remaining int
	seen      map[string]bool
}

func newStopTracker(n int, stop func()) *stopTracker {
	return &stopTracker{stop: stop, remaining: n, seen: make(map[string]bool)}
}

// stopped records that the named event log has stopped. Repeated calls for
// the same name count once.
func (t *stopTracker) stopped(name string) {
	t.mu.Lock()
	if !t.seen[name] {
		t.seen[name] = true
		t.remaining--
	}
	done := t.remaining <= 0
	t.mu.Unlock()
	if done {
		t.once.Do(t.stop)
	}
}
