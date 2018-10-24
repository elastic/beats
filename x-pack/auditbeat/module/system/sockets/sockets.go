// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package sockets

import (
	"github.com/pkg/errors"
	"net"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"

	"github.com/OneOfOne/xxhash"
	"github.com/elastic/beats/libbeat/logp"
	mbSocket "github.com/elastic/beats/metricbeat/module/system/socket"
	"github.com/elastic/gosigar/sys/linux"
	"github.com/joeshaw/multierror"
	"strconv"
)

const (
	moduleName    = "system"
	metricsetName = "sockets"

	ProcessType_PROCESS = "process"
	ProcessType_RPC     = "rpc"
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
	// TODO: Maybe at some point get this from the processes metricset in Auditbeat
	// instead of reading /proc again?
	ptable     *mbSocket.ProcTable
	rpcPortmap *RpcPortMap
}

// Socket represents information about a socket.
type Socket struct {
	Family      linux.AddressFamily
	State       linux.TCPState
	LocalIP     net.IP
	LocalPort   int
	RemoteIP    net.IP
	RemotePort  int
	Inode       uint32
	UID         uint32
	ProcessPID  int
	ProcessName string
	ProcessType string // set to "rpc" if socket was held by RPC service, otherwise empty
}

// Hash creates a hash for Socket.
func (s Socket) Hash() uint64 {
	h := xxhash.New64()

	// Local ports of RPC sockets change all the time so cannot be used for hashing
	//h.WriteString(strconv.Itoa(s.LocalPort))

	h.WriteString(s.LocalIP.String())
	h.WriteString(s.RemoteIP.String())
	h.WriteString(strconv.Itoa(s.RemotePort))
	h.WriteString(strconv.FormatUint(uint64(s.Inode), 10))
	h.WriteString(strconv.Itoa(s.ProcessPID))
	return h.Sum64()
}

func (s Socket) toMapStr() common.MapStr {
	return common.MapStr{
		"family":      s.Family.String(),
		"state":       s.State.String(),
		"local.ip":    s.LocalIP,
		"local.port":  s.LocalPort,
		"remote.ip":   s.RemoteIP,
		"remote.port": s.RemotePort,
		"inode":       s.Inode,
		"uid":         s.UID,

		"process": common.MapStr{
			"pid":  s.ProcessPID,
			"name": s.ProcessName,
			"type": s.ProcessType,
		},
	}
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

		netlink: mbSocket.NewNetlinkSession(),
		ptable:  ptable,
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

	if ms.cache != nil && !ms.cache.IsEmpty() {
		openedFromCache, closedfromCache := ms.cache.DiffAndUpdateCache(convertToCacheable(sockets))
		opened := convertToSocket(openedFromCache)
		closed := convertToSocket(closedfromCache)

		err = ms.enrichSockets(opened, closed)
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
			// purposely not returning - we only missed some enrichment
		}

		for _, s := range opened {
			socketMapStr := s.toMapStr()
			socketMapStr.Put("status", "OPENED")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"socket": socketMapStr,
				},
			})
		}

		for _, s := range closed {
			socketMapStr := s.toMapStr()
			socketMapStr.Put("status", "CLOSED")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"socket": socketMapStr,
				},
			})
		}
	} else {
		err = ms.enrichSockets(sockets)
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
			// purposely not returning - we only missed some enrichment
		}

		// Report all currently existing sockets
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
}

func convertToSocket(objects []interface{}) []*Socket {
	sockets := make([]*Socket, 0, len(objects))

	for _, o := range objects {
		sockets = append(sockets, o.(*Socket))
	}

	return sockets
}

func convertToCacheable(sockets []*Socket) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(sockets))

	for _, s := range sockets {
		c = append(c, s)
	}

	return c
}

func (ms *MetricSet) enrichSockets(socketLists ...[]*Socket) error {
	var errs multierror.Errors

	err := ms.ptable.Refresh()
	if err != nil {
		errs = append(errs, err)
	}

	for _, socketList := range socketLists {
		for _, socket := range socketList {
			proc := ms.ptable.ProcessBySocketInode(socket.Inode)
			if proc != nil {
				// Add process info by finding the process that holds the socket's inode.
				socket.ProcessPID = proc.PID
				socket.ProcessName = proc.Command
				socket.ProcessType = ProcessType_PROCESS
			} else {
				// If no process holds the inode, try find an RPC service that is using the socket's port
				if ms.rpcPortmap == nil {
					ms.rpcPortmap, err = GetRpcPortmap()
					if err != nil {
						errs = append(errs, err)
					}

					ms.log.Debugf("found %d ports mapped to RPC services", len(*ms.rpcPortmap))
				}

				if len(*ms.rpcPortmap) > 0 {
					rpcServices, found := (*ms.rpcPortmap)[uint32(socket.LocalPort)]

					if !found {
						rpcServices, found = (*ms.rpcPortmap)[uint32(socket.RemotePort)]
					}

					if found {
						// TODO: We assume every RPC service bound to a port is the same, is that true?
						socket.ProcessPID = -1
						socket.ProcessName = rpcServices[0].programName
						socket.ProcessType = ProcessType_RPC
					}
				}
			}
		}
	}

	ms.rpcPortmap = nil
	return errs.Err()
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
		}

		sockets = append(sockets, socket)
	}

	return sockets, nil
}
