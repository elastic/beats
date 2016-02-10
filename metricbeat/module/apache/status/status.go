// Reads server status from apache host under /server-status?auto mod_status is required.
package status

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/apache"
)

func init() {
	MetricSet.Register()
}

var MetricSet = helper.NewMetricSet("status", MetricSeter{}, apache.Module)

type MetricSeter struct {
}

func (m MetricSeter) Setup() error {
	return nil
}

func (m MetricSeter) Fetch() (events []common.MapStr, err error) {

	hosts := MetricSet.Module.GetHosts()

	for _, host := range hosts {
		resp, err := http.Get(host + "server-status?auto")
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

func (m MetricSeter) Cleanup() error {
	return nil
}
