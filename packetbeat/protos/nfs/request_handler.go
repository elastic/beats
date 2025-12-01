// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package nfs

// This file contains methods process RPC calls

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
)

const nfsProgramNumber = 100003

var acceptStatus = [...]string{
	"success",
	"prog_unavail",
	"prog_mismatch",
	"proc_unavail",
	"garbage_args",
	"system_err",
}

var unmatchedRequests = monitoring.NewInt(nil, "nfs.unmatched_requests")

// called by Cache, when re reply seen within expected time window
func (r *rpc) handleExpiredPacket(nfs *nfs) {
	nfs.event.Fields["status"] = "NO_REPLY"
	r.results(nfs.event)
	unmatchedRequests.Add(1)
}

// called when we process a RPC call
func (r *rpc) handleCall(xid string, xdr *xdr, ts time.Time, tcptuple *common.TCPTuple, dir uint8) {
	if _, err := xdr.getUInt(); dropMalformed("rpc call version", err) {
		return
	}
	rpcProg, err := xdr.getUInt()
	if dropMalformed("rpc call program", err) {
		return
	}
	if rpcProg != nfsProgramNumber {
		// not a NFS request
		return
	}

	src := common.Endpoint{
		IP:   tcptuple.SrcIP.String(),
		Port: tcptuple.SrcPort,
	}
	dst := common.Endpoint{
		IP:   tcptuple.DstIP.String(),
		Port: tcptuple.DstPort,
	}

	// The direction of the stream is based in the direction of first packet seen.
	// if we have stored stream in reverse order, swap src and dst
	if dir == tcp.TCPDirectionReverse {
		src, dst = dst, src
	}

	evt, pbf := pb.NewBeatEvent(ts)
	pbf.SetSource(&src)
	pbf.AddIP(src.IP)
	pbf.SetDestination(&dst)
	pbf.AddIP(dst.IP)
	pbf.Source.Bytes = int64(xdr.size())
	pbf.Event.Dataset = "nfs"
	pbf.Event.Start = ts
	pbf.Network.Transport = "tcp"

	nfsVers, err := xdr.getUInt()
	if dropMalformed("rpc call nfs version", err) {
		return
	}
	nfsProc, err := xdr.getUInt()
	if dropMalformed("rpc call nfs procedure", err) {
		return
	}

	switch nfsVers {
	case 3:
		pbf.Network.Protocol = "nfsv3"
	case 4:
		pbf.Network.Protocol = "nfsv4"
	default:
		pbf.Network.Protocol = "nfs"
	}

	// build event only if it's a nfs packet
	rpcInfo := mapstr.M{
		"xid": xid,
	}

	fields := evt.Fields

	authFlavor, err := xdr.getUInt()
	if dropMalformed("rpc auth flavor", err) {
		return
	}
	authOpaque, err := xdr.getDynamicOpaque()
	if dropMalformed("rpc auth opaque", err) {
		return
	}
	switch authFlavor {
	case 0:
		rpcInfo["auth_flavor"] = "none"
	case 1:
		rpcInfo["auth_flavor"] = "unix"
		cred := mapstr.M{}
		credXdr := makeXDR(authOpaque)
		stamp, err := credXdr.getUInt()
		if dropMalformed("rpc unix cred stamp", err) {
			return
		}
		cred["stamp"] = stamp
		machine, err := credXdr.getString()
		if dropMalformed("rpc unix cred machinename", err) {
			return
		}
		if machine == "" {
			machine = src.IP
		} else {
			pbf.Source.Domain = machine
		}
		cred["machinename"] = machine
		fields["host.hostname"] = machine

		uid, err := credXdr.getUInt()
		if dropMalformed("rpc unix cred uid", err) {
			return
		}
		cred["uid"] = uid
		fields["user.id"] = uid

		gid, err := credXdr.getUInt()
		if dropMalformed("rpc unix cred gid", err) {
			return
		}
		cred["gid"] = gid
		fields["group.id"] = gid

		gids, err := credXdr.getUIntVector()
		if dropMalformed("rpc unix cred gids", err) {
			return
		}
		cred["gids"] = gids
		rpcInfo["cred"] = cred
	case 6:
		rpcInfo["auth_flavor"] = "rpcsec_gss"
	default:
		rpcInfo["auth_flavor"] = fmt.Sprintf("unknown (%d)", authFlavor)
	}

	// eat auth verifier
	if _, err := xdr.getUInt(); dropMalformed("rpc verifier flavor", err) {
		return
	}
	if _, err := xdr.getDynamicOpaque(); dropMalformed("rpc verifier body", err) {
		return
	}

	fields["status"] = common.OK_STATUS // all packages are OK for now
	fields["type"] = pbf.Event.Dataset
	fields["rpc"] = rpcInfo
	nfs := nfs{
		vers:  nfsVers,
		proc:  nfsProc,
		pbf:   pbf,
		event: evt,
	}
	info, err := nfs.getRequestInfo(xdr)
	if dropMalformed("nfs request info", err) {
		return
	}
	fields["nfs"] = info

	if opcode, ok := info["opcode"].(string); ok && opcode != "" {
		pbf.Event.Action = "nfs." + opcode
	}

	// use xid+src ip to uniquely identify request
	reqID := xid + tcptuple.SrcIP.String()

	// populate cache to trace request reply
	r.callsSeen.Put(reqID, &nfs)
}

// called when we process a RPC reply
func (r *rpc) handleReply(xid string, xdr *xdr, ts time.Time, tcptuple *common.TCPTuple, dir uint8) {
	replyStatus, err := xdr.getUInt()
	if dropMalformed("rpc reply status", err) {
		return
	}
	// we are interested only in accepted rpc reply
	if replyStatus != 0 {
		return
	}

	// eat auth verifier
	if _, err := xdr.getUInt(); dropMalformed("rpc reply verifier flavor", err) {
		return
	}
	if _, err := xdr.getDynamicOpaque(); dropMalformed("rpc reply verifier body", err) {
		return
	}

	// xid+src ip is used to uniquely identify request.
	var reqID string
	if dir == tcp.TCPDirectionReverse {
		// stream in correct order: Src points to a client
		reqID = xid + tcptuple.SrcIP.String()
	} else {
		// stream in reverse order: Dst points to a client
		reqID = xid + tcptuple.DstIP.String()
	}

	// get cached request
	v := r.callsSeen.Delete(reqID)
	if v != nil {
		nfs, ok := v.(*nfs)
		if !ok {
			logp.Warn("nfs: failed to assert nfs interface")
			return

		}
		nfs.pbf.Event.End = ts
		nfs.pbf.Destination.Bytes = int64(xdr.size())

		fields := nfs.event.Fields
		rpcInfo, ok := fields["rpc"].(mapstr.M)
		if !ok {
			logp.Warn("nfs: failed to assert map[string]")
			return

		}
		statusVal, err := xdr.getUInt()
		if dropMalformed("rpc accept status", err) {
			return
		}
		status := int(statusVal)
		if status < 0 || status >= len(acceptStatus) {
			logp.Warn("nfs: rpc accept status %d out of range", status)
			return
		}
		rpcInfo["status"] = acceptStatus[status]

		// populate nfs info for successfully executed requests
		if status == 0 {
			nfsInfo, ok := fields["nfs"].(mapstr.M)
			if !ok {
				logp.Warn("nfs: failed to assert map[string]")
				return

			}
			nfsStatus, err := nfs.getNFSReplyStatus(xdr)
			if dropMalformed("nfs reply status", err) {
				return
			}
			nfsInfo["status"] = nfsStatus
		} else {
			nfs.pbf.Event.Outcome = "failure"
		}
		r.results(nfs.event)
	}
}
