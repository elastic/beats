package memstats

import (
	"net/http"
	"runtime"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/goprof"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("goprof", "memstats", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	client *http.Client // HTTP client that is reused across requests.
	url    string       // Httpprof endpoint URL.

	lastNumGC uint32
}

// New create a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	// Additional configuration options
	config := struct {
		VarsPath string `config:"vars_path"`
	}{
		VarsPath: "/debug/vars",
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	url := "http://" + base.Host() + config.VarsPath
	return &MetricSet{
		BaseMetricSet: base,
		url:           url,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	stats := struct {
		MemStats runtime.MemStats
	}{}
	err := goprof.RequestInto(&stats, m.url, m.client)
	if err != nil {
		return nil, err
	}

	ms := &stats.MemStats
	var events []common.MapStr

	// add garbage collector summary
	events = append(events, common.MapStr{
		"gc_summary": common.MapStr{
			"next_gc_limit": ms.NextGC,
			"gc_count":      ms.NumGC,
			"gc_total_pause": common.MapStr{
				"ns": ms.PauseTotalNs,
			},
		},
	})

	// add heap summary
	events = append(events, common.MapStr{
		"heap": common.MapStr{
			"allocations": common.MapStr{
				"mallocs": ms.Mallocs,
				"frees":   ms.Frees,
				"objects": ms.HeapObjects,

				// byte counters
				"total":     ms.TotalAlloc,
				"allocated": ms.HeapAlloc,
				"idle":      ms.HeapIdle,
				"active":    ms.HeapInuse,
			},
			"system": common.MapStr{
				"total":    ms.Sys,
				"optained": ms.HeapSys,
				"stack":    ms.StackSys,
				"released": ms.HeapReleased,
			},
		},
	})

	// collect per size class allocation stats
	for _, c := range ms.BySize {
		events = append(events, common.MapStr{
			"allocator": common.MapStr{
				"size":        c.Size,
				"allocations": c.Mallocs,
				"frees":       c.Frees,
			},
		})
	}

	// collect last gc run stats
	if m.lastNumGC < ms.NumGC {
		delta := ms.NumGC - m.lastNumGC
		start := m.lastNumGC
		if delta > 256 {
			logp.Err("Missing %v gc cycles", delta-255)
			start = m.lastNumGC - 255
			delta = 255
		}

		for i := start; i < delta; i++ {
			idx := i % 256
			end := time.Unix(0, 0).Add(time.Duration(ms.PauseEnd[idx]))
			d := ms.PauseNs[idx]
			start := time.Unix(0, 0).Add(time.Duration(ms.PauseEnd[idx] - d))
			events = append(events, common.MapStr{
				"gc_cycle": common.MapStr{
					"run":   i,
					"start": common.Time(start),
					"end":   common.Time(end),
					"duration": common.MapStr{
						"ns": d,
					},
				},
			})
		}

		m.lastNumGC = ms.NumGC
	}

	return events, nil
}
