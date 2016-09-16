package memprof

import (
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/goprof"
	uuid "github.com/satori/go.uuid"
)

// init registers the MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("goprof", "memprof", New); err != nil {
		panic(err)
	}
}

// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	client *http.Client // HTTP client that is reused across requests
	url    string       // httpprof endpoint url
}

// New creates a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		ProfPath string `config:"prof_path"`
	}{
		ProfPath: "/debug/pprof/heap?debug=1",
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	url := "http://" + base.Host() + config.ProfPath
	return &MetricSet{
		BaseMetricSet: base,
		url:           url,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

// Fetch methods implements the data gathering and data conversion.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	profile, err := m.fetchProfile()
	if err != nil {
		return nil, err
	}

	id := uuid.NewV4()

	zero := make([]int64, len(profile.SampleType))
	valueStats := func(values []int64) common.MapStr {
		if len(values) == 0 {
			values = zero
		}
		m := common.MapStr{}
		for i, t := range profile.SampleType {
			m[t.Type] = common.MapStr{t.Unit: values[i]}
		}
		return m
	}

	var events []common.MapStr
	emit := func(e common.MapStr) {
		e["run"] = id
		events = append(events, e)
	}

	// current profile run summary
	emit(common.MapStr{
		"summary": SumSamples(profile.Sample),
	})

	for _, f := range CollectFunctionStats(profile) {
		// per function allocation summaries
		emit(common.MapStr{
			"function": common.MapStr{
				"name": f.Function.Name,
				"file": f.Function.File,

				// mem stats done by function itself
				"self": common.MapStr{
					"stats": valueStats(f.StatsSelf),
				},

				// mem stats including function and children in call graph
				"total": common.MapStr{
					"stats": valueStats(f.StatsTotal),
				},

				// mem stats of children in call graph
				"children": common.MapStr{
					"stats": valueStats(SubValues(f.StatsTotal, f.StatsSelf)),
				},
			},
		})

		// per function direct allocations
		for _, sample := range f.SamplesSelf {
			loc := sample.Locations[0]
			emit(common.MapStr{
				"allocation": common.MapStr{
					"function": f.Function.Name,
					"file":     f.Function.File,
					"line":     loc.Line,
					"address":  loc.Addr,
					"stats":    valueStats(sample.Values),
				},
			})
		}

		// TODO: optional: function indirect allocations being done in call graph
		//       children nodes

		// Edge allocations: allocation stats for current functions children in call graph
		for _, c := range f.Children {
			fo := c.Other.Function
			emit(common.MapStr{
				"edge_allocation": common.MapStr{
					"parent": common.MapStr{
						"function": f.Function.Name,
						"file":     f.Function.File,
					},
					"child": common.MapStr{
						"function": fo.Name,
						"file":     fo.File,
					},
					"stats": valueStats(c.StatsTotal),
				},
			})
		}
	}

	return events, nil
}

func (m *MetricSet) fetchProfile() (*Profile, error) {
	buf, err := goprof.RequestLoad(m.url, m.client)
	if err != nil {
		return nil, err
	}

	return ParseHeap(buf)
}
