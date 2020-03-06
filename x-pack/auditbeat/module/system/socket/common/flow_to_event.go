// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package common

import (
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/flowhash"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func (f *Flow) ToEvent(final bool) mb.Event {
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
		if f.Process != nil {
			process["name"] = f.Process.name
			process["args"] = f.Process.args
			process["executable"] = f.Process.path
			if f.Process.createdTime != (time.Time{}) {
				process["created"] = f.Process.createdTime
			}

			if f.Process.hasCreds {
				uid := strconv.Itoa(int(f.Process.uid))
				gid := strconv.Itoa(int(f.Process.gid))
				root.Put("user.id", uid)
				root.Put("group.id", gid)
				if name := userCache.LookupUID(uid); name != "" {
					root.Put("user.name", name)
				}
				if name := groupCache.LookupGID(gid); name != "" {
					root.Put("group.name", name)
				}
				metricset["uid"] = f.Process.uid
				metricset["gid"] = f.Process.gid
				metricset["euid"] = f.Process.euid
				metricset["egid"] = f.Process.egid
			}

			if domain, found := f.Process.ResolveIP(f.local.addr.IP); found {
				local["domain"] = domain
			}
			if domain, found := f.Process.ResolveIP(f.remote.addr.IP); found {
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
