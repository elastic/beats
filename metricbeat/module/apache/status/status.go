// Package status reads Apache HTTPD server status from the mod_status module.
package status

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/joeshaw/multierror"
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
	helper.Registry.AddMetricSeter("apache", "status", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{}
}

type MetricSeter struct {
	URLs map[string]string // Map of host to endpoint URL.
}

// Setup the URLs to the mod_status endpoints.
func (m *MetricSeter) Setup(ms *helper.MetricSet) error {
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

	if err := ms.Module.ProcessConfig(&config); err != nil {
		return err
	}

	m.URLs = make(map[string]string, len(ms.Config.Hosts))

	// Parse the config, create URLs, and check for errors.
	var errs multierror.Errors
	for _, host := range ms.Config.Hosts {
		u, err := getURL(config.Username, config.Password, config.ServerStatusPath, host)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m.URLs[host] = u.String()
		debugf("apache-status URL=%s", redactPassword(*u))
	}

	return errs.Err()
}

// Fetch makes an HTTP request to fetch status metrics from the mod_status
// endpoint.
func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (event common.MapStr, err error) {
	url, ok := m.URLs[host]
	if !ok {
		return nil, fmt.Errorf("url not found for host '%s'", host)
	}

	req, err := http.NewRequest("GET", url, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return eventMapping(resp.Body, host, ms.Name), nil
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
		u, err = url.Parse(fmt.Sprintf("%s://%s", "http", rawHost))
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
	query.Set("auto", "")
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
