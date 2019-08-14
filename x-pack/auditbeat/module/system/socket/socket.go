// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package socket

import (
	"encoding/binary"
	"fmt"
	"net"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/flowhash"
	"github.com/elastic/beats/libbeat/logp"
	sock "github.com/elastic/beats/metricbeat/helper/socket"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/beats/x-pack/auditbeat/module/system"
	"github.com/elastic/gosigar/sys/linux"
)

const (
	moduleName    = "system"
	metricsetName = "socket"
	namespace     = "system.audit.socket"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

type ipProtocol uint8

const (
	// TODO: Unify IP protocol constants in Beats
	tcp ipProtocol = 6
)

func (proto ipProtocol) String() string {
	switch proto {
	case tcp:
		return "tcp"
	default:
		return ""
	}
}

type eventAction uint8

const (
	eventActionExistingSocket eventAction = iota
	eventActionSocketOpened
	eventActionSocketClosed
)

func (action eventAction) String() string {
	switch action {
	case eventActionExistingSocket:
		return "existing_socket"
	case eventActionSocketOpened:
		return "socket_opened"
	case eventActionSocketClosed:
		return "socket_closed"
	default:
		return ""
	}
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
	)
}

// MetricSet collects data about sockets.
type MetricSet struct {
	system.SystemMetricSet
	config Config
	cache  *cache.Cache
	log    *logp.Logger

	netlink *sock.NetlinkSession
	// TODO: Replace with process data collected in processes metricset
	ptable    *sock.ProcTable
	listeners *sock.ListenerTable

	lastState time.Time
}

// Socket represents information about a socket.
type Socket struct {
	Family      linux.AddressFamily
	Protocol    ipProtocol
	LocalIP     net.IP
	LocalPort   int
	RemoteIP    net.IP
	RemotePort  int
	Inode       uint32
	Direction   sock.Direction
	UID         uint32
	Username    string
	ProcessPID  int
	ProcessName string
	Error       error
}

// newSocket creates a new socket out of a netlink diag message.
func newSocket(diag *linux.InetDiagMsg) *Socket {
	return &Socket{
		Family:     linux.AddressFamily(diag.Family),
		Protocol:   tcp,
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
	mapstr := common.MapStr{
		"network": common.MapStr{
			"direction": s.Direction.String(),
			"transport": s.Protocol.String(),
			"type":      s.Family.String(),
		},
		"user": common.MapStr{
			"id": s.UID,
		},
	}

	communityID := s.communityID()
	if communityID != "" {
		mapstr.Put("network.community_id", communityID)
	}

	if s.Username != "" {
		mapstr.Put("user.name", s.Username)
	}

	if s.ProcessName != "" {
		mapstr.Put("process", common.MapStr{
			"pid":  s.ProcessPID,
			"name": s.ProcessName,
		})
	}

	switch s.Direction {
	case sock.Outbound:
		mapstr.Put("source", common.MapStr{
			"ip":   s.LocalIP,
			"port": s.LocalPort,
		})
		mapstr.Put("destination", common.MapStr{
			"ip":   s.RemoteIP,
			"port": s.RemotePort,
		})
	case sock.Inbound:
		mapstr.Put("source", common.MapStr{
			"ip":   s.RemoteIP,
			"port": s.RemotePort,
		})
		mapstr.Put("destination", common.MapStr{
			"ip":   s.LocalIP,
			"port": s.LocalPort,
		})
	case sock.Listening:
		mapstr.Put("destination", common.MapStr{
			"ip":   s.LocalIP,
			"port": s.LocalPort,
		})
	}

	if s.Error != nil {
		mapstr.Put("error.message", s.Error.Error())
	}

	return mapstr
}

// entityID creates an ID that uniquely identifies this socket across machines.
func (s Socket) entityID(hostID string) string {
	h := system.NewEntityHash()
	h.Write([]byte(hostID))
	binary.Write(h, binary.LittleEndian, int64(s.Inode))
	h.Write(s.LocalIP)
	h.Write(s.RemoteIP)
	binary.Write(h, binary.LittleEndian, int64(s.LocalPort))
	binary.Write(h, binary.LittleEndian, int64(s.RemotePort))
	return h.Sum()
}

// communityID calculates the community ID of this socket.
func (s Socket) communityID() string {
	var flow flowhash.Flow

	switch s.Direction {
	case sock.Inbound:
		flow.SourceIP = s.RemoteIP
		flow.SourcePort = uint16(s.RemotePort)
		flow.DestinationIP = s.LocalIP
		flow.DestinationPort = uint16(s.LocalPort)
	case sock.Outbound:
		flow.SourceIP = s.LocalIP
		flow.SourcePort = uint16(s.LocalPort)
		flow.DestinationIP = s.RemoteIP
		flow.DestinationPort = uint16(s.RemotePort)
	default:
		// Listening socket, not a flow
		return ""
	}

	flow.Protocol = uint8(s.Protocol)

	return flowhash.CommunityID.Hash(flow)
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v/%v dataset is beta", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	ptable, err := sock.NewProcTable("")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create process table")
	}

	ms := &MetricSet{
		SystemMetricSet: system.NewSystemMetricSet(base),
		config:          config,
		log:             logp.NewLogger(metricsetName),
		cache:           cache.New(),
		netlink:         sock.NewNetlinkSession(),
		ptable:          ptable,
		listeners:       sock.NewListenerTable(),
	}

	return ms, nil
}

// Fetch collects the user information. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	needsStateUpdate := time.Since(ms.lastState) > ms.config.effectiveStatePeriod()
	if needsStateUpdate {
		ms.log.Debug("Sending state")
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
	ms.lastState = time.Now()

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
	ms.refreshEnrichments(sockets)

	for _, socket := range sockets {
		err = ms.enrichSocket(socket)
		if err != nil {
			return err
		}

		event := ms.socketEvent(socket, eventTypeState, eventActionExistingSocket)
		event.RootFields.Put("event.id", stateID.String())
		report.Event(event)
	}

	// This will initialize the cache with the current sockets
	ms.cache.DiffAndUpdateCache(convertToCacheable(sockets))

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
		ms.refreshEnrichments(sockets)

		for _, s := range opened {
			err = ms.enrichSocket(s.(*Socket))
			if err != nil {
				return err
			}

			report.Event(ms.socketEvent(s.(*Socket), eventTypeEvent, eventActionSocketOpened))
		}
	}

	for _, s := range closed {
		report.Event(ms.socketEvent(s.(*Socket), eventTypeEvent, eventActionSocketClosed))
	}

	return nil
}

func (ms *MetricSet) socketEvent(socket *Socket, eventType string, action eventAction) mb.Event {
	event := mb.Event{
		RootFields: socket.toMapStr(),
	}

	event.RootFields.Put("event.kind", eventType)
	event.RootFields.Put("event.action", action.String())
	event.RootFields.Put("message", socketMessage(socket, action))

	if ms.HostID() != "" {
		event.RootFields.Put("socket.entity_id", socket.entityID(ms.HostID()))
	}

	return event
}

func socketMessage(socket *Socket, action eventAction) string {
	var actionString string
	switch action {
	case eventActionSocketOpened:
		actionString = "OPENED"
	case eventActionSocketClosed:
		actionString = "CLOSED"
	case eventActionExistingSocket:
		actionString = "OPEN"
	}

	var endpointString string
	switch socket.Direction {
	case sock.Inbound:
		endpointString = fmt.Sprintf("%v:%d -> %v:%d",
			socket.RemoteIP, socket.RemotePort, socket.LocalIP, socket.LocalPort)
	case sock.Outbound:
		endpointString = fmt.Sprintf("%v:%d -> %v:%d",
			socket.LocalIP, socket.LocalPort, socket.RemoteIP, socket.RemotePort)
	case sock.Listening:
		endpointString = fmt.Sprintf("%v:%d", socket.LocalIP, socket.LocalPort)
	}

	return fmt.Sprintf("%v socket (%v) %v by process %v (PID: %d) and user %v (UID: %d)",
		strings.Title(socket.Direction.String()), endpointString, actionString,
		socket.ProcessName, socket.ProcessPID, socket.Username, socket.UID)
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

	socket.Direction = ms.listeners.Direction(uint8(socket.Family), uint8(syscall.IPPROTO_TCP),
		socket.LocalIP, socket.LocalPort, socket.RemoteIP, socket.RemotePort)

	if ms.ptable != nil {
		proc := ms.ptable.ProcessBySocketInode(socket.Inode)
		if proc != nil {
			// Add process info by finding the process that holds the socket's inode.
			socket.ProcessPID = proc.PID
			socket.ProcessName = proc.Command
		} else if socket.Inode == 0 {
			socket.Error = fmt.Errorf("process has exited (inode=%v)", socket.Inode)
		} else {
			socket.Error = fmt.Errorf("process not found (inode=%v)", socket.Inode)
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
		socket := newSocket(diag)

		if !ms.config.IncludeLocalhost &&
			(socket.LocalIP.IsLoopback() || socket.RemoteIP.IsLoopback()) {

			continue
		}

		sockets = append(sockets, socket)
	}

	return sockets, nil
}

func (ms *MetricSet) refreshEnrichments(sockets []*Socket) {
	// Refresh inode to process mapping for process enrichment
	err := ms.ptable.Refresh()
	if err != nil {
		// Errors here can happen, e.g. if a process exits while its /proc is being read.
		ms.log.Warn(errors.Wrap(err, "error refreshing process data"))
	}

	// Register all listening sockets
	ms.listeners.Reset()
	for _, socket := range sockets {
		if socket.RemotePort == 0 {
			ms.listeners.Put(uint8(syscall.IPPROTO_TCP), socket.LocalIP, socket.LocalPort)
		}
	}
}
