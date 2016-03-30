// Reads server status from nginx host under /server-status, stub_status module is required.
package stubstatus

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
	_ "github.com/elastic/beats/metricbeat/module/nginx"
)

func init() {
	helper.Registry.AddMetricSeter("nginx", "stubstatus", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{requests: 0}
}

type MetricSeter struct {
	ServerStatusPath string
	requests         int
}

// Setup any metric specific configuration
func (m *MetricSeter) Setup(ms *helper.MetricSet) error {

	// Additional configuration options
	config := struct {
		ServerStatusPath string `config:"server_status_path"`
	}{
		ServerStatusPath: "server-status",
	}

	if err := ms.Module.ProcessConfig(&config); err != nil {
		return err
	}

	m.ServerStatusPath = config.ServerStatusPath

	return nil
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (event common.MapStr, err error) {

	u, err := url.Parse(host + m.ServerStatusPath)
	if err != nil {
		logp.Err("Invalid Nginx stub server-status page: %v", err)
		return nil, err
	}

	resp, err := http.Get(u.String())
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

	return eventMapping(m, resp.Body, u.Host, ms.Name), nil
}
