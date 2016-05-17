// Package stubstatus reads server status from nginx host under /server-status, ngx_http_stub_status_module is required.
package stubstatus

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the ngx_http_stub_status_module endpoint on Nginx.
	defaultPath = "/server-status"
)

var (
	debugf = logp.MakeDebug("nginx-status")
)

func init() {
	if err := mb.Registry.AddMetricSet("nginx", "stubstatus", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching Nginx stub status.
type MetricSet struct {
	mb.BaseMetricSet

	client *http.Client // HTTP client that is reused across requests.
	url    string       // Nginx stubstatus endpoint URL.

	requests int
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Additional configuration options
	config := struct {
		ServerStatusPath string `config:"server_status_path"`
	}{
		ServerStatusPath: defaultPath,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	u, err := getURL(config.ServerStatusPath, base.Host())
	if err != nil {
		return nil, err
	}

	debugf("nginx-stubstatus URL=%s", u)
	return &MetricSet{
		BaseMetricSet: base,
		url:           u.String(),
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
		requests:      0,
	}, nil
}

// Fetch makes an HTTP request to fetch status metrics from the stubstatus endpoint.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	req, err := http.NewRequest("GET", m.url, nil)
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return eventMapping(m, resp.Body, m.Host(), m.Name())
}

// getURL constructs a URL from the rawHost value and path if one was not set in the rawHost value.
func getURL(statusPath, rawHost string) (*url.URL, error) {
	u, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("error parsing nginx host: %v", err)
	}

	if u.Scheme == "" {
		// Add scheme and re-parse.
		u, err = url.Parse(fmt.Sprintf("%s://%s", "http", rawHost))
		if err != nil {
			return nil, fmt.Errorf("error parsing nginx host: %v", err)
		}
	}

	if u.Host == "" {
		return nil, fmt.Errorf("error parsing nginx host: empty host")
	}

	if u.Path == "" {
		// The path given in the host config takes precedence over the
		// server_status_path config value.
		path := statusPath
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		u.Path = path
	}

	return u, nil
}
