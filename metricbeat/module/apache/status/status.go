// Reads server status from apache host under /server-status?auto mod_status is required.
package status

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/metricbeat/helper"
	_ "github.com/elastic/beats/metricbeat/module/apache"
)

func init() {
	helper.Registry.AddMetricSeter("apache", "status", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{}
}

type MetricSeter struct{}

// Setup any metric specific configuration
func (m *MetricSeter) Setup(ms *helper.MetricSet) error {
	return nil
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (event common.MapStr, err error) {

	resp, err := http.Get(host + "server-status?auto")
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

	return eventMapping(resp.Body), nil

}
