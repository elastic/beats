package info

import (
	"strings"

	rd "github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
)

func init() {
	helper.Registry.AddMetricSeter("redis", "info", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{
		redisPools: map[string]*rd.Pool{},
	}
}

type MetricSeter struct {
	redisPools map[string]*rd.Pool
}

// Configure connection pool for each Redis host
func (m *MetricSeter) Setup(ms *helper.MetricSet) error {

	// Additional configuration options
	config := struct {
		// TODO: Introduce default value for network
		Network string `config:"network"`
		MaxConn int    `config:"maxconn"`
	}{
		Network: "tcp",
		MaxConn: 10,
	}

	if err := ms.Module.ProcessConfig(&config); err != nil {
		return err
	}

	for _, host := range ms.Config.Hosts {

		// Set up redis pool
		redisPool := rd.NewPool(func() (rd.Conn, error) {
			c, err := rd.Dial(config.Network, host)

			if err != nil {
				logp.Err("Failed to create Redis connection pool: %v", err)
				return nil, err
			}

			return c, err
		}, config.MaxConn)

		// TODO: add AUTH
		m.redisPools[host] = redisPool
	}

	return nil
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (events common.MapStr, err error) {

	// Fetch default INFO
	info := m.fetchRedisStats(host, "default")
	event := eventMapping(info)
	return event, nil
}

// fetchRedisStats returns a map of requested stats
func (m *MetricSeter) fetchRedisStats(host string, stat string) map[string]string {
	c := m.redisPools[host].Get()
	out, err := rd.String(c.Do("INFO", stat))
	c.Close()

	if err != nil {
		logp.Err("Error converting to string: %v", err)
	}
	return parseRedisInfo(out)
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
		parts := parseRedisLine(value, ":")
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}
	return values
}

// parseRedisLine parses a single line returned by INFO
func parseRedisLine(s string, delimeter string) []string {
	return strings.Split(s, delimeter)
}
