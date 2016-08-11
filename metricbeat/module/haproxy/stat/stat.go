package stat

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

const (
	// defaultSocket is the default path to the unix socket tfor stats on haproxy.
	statsMethod   = "stat"
	defaultSocket = "/var/lib/haproxy/stats"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("haproxy", "stat", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	statsMethod string
	statsPath   string
	counter     int
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	//config := struct{}{}

	config := struct {
		StatsMethod string `config:"stats_method"`
		StatsPath   string `config:"stats_path"`
	}{
		StatsMethod: "unix_socket",
		StatsPath:   defaultSocket,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		statsMethod:   config.StatsMethod,
		statsPath:     config.StatsPath,
		counter:       1,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	var output []byte
	if m.statsMethod == "unix_socket" {
		c, err := net.Dial("unix", config.StatsSocket)
		buf := make([]byte, 4096)

		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("HAProxy %s error: %s", statsMethod, err))
		}

		_, err = c.Write([]byte(fmt.Sprintf("show %s\n", statsMethod)))
		oputut, err := c.Read(buf)

	} else {

	}

	m.counter++

	return eventMapping(parseResponse(output)), nil
}
