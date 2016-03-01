// Loads data from the redis info command
package info

import (
	"strings"

	rd "github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
)

func init() {
	helper.Registry.AddMetricSeter("redis", "info", &MetricSeter{})
}

type MetricSeter struct{}

func (m *MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {

	for _, host := range ms.Config.Hosts {

		conn, err := redis.Connect(host)
		if err != nil {
			return nil, err
		}

		out, err := rd.String(conn.Do("INFO"))
		if err != nil {
			logp.Err("Error converting to string: %v", err)
		}

		event := eventMapping(parseRedisInfo(out))
		events = append(events, event)
	}

	return events, nil
}

// parseRedisInfo parses the string returned by the INFO command
// Every line is split up into key and value
func parseRedisInfo(info string) map[string]string {
	// Feed every line into
	result := strings.Split(info, "\r\n")

	// Load redis info values into array
	values := map[string]string{}

	for _, value := range result {
		// Values are separated by :
		parts := strings.Split(value, ":")
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}
	return values
}
