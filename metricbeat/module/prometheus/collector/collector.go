package collector

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	debugf = logp.MakeDebug("prometheus-collector")

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
	client    *http.Client
	namespace string
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The prometheus collector metricset is experimental")

	config := struct {
		Namespace string `config:"namespace" validate:"required"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
		namespace:     config.Namespace,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	req, err := http.NewRequest("GET", m.HostData().SanitizedURI, nil)
	if m.HostData().User != "" || m.HostData().Password != "" {
		req.SetBasicAuth(m.HostData().User, m.HostData().Password)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	eventList := map[string]common.MapStr{}
	scanner := bufio.NewScanner(resp.Body)

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
