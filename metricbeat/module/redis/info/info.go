package info

import (
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	rd "github.com/garyburd/redigo/redis"
)

var (
	debugf = logp.MakeDebug("redis-info")
)

func init() {
	if err := mb.Registry.AddMetricSet("redis", "info", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	mb.BaseMetricSet
	pool *rd.Pool
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Network  string `config:"network"`
		MaxConn  int    `config:"maxconn" validate:"min=1"`
		Password string `config:"password"`
	}{
		Network:  "tcp",
		MaxConn:  10,
		Password: "",
	}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		pool: createPool(base.Host(), config.Password, config.Network,
			config.MaxConn, base.Module().Config().Timeout),
	}, nil
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

// Fetch fetches metrics from Redis by issuing the INFO command.
func (m *MetricSet) Fetch() (events common.MapStr, err error) {
	// Fetch default INFO.
	info, err := m.fetchRedisStats("default")
	if err != nil {
		return nil, err
	}

	debugf("Redis INFO from %s: %+v", m.Host(), info)
	return eventMapping(info), nil
}

// fetchRedisStats returns a map of requested stats
func (m *MetricSet) fetchRedisStats(stat string) (map[string]string, error) {
	c := m.pool.Get()
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
