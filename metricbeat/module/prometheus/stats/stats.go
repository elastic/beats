package stats

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

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
	debugf = logp.MakeDebug("prometheus-stats")

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
	client *http.Client
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The prometheus stats metricset is experimental")

	return &MetricSet{
		BaseMetricSet: base,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

func (m *MetricSet) Fetch() (common.MapStr, error) {

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

	scanner := bufio.NewScanner(resp.Body)

	entries := map[string]interface{}{}

	// Iterate through all events to gather data
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and calculated lines
		if line[0] == '#' || strings.Contains(line, "quantile=") {
			continue
		}
		split := strings.Split(line, " ")
		entries[split[0]] = split[1]
	}

	data, err := eventMapping(entries)

	return data, err
}
