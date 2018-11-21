// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package socket

import (
	"fmt"
	"net"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	sock "github.com/elastic/beats/metricbeat/helper/socket"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/gosigar/sys/linux"
)

const (
	moduleName    = "system"
	metricsetName = "socket"

	bucketName              = "auditbeat.socket.v1"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"

	eventActionExistingSocket = "existing_socket"
	eventActionSocketOpened   = "socket_opened"
	eventActionSocketClosed   = "socket_closed"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about sockets.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	cache  *cache.Cache
	log    *logp.Logger

	netlink *sock.NetlinkSession
	// TODO: Replace with process data collected in processes metricset
	ptable    *sock.ProcTable
	listeners *sock.ListenerTable

	bucket    datastore.Bucket
	lastState time.Time
}

// Socket represents information about a socket.
type Socket struct {
	Family       linux.AddressFamily
	State        linux.TCPState
	LocalIP      net.IP
	LocalPort    int
	RemoteIP     net.IP
	RemotePort   int
	Inode        uint32
	Direction    sock.Direction
	UID          uint32
	Username     string
	ProcessPID   int
	ProcessName  string
	ProcessError error
}

// newSocket creates a new socket out of a netlink diag message.
func newSocket(diag *linux.InetDiagMsg) *Socket {
	return &Socket{
		Family:     linux.AddressFamily(diag.Family),
		State:      linux.TCPState(diag.State),
		LocalIP:    diag.SrcIP(),
		LocalPort:  diag.SrcPort(),
		RemoteIP:   diag.DstIP(),
		RemotePort: diag.DstPort(),
		Inode:      diag.Inode,
		UID:        diag.UID,
		ProcessPID: -1,
	}
}

// Hash creates a hash for Socket.
func (s Socket) Hash() uint64 {
	h := xxhash.New64()
	h.WriteString(s.LocalIP.String())
	h.WriteString(s.RemoteIP.String())
	h.WriteString(strconv.Itoa(s.LocalPort))
	h.WriteString(strconv.Itoa(s.RemotePort))
	h.WriteString(strconv.FormatUint(uint64(s.Inode), 10))
	return h.Sum64()
}

func (s Socket) toMapStr() common.MapStr {
	evt := common.MapStr{
		"family":    s.Family.String(),
		"state":     s.State.String(),
		"direction": s.Direction.String(),
		"local": common.MapStr{
			"ip":   s.LocalIP,
			"port": s.LocalPort,
		},
		"remote": common.MapStr{
			"ip":   s.RemoteIP,
			"port": s.RemotePort,
		},
		"user": common.MapStr{
			"uid": s.UID,
		},
	}

	if s.Username != "" {
		evt.Put("user.name", s.Username)
	}

	if s.ProcessName != "" {
		evt["process"] = common.MapStr{
			"pid":  s.ProcessPID,
			"name": s.ProcessName,
		}
	}

	if s.ProcessError != nil {
		evt.Put("process.error", s.ProcessError.Error())
	}

	return evt
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ptable, err := sock.NewProcTable("")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create process table")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(metricsetName),
		cache:         cache.New(),
		netlink:       sock.NewNetlinkSession(),
		ptable:        ptable,
		listeners:     sock.NewListenerTable(),
		bucket:        bucket,
	}

	// Load from disk: Time when state was last sent
	err = bucket.Load(bucketKeyStateTimestamp, func(blob []byte) error {
		if len(blob) > 0 {
			return ms.lastState.UnmarshalBinary(blob)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !ms.lastState.IsZero() {
		ms.log.Debugf("Last state was sent at %v. Next state update by %v.", ms.lastState, ms.lastState.Add(ms.config.effectiveStatePeriod()))
	} else {
		ms.log.Debug("No state timestamp found")
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

// Fetch collects the user information. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	needsStateUpdate := time.Since(ms.lastState) > ms.config.effectiveStatePeriod()
	if needsStateUpdate || ms.cache.IsEmpty() {
		ms.log.Debugf("State update needed (needsStateUpdate=%v, cache.IsEmpty()=%v)", needsStateUpdate, ms.cache.IsEmpty())
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

// reportState reports all existing sockets on the system.
func (ms *MetricSet) reportState(report mb.ReporterV2) error {
	// Only update lastState if this state update was regularly scheduled,
	// i.e. not caused by an Auditbeat restart (when the cache would be empty).
	if !ms.cache.IsEmpty() {
		ms.lastState = time.Now()
	}

	sockets, err := ms.getSockets()
	if err != nil {
		return errors.Wrap(err, "failed to get sockets")
	}
	ms.log.Debugf("Found %d sockets", len(sockets))

	stateID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "error generating state ID")
	}

	// Refresh data for direction and process enrichment
	err = ms.refreshEnrichments(sockets)
	if err != nil {
		return err
	}

	for _, socket := range sockets {
		err = ms.enrichSocket(socket)
		if err != nil {
			return err
		}

		event := socketEvent(socket, eventTypeState, eventActionExistingSocket)
		event.RootFields.Put("event.id", stateID.String())
		report.Event(event)
	}

	// This will initialize the cache with the current sockets
	ms.cache.DiffAndUpdateCache(convertToCacheable(sockets))

	// Save time so we know when to send the state again (config.StatePeriod)
	timeBytes, err := ms.lastState.MarshalBinary()
	if err != nil {
		return err
	}
	err = ms.bucket.Store(bucketKeyStateTimestamp, timeBytes)
	if err != nil {
		return errors.Wrap(err, "error writing state timestamp to disk")
	}

	return nil
}

// reportChanges detects and reports any changes to sockets on this system since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	sockets, err := ms.getSockets()
	if err != nil {
		return errors.Wrap(err, "failed to get sockets")
	}

	opened, closed := ms.cache.DiffAndUpdateCache(convertToCacheable(sockets))
	ms.log.Debugf("Found %d sockets (%d opened, %d closed)", len(sockets), len(opened), len(closed))

	if len(opened) > 0 {
		// Refresh data for direction and process enrichment - only new sockets
		// need enrichment
		err = ms.refreshEnrichments(sockets)
		if err != nil {
			return err
		}

		for _, s := range opened {
			err = ms.enrichSocket(s.(*Socket))
			if err != nil {
				return err
			}

			report.Event(socketEvent(s.(*Socket), eventTypeEvent, eventActionSocketOpened))
		}
	}

	for _, s := range closed {
		report.Event(socketEvent(s.(*Socket), eventTypeEvent, eventActionSocketClosed))
	}

	return nil
}

func socketEvent(socket *Socket, eventType string, eventAction string) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventType,
				"action": eventAction,
			},
			"user": common.MapStr{
				"id": socket.UID,
			},
		},
		MetricSetFields: socket.toMapStr(),
	}

	if socket.Username != "" {
		event.RootFields.Put("user.name", socket.Username)
	}

	if socket.ProcessName != "" {
		event.RootFields.Put("process", common.MapStr{
			"pid":  socket.ProcessPID,
			"name": socket.ProcessName,
		})
	}

	return event
}

func convertToCacheable(sockets []*Socket) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(sockets))

	for _, s := range sockets {
		c = append(c, s)
	}

	return c
}

func (ms *MetricSet) enrichSocket(socket *Socket) error {
	userAccount, err := user.LookupId(strconv.FormatUint(uint64(socket.UID), 10))
	if err != nil {
		return errors.Wrapf(err, "error looking up socket UID")
	}

	socket.Username = userAccount.Username

	socket.Direction = ms.listeners.Direction(uint8(syscall.IPPROTO_TCP),
		socket.LocalIP, socket.LocalPort, socket.RemoteIP, socket.RemotePort)

	if ms.ptable != nil {
		proc := ms.ptable.ProcessBySocketInode(socket.Inode)
		if proc != nil {
			// Add process info by finding the process that holds the socket's inode.
			socket.ProcessPID = proc.PID
			socket.ProcessName = proc.Command
		} else if socket.Inode == 0 {
			socket.ProcessError = fmt.Errorf("process has exited (inode=%v)", socket.Inode)
		} else {
			socket.ProcessError = fmt.Errorf("process not found (inode=%v)", socket.Inode)
		}
	}

	return nil
}

func (ms *MetricSet) getSockets() ([]*Socket, error) {
	diags, err := ms.netlink.GetSocketList()
	if err != nil {
		return nil, errors.Wrap(err, "error getting sockets")
	}

	sockets := make([]*Socket, 0, len(diags))
	for _, diag := range diags {
		sockets = append(sockets, newSocket(diag))
	}

	return sockets, nil
}

func (ms *MetricSet) refreshEnrichments(sockets []*Socket) error {
	// Refresh inode to process mapping for process enrichment
	err := ms.ptable.Refresh()
	if err != nil {
		return errors.Wrap(err, "error refreshing process data")
	}

	// Register all listening sockets
	ms.listeners.Reset()
	for _, socket := range sockets {
		if socket.RemotePort == 0 {
			ms.listeners.Put(uint8(syscall.IPPROTO_TCP), socket.LocalIP, socket.LocalPort)
		}
	}

	return nil
}
