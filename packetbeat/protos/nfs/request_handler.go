// This file contains methods process RPC calls
package nfs

import (
	"expvar"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

const NFS_PROGRAM_NUMBER = 100003

var ACCEPT_STATUS = [...]string{
	"success",
	"prog_unavail",
	"prog_mismatch",
	"proc_unavail",
	"garbage_args",
	"system_err",
}

var (
	unmatchedRequests = expvar.NewInt("nfs.unmatched_requests")
)

// called by Cache, when re reply seen within expected time window
func (rpc *Rpc) handleExpiredPacket(nfs *Nfs) {
	nfs.event["status"] = "NO_REPLY"
	rpc.results.PublishTransaction(nfs.event)
	unmatchedRequests.Add(1)
}

// called when we process a RPC call
func (rpc *Rpc) handleCall(xid string, xdr *Xdr, ts time.Time, tcptuple *common.TcpTuple, dir uint8) {

	// eat rpc version number
	xdr.getUInt()
	rpcProg := xdr.getUInt()
	if rpcProg != NFS_PROGRAM_NUMBER {
		// not a NFS request
		return
	}

	src := common.Endpoint{
		Ip:   tcptuple.Src_ip.String(),
		Port: tcptuple.Src_port,
	}
	dst := common.Endpoint{
		Ip:   tcptuple.Dst_ip.String(),
		Port: tcptuple.Dst_port,
	}

	// The direction of the stream is based in the direction of first packet seen.
	// if we have stored stream in reverse order, swap src and dst
	if dir == tcp.TcpDirectionReverse {
		src, dst = dst, src
	}

	event := common.MapStr{}
	event["@timestamp"] = common.Time(ts)
	event["status"] = common.OK_STATUS // all packages are OK for now
	event["src"] = &src
	event["dst"] = &dst

	nfsVers := xdr.getUInt()
	nfsProc := xdr.getUInt()

	// build event only if it's a nfs packet
	rpcInfo := common.MapStr{}
	rpcInfo["xid"] = xid
	rpcInfo["call_size"] = xdr.size()

	auth_flavor := xdr.getUInt()
	auth_opaque := xdr.getDynamicOpaque()
	switch auth_flavor {
	case 0:
		rpcInfo["auth_flavor"] = "none"
	case 1:
		rpcInfo["auth_flavor"] = "unix"
		cred := common.MapStr{}
		credXdr := Xdr{data: auth_opaque, offset: 0}
		cred["stamp"] = credXdr.getUInt()
		machine := credXdr.getString()
		if machine == "" {
			machine = src.Ip
		}
		cred["machinename"] = machine
		cred["uid"] = credXdr.getUInt()
		cred["gid"] = credXdr.getUInt()
		cred["gids"] = credXdr.getUIntVector()
		rpcInfo["cred"] = cred
	case 6:
		rpcInfo["auth_flavor"] = "rpcsec_gss"
	default:
		rpcInfo["auth_flavor"] = fmt.Sprintf("unknown (%d)", auth_flavor)
	}

	// eat auth verifier
	xdr.getUInt()
	xdr.getDynamicOpaque()

	event["type"] = "nfs"
	event["rpc"] = rpcInfo
	nfs := Nfs{vers: nfsVers, proc: nfsProc, event: event}
	event["nfs"] = nfs.getRequestInfo(xdr)

	// use xid+src ip to uniquely identify request
	reqId := xid + tcptuple.Src_ip.String()

	// populate cache to trace request reply
	rpc.callsSeen.Put(reqId, &nfs)
}

// called when we process a RPC reply
func (rpc *Rpc) handleReply(xid string, xdr *Xdr, ts time.Time, tcptuple *common.TcpTuple, dir uint8) {
	replyStatus := xdr.getUInt()
	// we are interested only in accepted rpc reply
	if replyStatus != 0 {
		return
	}

	// eat auth verifier
	xdr.getUInt()
	xdr.getDynamicOpaque()

	// xid+src ip is used to uniquely identify request.
	var reqId string
	if dir == tcp.TcpDirectionReverse {
		// stream in correct order: Src points to a client
		reqId = xid + tcptuple.Src_ip.String()
	} else {
		// stream in reverse order: Dst points to a client
		reqId = xid + tcptuple.Dst_ip.String()
	}

	// get cached request
	v := rpc.callsSeen.Delete(reqId)
	if v != nil {
		nfs := v.(*Nfs)
		event := nfs.event
		rpcInfo := event["rpc"].(common.MapStr)
		rpcInfo["reply_size"] = xdr.size()
		rpcTime := ts.Sub(time.Time(event["@timestamp"].(common.Time)))
		rpcInfo["time"] = rpcTime
		// the same in human readable form
		rpcInfo["time_str"] = fmt.Sprintf("%v", rpcTime)
		acceptStatus := int(xdr.getUInt())
		rpcInfo["status"] = ACCEPT_STATUS[acceptStatus]

		// populate nfs info for successfully executed requests
		if acceptStatus == 0 {
			nfsInfo := event["nfs"].(common.MapStr)
			nfsInfo["status"] = nfs.getNFSReplyStatus(xdr)
		}
		rpc.results.PublishTransaction(event)
	}
}
