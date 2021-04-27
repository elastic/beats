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

package network

import (
	"context"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/module/docker"

	sysinfo "github.com/elastic/go-sysinfo"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// NetService represents maps out the interface-level stats per-container
type NetService struct {
	NetworkStatPerContainer map[string]map[string]NetRaw
}

// NetworkCalculator is the interface that reports per-second stats
type NetworkCalculator interface {
	getRxBytesPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getRxDroppedPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getRxErrorsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getRxPacketsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getTxBytesPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getTxDroppedPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getTxErrorsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
	getTxPacketsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64
}

// NetRaw represents the raw network stats from docker
type NetRaw struct {
	Time      time.Time
	RxBytes   uint64
	RxDropped uint64
	RxErrors  uint64
	RxPackets uint64
	TxBytes   uint64
	TxDropped uint64
	TxErrors  uint64
	TxPackets uint64
}

// NetStats represents the network counters for a given network interface
type NetStats struct {
	Time          time.Time
	Container     *docker.Container
	NameInterface string
	RxBytes       float64
	RxDropped     float64
	RxErrors      float64
	RxPackets     float64
	TxBytes       float64
	TxDropped     float64
	TxErrors      float64
	TxPackets     float64
	Netstat       *sysinfotypes.NetworkCountersInfo
	Total         *types.NetworkStats
}

// NoSumStats lists "stats", often config/state values, that can't be safely summed across PIDs
var NoSumStats = []string{
	"RtoAlgorithm",
	"RtoMin",
	"RtoMax",
	"MaxConn",
	"Forwarding",
	"DefaultTTL",
}

func (n *NetService) getNetworkStatsPerContainer(client *client.Client, timeout time.Duration, rawStats []docker.Stat, cfg Config) ([]NetStats, error) {
	formattedStats := []NetStats{}
	for _, myStats := range rawStats {

		var stats *sysinfotypes.NetworkCountersInfo
		var err error
		if cfg.NetworkSummary {
			stats, err = fetchContainerNetStats(client, timeout, myStats.Container.ID)
			if err != nil {
				return nil, errors.Wrap(err, "error fetching per-PID stats")
			}
		}

		for nameInterface, rawnNetStats := range myStats.Stats.Networks {
			singleStat := n.getNetworkStats(nameInterface, rawnNetStats, myStats, cfg.DeDot)
			if cfg.NetworkSummary {
				singleStat.Netstat = stats
			}
			formattedStats = append(formattedStats, singleStat)
		}
	}

	return formattedStats, nil
}

// fetchContainerNetStats gathers the PIDs associated with a container, and then uses go-sysinfo to grab the /proc/[pid]/net counters and sum them across PIDs.
func fetchContainerNetStats(client *client.Client, timeout time.Duration, container string) (*sysinfotypes.NetworkCountersInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	body, err := client.ContainerTop(ctx, container, []string{})
	if err != nil {
		return &sysinfotypes.NetworkCountersInfo{}, errors.Wrap(err, "error fetching container pids")
	}

	var pidPos int
	for pos := range body.Titles {
		if body.Titles[pos] == "PID" {
			pidPos = pos
			break
		}
	}

	summedMetrics := &sysinfotypes.NetworkCountersInfo{}
	for _, pid := range body.Processes {
		strPID := pid[pidPos]
		intPID, err := strconv.Atoi(strPID)
		if err != nil {
			return &sysinfotypes.NetworkCountersInfo{}, errors.Wrap(err, "error converting PID to int")
		}

		proc, err := sysinfo.Process(intPID)
		procNet, ok := proc.(sysinfotypes.NetworkCounters)
		if !ok {
			break
		}

		counters, err := procNet.NetworkCounters()
		if err != nil {
			return &sysinfotypes.NetworkCountersInfo{}, errors.Wrapf(err, "error fetching network counters for PID %d", intPID)
		}

		summedMetrics = sumCounter(summedMetrics, counters)

	}

	return summedMetrics, nil

}

func sumCounter(totals, new *sysinfotypes.NetworkCountersInfo) *sysinfotypes.NetworkCountersInfo {

	newTotal := &sysinfotypes.NetworkCountersInfo{}

	newTotal.Netstat.IPExt = sumMapStr(new.Netstat.IPExt, totals.Netstat.IPExt)
	newTotal.Netstat.TCPExt = sumMapStr(new.Netstat.TCPExt, totals.Netstat.TCPExt)

	newTotal.SNMP.IP = sumMapStr(new.SNMP.IP, totals.SNMP.IP)
	newTotal.SNMP.ICMP = sumMapStr(new.SNMP.ICMP, totals.SNMP.ICMP)
	newTotal.SNMP.ICMPMsg = sumMapStr(new.SNMP.ICMPMsg, totals.SNMP.ICMPMsg)
	newTotal.SNMP.TCP = sumMapStr(new.SNMP.TCP, totals.SNMP.TCP)
	newTotal.SNMP.UDP = sumMapStr(new.SNMP.UDP, totals.SNMP.UDP)
	newTotal.SNMP.UDPLite = sumMapStr(new.SNMP.UDPLite, totals.SNMP.UDPLite)
	return newTotal

}

func sumMapStr(m1, m2 map[string]uint64) map[string]uint64 {

	final := make(map[string]uint64)
	for key, val := range m1 {
		// skip over values that aren't counters and can't be summed across interfaces
		// Most of these values are config settings, and it doesn't make sense to report them along with counters that aren't per-pid/per-interface
		var skip = false
		for _, name := range NoSumStats {
			if key == name {
				skip = true
			}
		}
		if skip {
			continue
		}
		// safety, make sure the field exists in m1
		if _, ok := m2[key]; !ok {
			final[key] = val
		}
		final[key] = m2[key] + val

	}
	return final
}

func (n *NetService) getNetworkStats(nameInterface string, rawNetStats types.NetworkStats, myRawstats docker.Stat, dedot bool) NetStats {
	newNetworkStats := createNetRaw(myRawstats.Stats.Read, &rawNetStats)
	oldNetworkStat, exist := n.NetworkStatPerContainer[myRawstats.Container.ID][nameInterface]

	netStats := NetStats{
		Container:     docker.NewContainer(myRawstats.Container, dedot),
		Time:          myRawstats.Stats.Read,
		NameInterface: nameInterface,
		Total:         &rawNetStats,
	}

	if exist {
		netStats.RxBytes = n.getRxBytesPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.RxDropped = n.getRxDroppedPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.RxErrors = n.getRxErrorsPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.RxPackets = n.getRxPacketsPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxBytes = n.getTxBytesPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxDropped = n.getTxDroppedPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxErrors = n.getTxErrorsPerSecond(&newNetworkStats, &oldNetworkStat)
		netStats.TxPackets = n.getTxPacketsPerSecond(&newNetworkStats, &oldNetworkStat)
	} else {
		n.NetworkStatPerContainer[myRawstats.Container.ID] = make(map[string]NetRaw)
	}

	n.NetworkStatPerContainer[myRawstats.Container.ID][nameInterface] = newNetworkStats

	return netStats
}

func createNetRaw(time time.Time, stats *types.NetworkStats) NetRaw {
	return NetRaw{
		Time:      time,
		RxBytes:   stats.RxBytes,
		RxDropped: stats.RxDropped,
		RxErrors:  stats.RxErrors,
		RxPackets: stats.RxPackets,
		TxBytes:   stats.TxBytes,
		TxDropped: stats.TxDropped,
		TxErrors:  stats.TxErrors,
		TxPackets: stats.TxPackets,
	}
}

func (n *NetService) checkStats(containerID string, nameInterface string) bool {
	if _, exist := n.NetworkStatPerContainer[containerID][nameInterface]; exist {
		return true
	}
	return false
}

func (n *NetService) getRxBytesPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.RxBytes, newStats.RxBytes)
}

func (n *NetService) getRxDroppedPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.RxDropped, newStats.RxDropped)
}

func (n *NetService) getRxErrorsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.RxErrors, newStats.RxErrors)
}

func (n *NetService) getRxPacketsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.RxPackets, newStats.RxPackets)
}

func (n *NetService) getTxBytesPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.TxBytes, newStats.TxBytes)
}

func (n *NetService) getTxDroppedPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.TxDropped, newStats.TxDropped)
}

func (n *NetService) getTxErrorsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.TxErrors, newStats.TxErrors)
}

func (n *NetService) getTxPacketsPerSecond(newStats *NetRaw, oldStats *NetRaw) float64 {
	duration := newStats.Time.Sub(oldStats.Time)
	return n.calculatePerSecond(duration, oldStats.TxPackets, newStats.TxPackets)
}

func (n *NetService) calculatePerSecond(duration time.Duration, oldValue uint64, newValue uint64) float64 {
	value := float64(newValue) - float64(oldValue)
	if value < 0 {
		value = 0
	}
	return value / duration.Seconds()
}
