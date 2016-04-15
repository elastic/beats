package info

import (
	"strings"
	"time"

	rd "github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
)

var (
	debugf = logp.MakeDebug("redis")
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
		Network  string `config:"network"`
		MaxConn  int    `config:"maxconn"`
		Password string `config:"password"`
	}{
		Network:  "tcp",
		MaxConn:  10,
		Password: "",
	}

	if err := ms.Module.ProcessConfig(&config); err != nil {
		return err
	}

	for _, host := range ms.Config.Hosts {
		redisPool := createPool(host, config.Password, config.Network, config.MaxConn, ms.Module.Timeout)
		m.redisPools[host] = redisPool
	}

	return nil
}

func createPool(host, password, network string, maxConn int, timeout time.Duration) *rd.Pool {

	return &rd.Pool{
		MaxIdle:     maxConn,
		IdleTimeout: timeout,
		Dial: func() (rd.Conn, error) {
			c, err := rd.Dial(network, host)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (events common.MapStr, err error) {
	// Fetch default INFO
	info, err := m.fetchRedisStats(host, "default")
	if err != nil {
		return nil, err
	}

	debugf("Redis INFO from %s: %+v", host, info)
	return eventMapping(info), nil
}

// fetchRedisStats returns a map of requested stats
func (m *MetricSeter) fetchRedisStats(host string, stat string) (map[string]string, error) {
	c := m.redisPools[host].Get()
	defer c.Close()
	out, err := rd.String(c.Do("INFO", stat))

	if err != nil {
		logp.Err("Error retrieving INFO stats: %v", err)
		return nil, err
	}
	return parseRedisInfo(out), nil
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
