package memprof

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

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
	gopath *regexp.Regexp
}

var defaultGoPath = regexp.MustCompile(`(?U)^.*/src/`)

// New creates a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		ProfPath string         `config:"prof_path"`
		GoPath   *regexp.Regexp `config:"gopath"`
	}{
		ProfPath: "/debug/pprof/heap?debug=1",
		GoPath:   defaultGoPath,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	url := "http://" + base.Host() + config.ProfPath
	return &MetricSet{
		BaseMetricSet: base,
		url:           url,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
		gopath:        config.GoPath,
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
		"summary": valueStats(SumSamples(profile.Sample)),
	})

	for _, f := range CollectFunctionStats(profile) {
		pkg, name := splitName(m.gopath, f.Function.Name)
		file := filepath.Base(f.Function.File)

		// per function allocation summaries
		emit(common.MapStr{
			"function": common.MapStr{
				"package": pkg,
				"name":    name,
				"file":    file,

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
					"package":  pkg,
					"function": name,
					"file":     file,
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
			foPkg, foName := splitName(m.gopath, fo.Name)

			emit(common.MapStr{
				"edge_allocation": common.MapStr{
					"parent": common.MapStr{
						"package":  pkg,
						"function": name,
						"file":     file,
					},
					"child": common.MapStr{
						"package":  foPkg,
						"function": foName,
						"file":     filepath.Base(fo.File),
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

func splitName(gopath *regexp.Regexp, fullName string) (string, string) {
	idx := strings.LastIndex(fullName, "/")
	if idx < 0 {
		idx = 0
	}

	path := idx
	idx = strings.Index(fullName[idx:], ".")
	if idx < 0 {
		return "", fullName
	}

	idx += path
	pkg, name := fullName[:idx], fullName[idx+1:]
	return withoutGoPath(gopath, pkg), name
}

func withoutGoPath(gopath *regexp.Regexp, name string) string {
	if gopath == nil {
		return name
	}

	if loc := gopath.FindStringIndex(name); len(loc) == 2 {
		name = name[loc[1]:]
	}
	return name
}
