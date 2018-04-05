// Package stubstatus reads server status from nginx host under /server-status, ngx_http_stub_status_module is required.
package stubstatus

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the ngx_http_stub_status_module endpoint on Nginx.
	defaultPath = "/server-status"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "server_status_path",
		DefaultPath:   defaultPath,
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("nginx", "stubstatus", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching Nginx stub status.
type MetricSet struct {
	mb.BaseMetricSet
	http                *helper.HTTP
	previousNumRequests int // Total number of requests as returned in the previous fetch.
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
	}, nil
}

// Fetch makes an HTTP request to fetch status metrics from the stubstatus endpoint.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	scanner, err := m.http.FetchScanner()
	if err != nil {
		return nil, err
	}

	return eventMapping(scanner, m)
}
