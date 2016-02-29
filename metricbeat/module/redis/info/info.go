// Loads data from the redis info command
package info

import (
	"fmt"
	"strings"

	rd "github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
)

func init() {
	helper.Registry.AddMetricSeter("redis", "info", MetricSeter{})
}

type MetricSeter struct{}

func (m MetricSeter) Setup() error {
	return nil
}

func (m MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {

	hosts := ms.Config.Hosts

	fmt.Printf("Hosts: %+v", hosts)

	for _, host := range hosts {

		conn, err := redis.Connect(host)
		if err != nil {
			return nil, err
		}

		out, err := rd.String(conn.Do("INFO"))
		if err != nil {
			logp.Err("Error converting to string: %v", err)
		}

		// Feed every line into
		result := strings.Split(out, "\r\n")

		// Load redis info values into array
		values := map[string]string{}

		for _, value := range result {
			// Values are separated by :
			parts := strings.Split(value, ":")
			if len(parts) == 2 {
				values[parts[0]] = parts[1]
			}
		}

		event := eventMapping(values)

		events = append(events, event)
	}

	return events, nil
}

func (m MetricSeter) Cleanup() error {
	return nil
}
