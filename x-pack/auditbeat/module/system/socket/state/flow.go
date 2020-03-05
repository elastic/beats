// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/flowhash"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

type Flow struct {
	*Socket

	socket uintptr

	inetType          InetType
	proto             FlowProto
	direction         FlowDirection
	created, lastSeen uint64 // kernel time
	pid               uint32
	local, remote     *Endpoint
	complete          bool

	// these are automatically calculated by state from kernelTimes above
	createdTime, lastSeenTime time.Time
}

func NewFlow(sock uintptr, pid uint32, inetType, proto uint16, lastSeen uint64, local, remote *Endpoint) *Flow {
	return &Flow{
		socket:   sock,
		pid:      pid,
		inetType: InetType(inetType),
		proto:    FlowProto(proto),
		lastSeen: lastSeen,
		local:    local,
		remote:   remote,
	}
}

func (f *Flow) MarkOutbound() *Flow {
	f.direction = DirectionOutbound
	return f
}

func (f *Flow) MarkInbound() *Flow {
	f.direction = DirectionInbound
	return f
}

func (f *Flow) MarkComplete() *Flow {
	f.complete = true
	return f
}

func (f *Flow) SetCreated(ts uint64) *Flow {
	f.created = ts
	return f
}

func (f *Flow) Terminate() {
	if f.Socket != nil && f.hasKey() {
		f.Socket.RemoveFlow(f.key())
	}
}

// If this flow should be reported or only captured partial data
func (f *Flow) IsValid() bool {
	return f.inetType != InetTypeUnknown && f.proto != ProtoUnknown && f.local.addr.IP != nil && f.remote.addr.IP != nil
}

func (f *Flow) Local() *Endpoint {
	return f.local
}

func (f *Flow) Remote() *Endpoint {
	return f.remote
}

func (f *Flow) Type() InetType {
	return f.inetType
}

func (f *Flow) LocalIP() net.IP {
	if f.local != nil {
		return f.local.addr.IP
	}
	return nil
}

func (f *Flow) RemoteIP() net.IP {
	if f.remote != nil {
		return f.remote.addr.IP
	}
	return nil
}

func (f *Flow) IsUDP() bool {
	return f.proto == ProtoUDP
}

func (f *Flow) hasKey() bool {
	return f.Remote() != nil
}

func (f *Flow) key() string {
	return f.remote.addr.String()
}

func (f *Flow) updateWith(ref *Flow) {
	f.lastSeenTime = ref.lastSeenTime
	if f.inetType == InetTypeUnknown {
		f.inetType = ref.inetType
	}

	if f.proto == ProtoUnknown {
		f.proto = ref.proto
	}

	if f.pid == 0 {
		f.pid = ref.pid
		f.process = ref.process
	}

	if f.process == nil {
		if ref.process != nil && f.pid == ref.pid {
			f.process = ref.process
		}
	}

	if f.direction == DirectionUnknown {
		f.direction = ref.direction
	}

	f.complete = ref.complete
	f.local.updateWith(ref.local)
	f.remote.updateWith(ref.remote)
}

func (f *Flow) ToEvent(final bool) (ev mb.Event) {
	localAddr := f.local.addr
	remoteAddr := f.remote.addr

	local := common.MapStr{
		"ip":      localAddr.IP.String(),
		"port":    localAddr.Port,
		"packets": f.local.packets,
		"bytes":   f.local.bytes,
	}

	remote := common.MapStr{
		"ip":      remoteAddr.IP.String(),
		"port":    remoteAddr.Port,
		"packets": f.remote.packets,
		"bytes":   f.remote.bytes,
	}

	src, dst := local, remote
	if f.direction == DirectionInbound {
		src, dst = dst, src
	}

	inetType := f.inetType
	// Under Linux, a socket created as AF_INET6 can receive IPv4 connections
	// and it will use the IPv4 stack.
	// This results in src and dst address using IPv4 mapped addresses (which
	// Golang converts to IPv4 automatically). It will be misleading to report
	// network.type: ipv6 and have v4 addresses, so it's better to report
	// a network.type of ipv4 (which also matches the actual stack used).
	if inetType == InetTypeIPv6 && f.local.addr.IP.To4() != nil && f.remote.addr.IP.To4() != nil {
		inetType = InetTypeIPv4
	}
	root := common.MapStr{
		"source":      src,
		"client":      src,
		"destination": dst,
		"server":      dst,
		"network": common.MapStr{
			"direction": f.direction.String(),
			"type":      inetType.String(),
			"transport": f.proto.String(),
			"packets":   f.local.packets + f.remote.packets,
			"bytes":     f.local.bytes + f.remote.bytes,
			"community_id": flowhash.CommunityID.Hash(flowhash.Flow{
				SourceIP:        localAddr.IP,
				SourcePort:      uint16(localAddr.Port),
				DestinationIP:   remoteAddr.IP,
				DestinationPort: uint16(remoteAddr.Port),
				Protocol:        uint8(f.proto),
			}),
		},
		"event": common.MapStr{
			"kind":     "event",
			"action":   "network_flow",
			"category": "network_traffic",
			"start":    f.createdTime,
			"end":      f.lastSeenTime,
			"duration": f.lastSeenTime.Sub(f.createdTime).Nanoseconds(),
		},
		"flow": common.MapStr{
			"final":    final,
			"complete": f.complete,
		},
	}

	metricset := common.MapStr{
		"kernel_sock_address": fmt.Sprintf("0x%x", f.socket),
	}

	if f.pid != 0 {
		process := common.MapStr{
			"pid": int(f.pid),
		}
		if f.process != nil {
			process["name"] = f.process.name
			process["args"] = f.process.args
			process["executable"] = f.process.path
			if f.process.createdTime != (time.Time{}) {
				process["created"] = f.process.createdTime
			}

			if f.process.hasCreds {
				uid := strconv.Itoa(int(f.process.uid))
				gid := strconv.Itoa(int(f.process.gid))
				root.Put("user.id", uid)
				root.Put("group.id", gid)
				if name := userCache.LookupUID(uid); name != "" {
					root.Put("user.name", name)
				}
				if name := groupCache.LookupGID(gid); name != "" {
					root.Put("group.name", name)
				}
				metricset["uid"] = f.process.uid
				metricset["gid"] = f.process.gid
				metricset["euid"] = f.process.euid
				metricset["egid"] = f.process.egid
			}

			if domain, found := f.process.ResolveIP(f.local.addr.IP); found {
				local["domain"] = domain
			}
			if domain, found := f.process.ResolveIP(f.remote.addr.IP); found {
				remote["domain"] = domain
			}
		}
		root["process"] = process
	}

	return mb.Event{
		RootFields:      root,
		MetricSetFields: metricset,
	}
}
