// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package sockets

import (
	"net"

	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"

	"strconv"
	"syscall"

	"github.com/OneOfOne/xxhash"

	"github.com/elastic/beats/libbeat/logp"
	mbSocket "github.com/elastic/beats/metricbeat/module/system/socket"
	"github.com/elastic/gosigar/sys/linux"
)

const (
	moduleName    = "system"
	metricsetName = "sockets"
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

	netlink *mbSocket.NetlinkSession
	// TODO: Replace with process data collected in processes metricset
	ptable    *mbSocket.ProcTable
	listeners *mbSocket.ListenerTable
	// TODO: Replace with user data collected in host metricset
	users mbSocket.UserCache
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
	Direction    mbSocket.Direction
	UID          uint32
	Username     string
	ProcessPID   int
	ProcessName  string
	ProcessError error
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
			"id": s.UID,
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
// TODO: Extend beyond Linux.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	ptable, err := mbSocket.NewProcTable("")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create process table")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(moduleName),

		netlink:   mbSocket.NewNetlinkSession(),
		ptable:    ptable,
		listeners: mbSocket.NewListenerTable(),
		users:     mbSocket.NewUserCache(),
	}

	if config.ReportChanges {
		ms.cache = cache.New()
	}

	return ms, nil
}

// Fetch checks which sockets exist on the host and reports them.
// It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	sockets, err := ms.getSockets()
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}
	ms.log.Debugf("found %d sockets", len(sockets))

	err = ms.refreshEnrichmentData(sockets)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}

	if ms.cache != nil && !ms.cache.IsEmpty() {
		opened, closed := ms.cache.DiffAndUpdateCache(convertToCacheable(sockets))

		for _, socket := range opened {
			ms.enrichSocket(socket.(*Socket))
		}

		for _, socket := range closed {
			ms.enrichSocket(socket.(*Socket))
		}

		for _, s := range opened {
			socketMapStr := s.(*Socket).toMapStr()
			socketMapStr.Put("status", "OPENED")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"socket": socketMapStr,
				},
			})
		}

		for _, s := range closed {
			socketMapStr := s.(*Socket).toMapStr()
			socketMapStr.Put("status", "CLOSED")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"socket": socketMapStr,
				},
			})
		}
	} else {
		// Report all currently existing sockets
		for _, socket := range sockets {
			ms.enrichSocket(socket)
		}

		var socketMapStrAll []common.MapStr

		for _, socket := range sockets {
			socketMapStr := socket.toMapStr()
			socketMapStr.Put("status", "OPEN")

			socketMapStrAll = append(socketMapStrAll, socketMapStr)
		}

		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"socket": socketMapStrAll,
			},
		})

		if ms.cache != nil {
			// This will initialize the cache with the current sockets
			ms.cache.DiffAndUpdateCache(convertToCacheable(sockets))
		}
	}

	// Reset the listeners for the next iteration.
	ms.listeners.Reset()
}

func convertToCacheable(sockets []*Socket) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(sockets))

	for _, s := range sockets {
		c = append(c, s)
	}

	return c
}

func (ms *MetricSet) refreshEnrichmentData(allSockets []*Socket) error {
	// Register all listening sockets.
	for _, socket := range allSockets {
		if socket.RemotePort == 0 {
			ms.listeners.Put(uint8(syscall.IPPROTO_TCP), socket.LocalIP, socket.LocalPort)
		}
	}

	err := ms.ptable.Refresh()
	return errors.Wrap(err, "error refreshing process data")
}

func (ms *MetricSet) enrichSocket(socket *Socket) {
	socket.Username = ms.users.LookupUID(int(socket.UID))

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
}

func (ms *MetricSet) getSockets() ([]*Socket, error) {
	diags, err := ms.netlink.GetSocketList()
	if err != nil {
		return nil, errors.Wrap(err, "error getting sockets")
	}

	sockets := make([]*Socket, 0, len(diags))
	for _, diag := range diags {
		socket := &Socket{
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

		sockets = append(sockets, socket)
	}

	return sockets, nil
}
