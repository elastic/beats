package collector

import (
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
		PathConfigKey: "metrics_path",
	}.Build()
)

func init() {
	if err := mb.Registry.AddMetricSet("prometheus", "collector", New, hostParser); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	http      *helper.HTTP
	namespace string
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("BETA: The prometheus collector metricset is beta")

	config := struct {
		Namespace string `config:"namespace" validate:"required"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          helper.NewHTTP(base),
		namespace:     config.Namespace,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	scanner, err := m.http.FetchScanner()
	if err != nil {
		return nil, err
	}
	eventList := map[string]common.MapStr{}

	// Iterate through all events to gather data
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comment lines
		if line[0] == '#' {
			continue
		}

		promEvent := NewPromEvent(line)
		if promEvent.value == nil {
			continue
		}

		// If MapString for this label group does not exist yet, it is created
		if _, ok := eventList[promEvent.labelHash]; !ok {
			eventList[promEvent.labelHash] = common.MapStr{}

			// Add labels
			if len(promEvent.labels) > 0 {
				eventList[promEvent.labelHash]["label"] = promEvent.labels
			}

		}
		eventList[promEvent.labelHash][promEvent.key] = promEvent.value
	}

	// Converts hash list to slice
	events := []common.MapStr{}
	for _, e := range eventList {
		e["_namespace"] = m.namespace
		events = append(events, e)
	}

	return events, err
}
