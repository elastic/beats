package stat

import (
	//"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/haproxy"
	//"net"
)

const (
	// defaultSocket is the default path to the unix socket tfor stats on haproxy.
	statsMethod = "stat"
	defaultAddr = "unix:///var/lib/haproxy/stats"
	//defaultHttpPath = "http://localhost:8000/haproxy?stats;csv"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("haproxy", statsMethod, New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	//statsMethod string
	//statsPath   string
	statsAddr string
	counter   int
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		//StatsMethod string `config:"stats_method"`
		//StatsPath   string `config:"stats_path"`
		StatsAddr string `config:"stats_addr"`
	}{
		//StatsMethod: "unix_socket",
		//StatsPath:   defaultSocket,
		StatsAddr: defaultAddr,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		//statsMethod:   config.StatsMethod,
		//statsPath:     config.StatsPath,
		statsAddr: config.StatsAddr,
		counter:   1,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	//var metricSetSlice []common.MapStr

	/*
		if m.statsMethod == "unix_socket" {

			m.counter++

			c, err := net.Dial("unix", m.statsPath)
			if err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("HAProxy %s error: %s", statsMethod, err))
			}
			defer c.Close()

			// Write the command to the socket
			_, err = c.Write([]byte(fmt.Sprintf("show %s\n", statsMethod)))
			if err != nil {
				return nil, fmt.Errorf("Socket write error: %s", err)
			}

			// Now read from the socket
			buf := make([]byte, 2048)
			for {
				_, err := c.Read(buf[:])
				if err != nil {
					return nil, err
				}
				return eventMapping(parseResponse(buf)), nil
			}

		} else {
			// Get the data from the HTTP URI
			m.counter++

		}

		return nil, errors.New("Error getting HAProxy stat")
	*/

	hapc, err := haproxy.NewHaproxyClient(m.statsAddr)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("HAProxy Client error: %s", err))
	}

	res, err := hapc.GetStat()

	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("HAProxy Client error fetching %s: %s", statsMethod, err))
	}
	m.counter++

	return eventMapping(res), nil

}
