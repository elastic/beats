// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"net"
	"time"

	"github.com/joeshaw/multierror"
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

// Host represents information about a host.
type Host struct {
	info  types.HostInfo
	addrs []net.Addr
	macs  []net.HardwareAddr
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

	host, err := getHost()
	if err != nil {
		return err
	}

	report.Event(hostEvent(host, eventTypeState, eventActionHost))

	return nil
}

func getHost() (*Host, error) {
	sysinfoHost, err := sysinfo.Host()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load host information")
	}

	addrs, macs, err := getNetInfo()
	if err != nil {
		return nil, err
	}

	host := &Host{
		info:  sysinfoHost.Info(),
		addrs: addrs,
		macs:  macs,
	}

	return host, nil
}

func hostEvent(host *Host, eventType string, eventAction string) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventType,
				"action": eventAction,
			},
		},
		MetricSetFields: common.MapStr{
			// https://github.com/elastic/ecs#-host-fields
			"uptime":              host.info.Uptime(),
			"boottime":            host.info.BootTime,
			"containerized":       host.info.Containerized,
			"timezone.name":       host.info.Timezone,
			"timezone.offset.sec": host.info.TimezoneOffsetSec,
			"name":                host.info.Hostname,
			"id":                  host.info.UniqueID,
			// TODO "host.type": ?
			"architecture": host.info.Architecture,

			// https://github.com/elastic/ecs#-operating-system-fields
			"os": common.MapStr{
				"platform": host.info.OS.Platform,
				"name":     host.info.OS.Name,
				"family":   host.info.OS.Family,
				"version":  host.info.OS.Version,
				"kernel":   host.info.KernelVersion,
			},
		},
	}

	var ipStrings []string
	for _, addr := range host.addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			ipStrings = append(ipStrings, v.IP.String())
		case *net.IPAddr:
			ipStrings = append(ipStrings, v.IP.String())
		}
	}
	event.MetricSetFields.Put("ip", ipStrings)

	var macStrings []string
	for _, mac := range host.macs {
		macStr := mac.String()
		if macStr != "" {
			macStrings = append(macStrings, macStr)
		}
	}
	event.MetricSetFields.Put("mac", macStrings)

	return event
}

// getNetInfo is originally copied from libbeat/processors/add_host_metadata.go.
// TODO: Maybe these two can share an implementation?
func getNetInfo() ([]net.Addr, []net.HardwareAddr, error) {
	var addrList []net.Addr
	var hwList []net.HardwareAddr

	// Get all interfaces and loop through them
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	// Keep track of all errors
	var errs multierror.Errors

	for _, i := range ifaces {
		// Skip loopback interfaces
		if i.Flags&net.FlagLoopback == net.FlagLoopback {
			continue
		}

		hwList = append(hwList, i.HardwareAddr)

		addrs, err := i.Addrs()
		if err != nil {
			// If we get an error, keep track of it and continue with the next interface
			errs = append(errs, err)
			continue
		}

		addrList = append(addrList, addrs...)
	}

	return addrList, hwList, errs.Err()
}
