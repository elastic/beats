// Package status reads Apache HTTPD server status from the mod_status module.
package status

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

	// defaultPath is the default path to the mod_status endpoint on the
	// Apache HTTPD server.
	defaultPath = "/server-status"

	// autoQueryParam is a query parameter added to the request so that
	// mod_status returns machine-readable output.
	autoQueryParam = "auto"
)

var (
	debugf = logp.MakeDebug("apache-status")
)

func init() {
	if err := mb.Registry.AddMetricSet("apache", "status", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching Apache HTTPD server status.
type MetricSet struct {
	mb.BaseMetricSet
	client *http.Client // HTTP client that is reused across requests.
	url    string       // Apache HTTP server status endpoint URL.
}

// New creates new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Additional configuration options
	config := struct {
		ServerStatusPath string `config:"server_status_path"`
		Username         string `config:"username"`
		Password         string `config:"password"`
	}{
		ServerStatusPath: defaultPath,
		Username:         "",
		Password:         "",
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	u, err := getURL(config.Username, config.Password, config.ServerStatusPath, base.Host())
	if err != nil {
		return nil, err
	}

	debugf("apache-status URL=%s", redactPassword(*u))
	return &MetricSet{
		BaseMetricSet: base,
		url:           u.String(),
		client:        &http.Client{Timeout: base.Module().Config().Timeout},
	}, nil
}

// Fetch makes an HTTP request to fetch status metrics from the mod_status
// endpoint.
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

	return eventMapping(resp.Body, m.Host()), nil
}

// getURL constructs a URL from the rawHost value and adds the provided user,
// password, and path if one was not set in the rawHost value.
func getURL(user, password, statusPath, rawHost string) (*url.URL, error) {
	u, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("error parsing apache host: %v", err)
	}

	if u.Scheme == "" {
		// Add scheme and re-parse.
		u, err = url.Parse(fmt.Sprintf("%s://%s", defaultScheme, rawHost))
		if err != nil {
			return nil, fmt.Errorf("error parsing apache host: %v", err)
		}
	}

	if u.User == nil && user != "" {
		// Set username and password if not set in host config.
		u.User = url.UserPassword(user, password)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("error parsing apache host: empty host")
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

	// Add the 'auto' query parameter so that server-status returns
	// machine readable output.
	query := u.Query()
	query.Set(autoQueryParam, "")
	u.RawQuery = query.Encode()

	return u, nil
}

// redactPassword returns the URL as a string with the password redacted.
func redactPassword(u url.URL) string {

	if u.User == nil {
		return u.String()
	}

	u.User = url.User(u.User.Username())
	return u.String()
}
