// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package socket_summary

import (
	"syscall"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/net"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "socket_summary", New,
		mb.WithNamespace("system.socket.summary"),
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	sockstat string
	mod      resolve.Resolver
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	sys := base.Module().(resolve.Resolver)
	return &MetricSet{
		mod:           sys,
		BaseMetricSet: base,
	}, nil
}

func calculateConnStats(conns []net.ConnectionStat) common.MapStr {
	var (
		allConns       = len(conns)
		allListening   = 0
		tcpConns       = 0
		tcpListening   = 0
		tcpClosewait   = 0
		tcpEstablished = 0
		tcpTimewait    = 0
		tcpSynsent     = 0
		tcpSynrecv     = 0
		tcpFinwait1    = 0
		tcpFinwait2    = 0
		tcpLastack     = 0
		tcpClosing     = 0
		udpConns       = 0
	)

	for _, conn := range conns {
		if conn.Status == "LISTEN" {
			allListening++
		}
		switch conn.Type {
		case syscall.SOCK_STREAM:
			tcpConns++
			if conn.Status == "ESTABLISHED" {
				tcpEstablished++
			}
			if conn.Status == "CLOSE_WAIT" {
				tcpClosewait++
			}
			if conn.Status == "TIME_WAIT" {
				tcpTimewait++
			}
			if conn.Status == "LISTEN" {
				tcpListening++
			}
			if conn.Status == "SYN_SENT" {
				tcpSynsent++
			}
			if conn.Status == "SYN_RECV" {
				tcpSynrecv++
			}
			if conn.Status == "FIN_WAIT1" {
				tcpFinwait1++
			}
			if conn.Status == "FIN_WAIT2" {
				tcpFinwait2++
			}
			if conn.Status == "LAST_ACK" {
				tcpLastack++
			}
			if conn.Status == "CLOSING" {
				tcpClosing++
			}
		case syscall.SOCK_DGRAM:
			udpConns++
		}
	}

	return common.MapStr{
		"all": common.MapStr{
			"count":     allConns,
			"listening": allListening,
		},
		"tcp": common.MapStr{
			"all": common.MapStr{
				"count":       tcpConns,
				"listening":   tcpListening,
				"established": tcpEstablished,
				"close_wait":  tcpClosewait,
				"time_wait":   tcpTimewait,
				"syn_sent":    tcpSynsent,
				"syn_recv":    tcpSynrecv,
				"fin_wait1":   tcpFinwait1,
				"fin_wait2":   tcpFinwait2,
				"last_ack":    tcpLastack,
				"closing":     tcpClosing,
			},
		},
		"udp": common.MapStr{
			"all": common.MapStr{
				"count": udpConns,
			},
		},
	}
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// all network connections
	conns, err := connections("inet")

	if err != nil {
		return errors.Wrap(err, "error getting connections")
	}

	stats := calculateConnStats(conns)
	newStats, err := applyEnhancements(stats, m.mod)
	if err != nil {
		m.Logger().Debugf("error applying enhancements: %s", err)
		newStats = stats
	}

	report.Event(mb.Event{
		MetricSetFields: newStats,
	})

	return nil
}
