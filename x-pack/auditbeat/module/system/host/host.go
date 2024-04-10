// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	metricsetName = "host"
	namespace     = "system.audit.host"

	bucketName        = "host.v1"
	bucketKeyLastHost = "lastHost"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

type eventAction uint8

const (
	eventActionHost eventAction = iota
	eventActionIDChanged
	eventActionReboot
	eventActionHostnameChanged
	eventActionHostChanged
)

func (action eventAction) String() string {
	switch action {
	case eventActionHost:
		return "host"
	case eventActionIDChanged:
		return "host_id_changed"
	case eventActionReboot:
		return "reboot"
	case eventActionHostnameChanged:
		return "hostname_changed"
	case eventActionHostChanged:
		return "host_changed"
	default:
		return ""
	}
}

func (action eventAction) Type() string {
	switch action {
	case eventActionHost:
		return "info"
	case eventActionIDChanged:
		return "change"
	case eventActionReboot:
		return "start"
	case eventActionHostnameChanged:
		return "change"
	case eventActionHostChanged:
		return "change"
	default:
		return "info"
	}
}

// Host represents information about a host.
type Host struct {
	Info types.HostInfo
	// Uptime() in types.HostInfo recalculates the uptime every time it is called -
	// so storing it permanently here.
	Uptime time.Duration
	Ips    []net.IP
	Macs   []net.HardwareAddr
}

// changeDetectionHash creates a hash of selected parts of the host information.
// This is used later to detect changes to a host over time.
//
//nolint:errcheck // All checks are for writes to a hasher.
func (host *Host) changeDetectionHash() uint64 {
	h := xxhash.New()

	if host.Info.Containerized != nil {
		h.WriteString(strconv.FormatBool(*host.Info.Containerized))
	}

	h.WriteString(host.Info.Timezone)
	binary.Write(h, binary.BigEndian, int32(host.Info.TimezoneOffsetSec))
	h.WriteString(host.Info.Architecture)
	h.WriteString(host.Info.OS.Platform)
	h.WriteString(host.Info.OS.Name)
	h.WriteString(host.Info.OS.Family)
	h.WriteString(host.Info.OS.Version)
	h.WriteString(host.Info.KernelVersion)

	return h.Sum64()
}

//nolint:errcheck // All checks are for mapstr.Put.
func (host *Host) toMapStr() mapstr.M {
	mapstr := mapstr.M{
		// https://github.com/elastic/ecs#-host-fields
		"uptime":              host.Uptime,
		"boottime":            host.Info.BootTime,
		"timezone.name":       host.Info.Timezone,
		"timezone.offset.sec": host.Info.TimezoneOffsetSec,
		"hostname":            host.Info.Hostname,
		"id":                  host.Info.UniqueID,
		"architecture":        host.Info.Architecture,

		// https://github.com/elastic/ecs#-operating-system-fields
		"os": mapstr.M{
			"platform": host.Info.OS.Platform,
			"name":     host.Info.OS.Name,
			"family":   host.Info.OS.Family,
			"version":  host.Info.OS.Version,
			"kernel":   host.Info.KernelVersion,
		},
	}

	if host.Info.Containerized != nil {
		mapstr.Put("containerized", host.Info.Containerized)
	}

	if host.Info.OS.Codename != "" {
		mapstr.Put("os.codename", host.Info.OS.Codename)
	}

	if host.Info.OS.Type != "" {
		mapstr.Put("os.type", host.Info.OS.Type)
	}

	var ipStrings []string
	for _, ip := range host.Ips {
		ipStrings = append(ipStrings, ip.String())
	}
	mapstr.Put("ip", ipStrings)

	var macStrings []string
	for _, mac := range host.Macs {
		if len(mac) != 0 {
			macStrings = append(macStrings, formatHardwareAddr(mac))
		}
	}
	mapstr.Put("mac", macStrings)

	return mapstr
}

// formatHardwareAddr formats hardware addresses according to the ECS spec.
func formatHardwareAddr(addr net.HardwareAddr) string {
	buf := make([]byte, 0, len(addr)*3-1)
	for _, b := range addr {
		if len(buf) != 0 {
			buf = append(buf, '-')
		}
		const hexDigit = "0123456789ABCDEF"
		buf = append(buf, hexDigit[b>>4], hexDigit[b&0xf])
	}
	return string(buf)
}

// InitializeModule initializes this module.
func InitializeModule() {
	mb.Registry.MustAddMetricSet(system.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	config    config
	log       *logp.Logger
	bucket    datastore.Bucket
	lastState time.Time
	lastHost  *Host
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v/%v dataset is beta", system.ModuleName, metricsetName)

	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %v/%v config: %w", system.ModuleName, metricsetName, err)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to open persistent datastore: %w", err)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(system.ModuleName),
		bucket:        bucket,
	}

	// Load state (lastHost) from disk
	err = ms.restoreStateFromDisk()
	if err != nil {
		return nil, fmt.Errorf("failed to restore state from disk: %w", err)
	}

	return ms, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
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

	return ms.saveStateToDisk()
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

	var events []mb.Event

	// Report ID changes as a separate, special event.
	//nolint:errcheck // All checks are for mapstr.Put.
	if ms.lastHost.Info.UniqueID != currentHost.Info.UniqueID {
		/*
		 Issue two events - one for the host with the old ID, one for the new
		 - to link them (since the unique ID is what identifies a host).
		*/
		eventOldHost := hostEvent(ms.lastHost, eventTypeEvent, eventActionIDChanged)
		eventOldHost.MetricSetFields.Put("new_id", currentHost.Info.UniqueID)
		events = append(events, eventOldHost)

		eventNewHost := hostEvent(currentHost, eventTypeEvent, eventActionIDChanged)
		eventNewHost.MetricSetFields.Put("old_id", ms.lastHost.Info.UniqueID)
		events = append(events, eventNewHost)
	}

	// Report reboots separately
	// On Windows, BootTime is not fully accurate and can vary by a few milliseconds.
	// So we only report a reboot if the new BootTime is at least 1 second after the old.
	if currentHost.Info.BootTime.After(ms.lastHost.Info.BootTime.Add(1 * time.Second)) {
		events = append(events, hostEvent(currentHost, eventTypeEvent, eventActionReboot))
	}

	// Report hostname changes separately
	if currentHost.Info.Hostname != ms.lastHost.Info.Hostname {
		events = append(events, hostEvent(currentHost, eventTypeEvent, eventActionHostnameChanged))
	}

	// Report any other changes.
	if ms.lastHost.changeDetectionHash() != currentHost.changeDetectionHash() {
		events = append(events, hostEvent(currentHost, eventTypeEvent, eventActionHostChanged))
	}

	for _, event := range events {
		report.Event(event)
	}

	if len(events) > 0 {
		return ms.saveStateToDisk()
	}

	return nil
}

func getHost() (*Host, error) {
	sysinfoHost, err := sysinfo.Host()
	if err != nil {
		return nil, fmt.Errorf("failed to load host information: %w", err)
	}

	ips, macs, err := getNetInfo()
	if err != nil {
		return nil, err
	}

	host := &Host{
		Info:   sysinfoHost.Info(),
		Uptime: sysinfoHost.Info().Uptime(),
		Ips:    ips,
		Macs:   macs,
	}

	return host, nil
}

//nolint:errcheck // All checks are for mapstr.CopyFieldsTo.
func hostEvent(host *Host, eventType string, action eventAction) mb.Event {
	hostFields := host.toMapStr()

	event := mb.Event{
		RootFields: mapstr.M{
			"event": mapstr.M{
				"kind":     eventType,
				"category": []string{"host"},
				"type":     []string{action.Type()},
				"action":   action.String(),
			},
			"message": hostMessage(host, action),
		},
		MetricSetFields: hostFields,
	}

	// Copy select host.* fields in case add_host_metadata is not configured.
	hostTopLevel := mapstr.M{}
	hostFields.CopyFieldsTo(hostTopLevel, "architecture")
	hostFields.CopyFieldsTo(hostTopLevel, "containerized")
	hostFields.CopyFieldsTo(hostTopLevel, "hostname")
	hostFields.CopyFieldsTo(hostTopLevel, "id")
	hostFields.CopyFieldsTo(hostTopLevel, "ip")
	hostFields.CopyFieldsTo(hostTopLevel, "mac")
	hostFields.CopyFieldsTo(hostTopLevel, "os.codename")
	hostFields.CopyFieldsTo(hostTopLevel, "os.family")
	hostFields.CopyFieldsTo(hostTopLevel, "os.kernel")
	hostFields.CopyFieldsTo(hostTopLevel, "os.name")
	hostFields.CopyFieldsTo(hostTopLevel, "os.platform")
	hostFields.CopyFieldsTo(hostTopLevel, "os.type")
	hostFields.CopyFieldsTo(hostTopLevel, "os.version")

	event.RootFields.Put("host", hostTopLevel)

	return event
}

func hostMessage(host *Host, action eventAction) string {
	var firstIP string
	if len(host.Ips) > 0 {
		firstIP = host.Ips[0].String()
	}

	// Hostname + IP of the first non-loopback interface.
	hostString := fmt.Sprintf("%v (IP: %v)", host.Info.Hostname, firstIP)

	var message string
	switch action {
	case eventActionHost:
		message = fmt.Sprintf("%v host %v is up for %v",
			host.Info.OS.Name, hostString, fmtDuration(host.Uptime))
	case eventActionIDChanged:
		message = fmt.Sprintf("ID of host %v has changed", hostString)
	case eventActionReboot:
		message = fmt.Sprintf("Host %v restarted", hostString)
	case eventActionHostnameChanged:
		message = fmt.Sprintf("Hostname changed to %v", hostString)
	case eventActionHostChanged:
		message = fmt.Sprintf("Host %v changed", hostString)
	}

	return message
}

func fmtDuration(d time.Duration) string {
	const dayMinutes = 60 * 24

	remainingMinutes := math.Floor(d.Minutes())
	days := math.Floor(remainingMinutes / dayMinutes)

	remainingMinutes -= days * dayMinutes
	hours := math.Floor(remainingMinutes / 60)

	remainingMinutes -= hours * 60
	minutes := math.Floor(remainingMinutes)

	return fmt.Sprintf("%.f %v, %.f %v, %.f %v",
		days, inflect("day", int(days)),
		hours, inflect("hour", int(hours)),
		minutes, inflect("minute", int(minutes)))
}

func inflect(noun string, count int) string {
	if count == 1 {
		return noun
	}
	return noun + "s"
}

func (ms *MetricSet) saveStateToDisk() error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if ms.lastHost != nil {
		err := encoder.Encode(*ms.lastHost)
		if err != nil {
			return fmt.Errorf("error encoding host information: %w", err)
		}

		err = ms.bucket.Store(bucketKeyLastHost, buf.Bytes())
		if err != nil {
			return fmt.Errorf("error writing host information to disk: %w", err)
		}

		ms.log.Debug("Wrote host information to disk.")
	}
	return nil
}

func (ms *MetricSet) restoreStateFromDisk() error {
	var decoder *gob.Decoder
	err := ms.bucket.Load(bucketKeyLastHost, func(blob []byte) error {
		if len(blob) > 0 {
			buf := bytes.NewBuffer(blob)
			decoder = gob.NewDecoder(buf)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if decoder != nil {
		var lastHost Host
		err = decoder.Decode(&lastHost)
		switch err { //nolint:errorlint // Bad linter! io.EOF is never wrapped.
		case nil:
			ms.lastHost = &lastHost
		case io.EOF:
		default:
			return fmt.Errorf("error decoding host information: %w", err)
		}
	}

	if ms.lastHost != nil {
		ms.log.Debug("Restored last host information from disk.")
	} else {
		ms.log.Debug("No last host information found on disk.")
	}

	return nil
}

// getNetInfo is originally copied from libbeat/processors/add_host_metadata.go.
// TODO: Maybe these two can share an implementation?
func getNetInfo() ([]net.IP, []net.HardwareAddr, error) {
	var ipv4List []net.IP
	var ipv6List []net.IP
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

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			if ip.To4() != nil {
				ipv4List = append(ipv4List, ip)
			} else {
				ipv6List = append(ipv6List, ip)
			}
		}
	}

	return append(ipv4List, ipv6List...), hwList, errs.Err()
}
