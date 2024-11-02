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

//go:build darwin || freebsd || linux || windows || aix

package network

import (
	"fmt"
	"math"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/shirou/gopsutil/v3/net"
)

var debugf = logp.MakeDebug("system-network")

func init() {
	mb.Registry.MustAddMetricSet("system", "network", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system network IO metrics.
type MetricSet struct {
	mb.BaseMetricSet
	interfaces           map[string]struct{}
	prevInterfaceCounter map[string]networkCounter
	currentGaugeCounter  map[string]networkCounter
}

// networkCounter stores previous network counter values for calculating gauges in next collection
type networkCounter struct {
	NetworkInBytes    uint64
	NetworkInPackets  uint64
	NetworkOutBytes   uint64
	NetworkOutPackets uint64
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Interfaces []string `config:"interfaces"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	var interfaceSet map[string]struct{}
	if len(config.Interfaces) > 0 {
		interfaceSet = make(map[string]struct{}, len(config.Interfaces))
		for _, ifc := range config.Interfaces {
			interfaceSet[strings.ToLower(ifc)] = struct{}{}
		}
		debugf("network io stats will be included for %v", interfaceSet)
	}

	return &MetricSet{
		BaseMetricSet:        base,
		interfaces:           interfaceSet,
		prevInterfaceCounter: map[string]networkCounter{},
		currentGaugeCounter:  map[string]networkCounter{},
	}, nil
}

// Fetch fetches network IO metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := net.IOCounters(true)
	if err != nil {
		return fmt.Errorf("network io counters: %w", err)
	}

	for _, counters := range stats {
		if m.interfaces != nil {
			// Select stats by interface name.
			name := strings.ToLower(counters.Name)
			if _, include := m.interfaces[name]; !include {
				continue
			}
		}

		isOpen := r.Event(mb.Event{
			MetricSetFields: ioCountersToMapStr(counters),
		})

		// sum the values at a per-interface level
		// Makes us less likely to overload a value somewhere.
		prevCounters, ok := m.prevInterfaceCounter[counters.Name]
		if !ok {
			m.prevInterfaceCounter[counters.Name] = networkCounter{
				NetworkInBytes:    counters.BytesRecv,
				NetworkInPackets:  counters.PacketsRecv,
				NetworkOutBytes:   counters.BytesSent,
				NetworkOutPackets: counters.PacketsSent,
			}
			continue
		}
		// create current set of gauges
		currentDiff := networkCounter{
			NetworkInBytes:    createGaugeWithRollover(counters.BytesRecv, prevCounters.NetworkInBytes),
			NetworkInPackets:  createGaugeWithRollover(counters.PacketsRecv, prevCounters.NetworkInPackets),
			NetworkOutBytes:   createGaugeWithRollover(counters.BytesSent, prevCounters.NetworkOutBytes),
			NetworkOutPackets: createGaugeWithRollover(counters.PacketsSent, prevCounters.NetworkOutPackets),
		}

		m.currentGaugeCounter[counters.Name] = currentDiff

		m.prevInterfaceCounter[counters.Name] = networkCounter{
			NetworkInBytes:    counters.BytesRecv,
			NetworkInPackets:  counters.PacketsRecv,
			NetworkOutBytes:   counters.BytesSent,
			NetworkOutPackets: counters.PacketsSent,
		}

		if !isOpen {
			return nil
		}
	}

	if len(m.currentGaugeCounter) != 0 {

		var totalNetworkInBytes, totalNetworkInPackets, totalNetworkOutBytes, totalNetworkOutPackets uint64
		for _, iface := range m.currentGaugeCounter {
			totalNetworkInBytes += iface.NetworkInBytes
			totalNetworkInPackets += iface.NetworkInPackets
			totalNetworkOutBytes += iface.NetworkOutBytes
			totalNetworkOutPackets += iface.NetworkOutPackets
		}
		r.Event(mb.Event{
			RootFields: mapstr.M{
				"host": mapstr.M{
					"network": mapstr.M{
						"ingress": mapstr.M{
							"bytes":   totalNetworkInBytes,
							"packets": totalNetworkInPackets,
						},
						"egress": mapstr.M{
							"bytes":   totalNetworkOutBytes,
							"packets": totalNetworkOutPackets,
						},
					},
				},
			},
		})
	}

	return nil
}

// Create a gauged difference between two numbers, taking into account rollover that might happen, and the current number might be lower.
// The /proc/net/dev interface is defined in net/core/net-procfs.c,
// where it prints the data from rtnl_link_stats64 defined in uapi/linux/if_link.h.
// There's an extra bit of logic here: the underlying network device object in the kernel, net_device,
// can define either ndo_get_stats64() or ndo_get_stats() as a metrics callback, with the latter returning an unsigned long (32 bit) set of metrics.
// See dev_get_stats() in net/core/dev.c for context. The exact implementation depends upon the network driver.
// For example, the tg3 network driver used by the broadcom network controller on my dev machine
// uses 64 bit metrics, defined in the drivers/net/ethernet/broadcom/tg3.h,
// with the ndo_get_stats64() callback defined in  net/ethernet/broadcom/tg3.c.
// Long story short, we can't be completely sure if we're rolling over at max_u32 or max_u64.
// if the previous value was > max_u32, do math assuming we've rolled over at max_u64.
// On windows: This uses GetIfEntry: https://learn.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_if_row2 which uses ulong64.
// On Darwin we just call netstat.
// I'm assuming rollover behavior is similar.
func createGaugeWithRollover(current uint64, prev uint64) uint64 {
	// base case: no rollover
	if current >= prev {
		return current - prev
	}

	// case: rollover
	// case: we rolled over at 64 bits
	if prev > math.MaxUint32 {
		debugf("Warning: Rollover 64 bit gauge detected. Current value: %d, previous: %d", current, prev)
		remaining := math.MaxUint64 - prev
		return current + remaining + 1 // the +1 counts the actual "rollover" increment.
	}
	// case: we rolled over at 32 bits
	debugf("Warning: Rollover 32 bit gauge detected. Current value: %d, previous: %d", current, prev)
	remaining := math.MaxUint32 - prev
	return current + remaining + 1

}

func ioCountersToMapStr(counters net.IOCountersStat) mapstr.M {
	return mapstr.M{
		"name": counters.Name,
		"in": mapstr.M{
			"errors":  counters.Errin,
			"dropped": counters.Dropin,
			"bytes":   counters.BytesRecv,
			"packets": counters.PacketsRecv,
		},
		"out": mapstr.M{
			"errors":  counters.Errout,
			"dropped": counters.Dropout,
			"packets": counters.PacketsSent,
			"bytes":   counters.BytesSent,
		},
	}
}
