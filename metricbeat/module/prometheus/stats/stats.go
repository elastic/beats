package stats

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

func init() {
	if err := mb.Registry.AddMetricSet("prometheus", "stats", New, hostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("BETA: The prometheus stats metricset is beta")

	return &MetricSet{
		BaseMetricSet: base,
		http:          helper.NewHTTP(base),
	}, nil
}

func (m *MetricSet) Fetch() (common.MapStr, error) {

	scanner, err := m.http.FetchScanner()
	if err != nil {
		return nil, err
	}

	entries := map[string]interface{}{}

	// Iterate through all events to gather data
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and calculated lines
		if line[0] == '#' || strings.Contains(line, "quantile=") {
			continue
		}

		splitPos := strings.LastIndex(line, " ")
		split := []string{line[:splitPos], line[splitPos+1:]}

		entries[split[0]] = split[1]
	}

	data, err := eventMapping(entries)

	return data, err
}
