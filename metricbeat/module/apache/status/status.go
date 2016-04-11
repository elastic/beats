// Reads server status from apache host under /server-status?auto mod_status is required.
package status

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	_ "github.com/elastic/beats/metricbeat/module/apache"
)

const AUTO_STRING = "?auto"

func init() {
	helper.Registry.AddMetricSeter("apache", "status", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{}
}

type MetricSeter struct {
	ServerStatusPath string
	Username         string
	Password         string
	Authentication   bool
}

// Setup any metric specific configuration
func (m *MetricSeter) Setup(ms *helper.MetricSet) error {

	// Additional configuration options
	config := struct {
		ServerStatusPath string `config:"server_status_path"`
		Username         string `config:"username"`
		Password         string `config:"password"`
	}{
		ServerStatusPath: "server-status",
		Username:         "",
		Password:         "",
	}

	if err := ms.Module.ProcessConfig(&config); err != nil {
		return err
	}

	m.ServerStatusPath = config.ServerStatusPath
	m.Username = config.Username
	m.Password = config.Password
	if m.Password != "" && m.Username != "" {
		m.Authentication = true
	} else {
		m.Authentication = false
	}

	return nil
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (event common.MapStr, err error) {

	u, err := url.Parse(host + m.ServerStatusPath)
	if err != nil {
		logp.Err("Invalid Apache HTTPD server-status page: %v", err)
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", u.String()+AUTO_STRING, nil)

	if m.Authentication {
		req.SetBasicAuth(m.Username, m.Password)
	}
	resp, err := client.Do(req)

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("Error during Request: %s", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP Error %d: %s", resp.StatusCode, resp.Status)
	}

	return eventMapping(resp.Body, u.Host, ms.Name), nil
}
