// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"net"
	"time"

	"github.com/OneOfOne/xxhash"
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
	eventTypeEvent = "event"

	eventActionHost        = "host"
	eventActionIdChanged   = "host_id_changed"
	eventActionReboot      = "reboot"
	eventActionHostChanged = "host_changed"
)

// Host represents information about a host.
type Host struct {
	info types.HostInfo
	// Uptime() in types.HostInfo recalculates the uptime every time it is called -
	// so storing it permanently here.
	uptime time.Duration
	addrs  []net.Addr
	macs   []net.HardwareAddr
}

// changeDetectionHash creates a hash of selected parts of the host information.
// This is used later to detect changes to a host over time.
func (host *Host) changeDetectionHash() uint64 {
	h := xxhash.New64()

	mapstr := host.toMapStr()

	// We detect changes to the ID separately, so ignore here.
	mapstr.Delete("id")

	// Changes in uptime and boottime indicate a system reboot,
	// which we report separately.
	mapstr.Delete("uptime")
	mapstr.Delete("boottime")

	h.WriteString(mapstr.String())
	return h.Sum64()
}

func (host *Host) toMapStr() common.MapStr {
	mapstr := common.MapStr{
		// https://github.com/elastic/ecs#-host-fields
		"uptime":              host.uptime,
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
	mapstr.Put("ip", ipStrings)

	var macStrings []string
	for _, mac := range host.macs {
		macStr := mac.String()
		if macStr != "" {
			macStrings = append(macStrings, macStr)
		}
	}
	mapstr.Put("mac", macStrings)

	return mapstr
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	config    config
	log       *logp.Logger
	lastState time.Time
	lastHost  *Host
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(moduleName),
	}, nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	needsStateUpdate := time.Since(ms.lastState) > ms.config.effectiveStatePeriod()
	if needsStateUpdate {
		ms.log.Debug("State update needed.")
		err := ms.reportState(report)
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
		}
		ms.log.Debugf("Next state update by %v", ms.lastState.Add(ms.config.effectiveStatePeriod()))
	}

	err := ms.reportChanges(report)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
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

// reportChanges detects and reports any changes to this host since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	currentHost, err := getHost()
	if err != nil {
		return err
	}

	defer func() {
		ms.lastHost = currentHost
	}()

	if ms.lastHost == nil {
		// First run - no changes possible
		return nil
	}

	// Report ID changes as a separate, special event.
	if ms.lastHost.info.UniqueID != currentHost.info.UniqueID {
		/*
		 Issue two events - one for the host with the old ID, one for the new
		 - to link them (since the unique ID is what identifies a host).
		*/
		eventOldHost := hostEvent(ms.lastHost, eventTypeEvent, eventActionIdChanged)
		eventOldHost.MetricSetFields.Put("new_id", currentHost.info.UniqueID)
		report.Event(eventOldHost)

		eventNewHost := hostEvent(currentHost, eventTypeEvent, eventActionIdChanged)
		eventNewHost.MetricSetFields.Put("old_id", ms.lastHost.info.UniqueID)
		report.Event(eventNewHost)
	}

	// Report reboots separately
	if !currentHost.info.BootTime.Equal(ms.lastHost.info.BootTime) {
		report.Event(hostEvent(currentHost, eventTypeEvent, eventActionReboot))
	}

	// Report any other changes.
	if ms.lastHost.changeDetectionHash() != currentHost.changeDetectionHash() {
		report.Event(hostEvent(currentHost, eventTypeEvent, eventActionHostChanged))
	}

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
		info:   sysinfoHost.Info(),
		uptime: sysinfoHost.Info().Uptime(),
		addrs:  addrs,
		macs:   macs,
	}

	return host, nil
}

func hostEvent(host *Host, eventType string, eventAction string) mb.Event {
	return mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventType,
				"action": eventAction,
			},
		},
		MetricSetFields: host.toMapStr(),
	}
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
