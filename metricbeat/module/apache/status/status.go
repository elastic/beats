// Package status reads Apache HTTPD server status from the mod_status module.
package status

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the mod_status endpoint on the
	// Apache HTTPD server.
	defaultPath = "/server-status"

	// autoQueryParam is a query parameter added to the request so that
	// mod_status returns machine-readable output.
	autoQueryParam = "auto"
)

var (
	debugf = logp.MakeDebug("apache-status")

	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "server_status_path",
		DefaultPath:   defaultPath,
		QueryParams:   autoQueryParam,
	}.Build()
)

func init() {
	if err := mb.Registry.AddMetricSet("apache", "status", New, hostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching Apache HTTPD server status.
type MetricSet struct {
	mb.BaseMetricSet
	client *http.Client // HTTP client that is reused across requests.
}

// New creates new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

// Fetch makes an HTTP request to fetch status metrics from the mod_status
// endpoint.
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

	return eventMapping(resp.Body, m.Host()), nil
}
