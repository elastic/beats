// Reads server status from apache host under /server-status?auto mod_status is required.
package status

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	_ "github.com/elastic/beats/metricbeat/module/apache"
	"github.com/urso/ucfg"
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
func (m *MetricSeter) Setup(cfg *ucfg.Config) error {
	return nil
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {

	hosts := ms.Config.Hosts

	for _, host := range hosts {
		resp, err := http.Get(host + "server-status?auto")

		if resp == nil {
			continue
		}
		defer resp.Body.Close()

		if err != nil {
			logp.Err("Error during Request: %s", err)
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP Error %s: %s", resp.StatusCode, resp.Status)
		}

		event := eventMapping(resp.Body)
		events = append(events, event)
	}

	return events, nil
}
