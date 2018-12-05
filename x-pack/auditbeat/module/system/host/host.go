// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	moduleName    = "system"
	metricsetName = "host"

	eventTypeState = "state"

	eventActionHost = "host"
)

type Host struct {
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	log       *logp.Logger
	lastState time.Time
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(moduleName),
	}, nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	err := ms.reportState(report)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}
}

// reportState reports the current state of the host.
func (ms *MetricSet) reportState(report mb.ReporterV2) error {
	ms.lastState = time.Now()

	host, err := sysinfo.Host()
	if err != nil {
		return errors.Wrap(err, "failed to load host information")
	}

	report.Event(hostEvent(host.Info(), eventTypeState, eventActionHost))

	return nil
}

func hostEvent(hostInfo types.HostInfo, eventType string, eventAction string) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventType,
				"action": eventAction,
			},
		},
		MetricSetFields: common.MapStr{
			// https://github.com/elastic/ecs#-host-fields
			"uptime":              hostInfo.Uptime(),
			"boottime":            hostInfo.BootTime,
			"containerized":       hostInfo.Containerized,
			"timezone.name":       hostInfo.Timezone,
			"timezone.offset.sec": hostInfo.TimezoneOffsetSec,
			"name":                hostInfo.Hostname,
			"id":                  hostInfo.UniqueID,
			// TODO "host.type": ?
			"architecture": hostInfo.Architecture,
			"ip":           hostInfo.IPs,
			"mac":          hostInfo.MACs,

			// https://github.com/elastic/ecs#-operating-system-fields
			"os": common.MapStr{
				"platform": hostInfo.OS.Platform,
				"name":     hostInfo.OS.Name,
				"family":   hostInfo.OS.Family,
				"version":  hostInfo.OS.Version,
				"kernel":   hostInfo.KernelVersion,
			},
		},
	}

	return event
}

// NetworkInterface represent information on a network interface.
type NetworkInterface struct {
	net.Interface

	ips []net.IP
}

func (ifc NetworkInterface) toMapStr() common.MapStr {
	return common.MapStr{
		"index": ifc.Index,
		"mtu":   ifc.MTU,
		"name":  ifc.Name,
		"mac":   ifc.HardwareAddr.String(),
		"flags": ifc.Flags.String(),
		"ip":    ifc.ips,
	}
}

// getInterfaces fetches information about the system's network interfaces.
// TODO: Move to go-sysinfo?
func getNetworkInterfaces() ([]NetworkInterface, error) {
	ifcs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var networkInterfaces []NetworkInterface

	for _, ifc := range ifcs {
		addrs, err := ifc.Addrs()
		if err != nil {
			return nil, err
		}

		var ips []net.IP
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				return nil, err
			}

			ips = append(ips, ip)
		}

		isLoopback := ifc.Flags&net.FlagLoopback != 0
		if !isLoopback {
			networkInterfaces = append(networkInterfaces, NetworkInterface{
				ifc,
				ips,
			})
		}
	}

	return networkInterfaces, nil
}
