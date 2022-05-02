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

package log

import (
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/monitoring/report"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// List of metrics that are gauges. This is used to identify metrics that should
// not be reported as deltas. Instead we log the raw value if there was any
// observable change during the interval.
//
// TODO: Replace this with a proper solution that uses the metric type from
// where it is defined. See: https://github.com/elastic/beats/issues/5433
var gauges = map[string]bool{
	"libbeat.output.events.active":   true,
	"libbeat.pipeline.events.active": true,
	"libbeat.pipeline.clients":       true,
	"libbeat.config.module.running":  true,
	"registrar.states.current":       true,
	"filebeat.harvester.running":     true,
	"filebeat.harvester.open_files":  true,
	"beat.memstats.memory_total":     true,
	"beat.memstats.memory_alloc":     true,
	"beat.memstats.rss":              true,
	"beat.memstats.gc_next":          true,
	"beat.info.uptime.ms":            true,
	"beat.cpu.user.ticks":            true,
	"beat.cpu.system.ticks":          true,
	"beat.cpu.total.value":           true,
	"beat.cpu.total.ticks":           true,
	"beat.handles.open":              true,
	"beat.handles.limit.hard":        true,
	"beat.handles.limit.soft":        true,
	"beat.runtime.goroutines":        true,
	"system.load.1":                  true,
	"system.load.5":                  true,
	"system.load.15":                 true,
	"system.load.norm.1":             true,
	"system.load.norm.5":             true,
	"system.load.norm.15":            true,
}

// isGauge returns true when the given metric key name represents a gauge value.
// Any metric name suffixed in '_gauge' or containing '.histogram.' is
// treated as a gauge. Other metrics can specifically be marked as gauges
// through the list maintained in this package.
func isGauge(key string) bool {
	if strings.HasSuffix(key, "_gauge") || strings.Contains(key, ".histogram.") {
		return true
	}
	_, found := gauges[key]
	return found
}

// TODO: Change this when gauges are refactored, too.
var strConsts = map[string]bool{
	"beat.info.ephemeral_id": true,
	"beat.info.version":      true,
}

var (
	// StartTime is the time that the process was started.
	StartTime = time.Now()
)

type reporter struct {
	config
	wg         sync.WaitGroup
	done       chan struct{}
	registries map[string]*monitoring.Registry

	// output
	logger *logp.Logger
}

// MakeReporter returns a new Reporter that periodically reports metrics via
// logp. If cfg is nil defaults will be used.
func MakeReporter(beat beat.Info, cfg *conf.C) (report.Reporter, error) {
	config := defaultConfig()
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	r := &reporter{
		config:     config,
		done:       make(chan struct{}),
		logger:     logp.NewLogger("monitoring"),
		registries: map[string]*monitoring.Registry{},
	}

	for _, ns := range r.config.Namespaces {
		reg := monitoring.GetNamespace(ns).GetRegistry()

		// That 'stats' namespace is reported as 'metrics' in the Elasticsearch
		// reporter so use the same name for consistency.
		if ns == "stats" {
			ns = "metrics"
		}
		r.registries[ns] = reg
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.snapshotLoop()
	}()
	return r, nil
}

func (r *reporter) Stop() {
	close(r.done)
	r.wg.Wait()
}

func (r *reporter) snapshotLoop() {
	r.logger.Infof("Starting metrics logging every %v", r.Period)
	defer r.logger.Infof("Stopping metrics logging.")
	defer func() {
		snaps := map[string]monitoring.FlatSnapshot{}
		for name, reg := range r.registries {
			snap := makeSnapshot(reg)
			snaps[name] = snap
		}
		r.logTotals(snaps)
	}()

	ticker := time.NewTicker(r.Period)
	defer ticker.Stop()

	lastSnaps := map[string]monitoring.FlatSnapshot{}
	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
		}

		snaps := make(map[string]monitoring.FlatSnapshot, len(r.registries))
		for name, reg := range r.registries {
			snap := makeSnapshot(reg)
			lastSnap := lastSnaps[name]
			lastSnaps[name] = snap
			delta := makeDeltaSnapshot(lastSnap, snap)
			snaps[name] = delta
		}

		r.logSnapshot(snaps)
	}
}

func (r *reporter) logSnapshot(snaps map[string]monitoring.FlatSnapshot) {
	var snapsLen int
	for _, s := range snaps {
		snapsLen += snapshotLen(s)
	}

	if snapsLen > 0 {
		r.logger.Infow("Non-zero metrics in the last "+r.Period.String(), toKeyValuePairs(snaps)...)
		return
	}

	r.logger.Infof("No non-zero metrics in the last %v", r.Period)
}

func (r *reporter) logTotals(snaps map[string]monitoring.FlatSnapshot) {
	r.logger.Infow("Total metrics", toKeyValuePairs(snaps)...)
	r.logger.Infof("Uptime: %v", time.Since(StartTime))
}

func makeSnapshot(R *monitoring.Registry) monitoring.FlatSnapshot {
	mode := monitoring.Full
	return monitoring.CollectFlatSnapshot(R, mode, false)
}

func makeDeltaSnapshot(prev, cur monitoring.FlatSnapshot) monitoring.FlatSnapshot {
	delta := monitoring.MakeFlatSnapshot()

	for k, b := range cur.Bools {
		if p, ok := prev.Bools[k]; !ok || p != b {
			delta.Bools[k] = b
		}
	}

	for k, s := range cur.Strings {
		if _, found := strConsts[k]; found {
			delta.Strings[k] = s
		} else if p, ok := prev.Strings[k]; !ok || p != s {
			delta.Strings[k] = s
		}
	}

	for k, i := range cur.Ints {
		if isGauge(k) {
			delta.Ints[k] = i
		} else {
			if p := prev.Ints[k]; p != i {
				delta.Ints[k] = i - p
			}
		}
	}

	for k, f := range cur.Floats {
		if isGauge(k) {
			delta.Floats[k] = f
		} else if p := prev.Floats[k]; p != f {
			delta.Floats[k] = f - p
		}
	}

	return delta
}

func snapshotLen(s monitoring.FlatSnapshot) int {
	return len(s.Bools) + len(s.Floats) + len(s.Ints) + len(s.Strings)
}

func toKeyValuePairs(snaps map[string]monitoring.FlatSnapshot) []interface{} {
	args := []interface{}{logp.Namespace("monitoring")}

	for name, snap := range snaps {
		data := make(mapstr.M, snapshotLen(snap))
		for k, v := range snap.Bools {
			data.Put(k, v)
		}
		for k, v := range snap.Floats {
			data.Put(k, v)
		}
		for k, v := range snap.Ints {
			data.Put(k, v)
		}
		for k, v := range snap.Strings {
			data.Put(k, v)
		}
		if len(data) > 0 {
			args = append(args, logp.Reflect(name, data))
		}
	}

	return args
}
