package php_fpm

import (
	"fmt"
	"io"
	"net/http"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/status"
)

// HostParser is used for parsing the configured php-fpm hosts.
var HostParser = parse.URLHostParserBuilder{
	DefaultScheme: defaultScheme,
	DefaultPath:   defaultPath,
	QueryParams:   "json",
	PathConfigKey: "status_path",
}.Build()

// StatsClient provides access to php-fpm stats api
type StatsClient struct {
	address  string
	user     string
	password string
	http     *http.Client
}

// NewStatsClient creates a new StatsClient
func NewStatsClient(m mb.BaseMetricSet) *StatsClient {
	return &StatsClient{
		address:  m.HostData().SanitizedURI,
		user:     m.HostData().User,
		password: m.HostData().Password,
		http:     &http.Client{Timeout: m.Module().Config().Timeout},
	}
}

// Fetch php-fpm stats
func (c *StatsClient) Fetch() (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.address, nil)
	if c.user != "" || c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}
