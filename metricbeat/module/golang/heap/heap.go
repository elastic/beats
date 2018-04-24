package heap

import (
	"encoding/json"
	"runtime"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/golang"
)

const (
	defaultScheme = "http"
	defaultPath   = "/debug/vars"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "heap.path",
	}.Build()
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("golang", "heap", New, hostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	http      *helper.HTTP
	lastNumGC uint32
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The golang heap metricset is beta")

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	data, err := m.http.FetchContent()
	if err != nil {
		return nil, err
	}

	stats := struct {
		MemStats runtime.MemStats
		Cmdline  []interface{}
	}{}

	err = json.Unmarshal(data, &stats)
	if err != nil {
		return nil, err
	}

	var event = common.MapStr{
		"cmdline": golang.GetCmdStr(stats.Cmdline),
	}

	ms := &stats.MemStats

	// add heap summary
	event["allocations"] = common.MapStr{
		"mallocs": ms.Mallocs,
		"frees":   ms.Frees,
		"objects": ms.HeapObjects,

		// byte counters
		"total":     ms.TotalAlloc,
		"allocated": ms.HeapAlloc,
		"idle":      ms.HeapIdle,
		"active":    ms.HeapInuse,
	}

	event["system"] = common.MapStr{
		"total":    ms.Sys,
		"obtained": ms.HeapSys,
		"stack":    ms.StackSys,
		"released": ms.HeapReleased,
	}

	// garbage collector summary
	var duration, maxDuration, avgDuration, count uint64
	// collect last gc run stats
	if m.lastNumGC < ms.NumGC {
		delta := ms.NumGC - m.lastNumGC
		start := m.lastNumGC
		if delta > 256 {
			logp.Debug("golang", "Missing %v gc cycles", delta-256)
			start = ms.NumGC - 256
			delta = 256
		}

		end := start + delta
		for i := start; i < end; i++ {
			idx := i % 256
			d := ms.PauseNs[idx]
			count++
			duration += d
			if d > maxDuration {
				maxDuration = d
			}
		}

		avgDuration = duration / count
		m.lastNumGC = ms.NumGC
	}

	event["gc"] = common.MapStr{
		"next_gc_limit": ms.NextGC,
		"total_count":   ms.NumGC,
		"cpu_fraction":  ms.GCCPUFraction,
		"total_pause": common.MapStr{
			"ns": ms.PauseTotalNs,
		},
		"pause": common.MapStr{
			"count": count,
			"sum": common.MapStr{
				"ns": duration,
			},
			"avg": common.MapStr{
				"ns": avgDuration,
			},
			"max": common.MapStr{
				"ns": maxDuration,
			},
		},
	}

	return event, nil
}
