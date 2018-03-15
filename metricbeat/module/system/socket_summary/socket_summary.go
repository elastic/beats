package socket_summary

import (
	"syscall"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/shirou/gopsutil/net"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "socket_summary", New,
		mb.WithNamespace("system.socket.summary"),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The socket_summary metricset is experimental.")

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

func calculateConnStats(conns []net.ConnectionStat) common.MapStr {
	var (
		tcpConns     = 0
		tcpListening = 0
		udpConns     = 0
	)

	for _, conn := range conns {
		switch conn.Type {
		case syscall.SOCK_STREAM:
			tcpConns++

			if conn.Status == "LISTEN" {
				tcpListening++
			}
		case syscall.SOCK_DGRAM:
			udpConns++
		}
	}

	return common.MapStr{
		"tcp": common.MapStr{
			"connections": tcpConns,
			"listening":   tcpListening,
		},
		"udp": common.MapStr{
			"connections": udpConns,
		},
	}
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {

	conns, err := net.Connections("inet")

	if err != nil {
		report.Error(err)
		return
	}

	report.Event(mb.Event{
		MetricSetFields: calculateConnStats(conns),
	})
}
