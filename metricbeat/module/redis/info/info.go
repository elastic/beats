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
	MetricSet.Register()
}

// Metric object
var MetricSet = helper.NewMetricSet("info", MetricSeter{}, redis.Module)

var Config = &MetricSetConfig{}

type MetricSetConfig struct {
}

type MetricSeter struct {
	Name   string
	Config MetricSetConfig
}

func (m MetricSeter) Setup() error {
	// Loads module config
	// This is module specific config object
	MetricSet.LoadConfig(&Config)
	return nil
}

func (m MetricSeter) Fetch() (events []common.MapStr, err error) {

	hosts := MetricSet.Module.GetHosts()

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
