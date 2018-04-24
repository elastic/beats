package stats

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
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
	mb.Registry.MustAddMetricSet("prometheus", "stats", New,
		mb.WithHostParser(hostParser),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The prometheus stats metricset is beta")

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
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
