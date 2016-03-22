package nfs

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/publish"
	"time"
)

const (
	RPC_CALL  = 0
	RPC_REPLY = 1
)

const NFS_PROGRAM_NUMBER = 100003

type RpcMessage struct {
	ts  time.Time
	xdr Xdr
}

var ACCEPT_STATUS = [...]string{
	"success",
	"prog_unavail",
	"prog_mismatch",
	"proc_unavail",
	"garbage_args",
	"system_err",
}

var calls_seen = common.NewCache(1*time.Minute, 8192)

func (msg *RpcMessage) fillEvent(event common.MapStr, results publish.Transactions, size int) {

	xid := fmt.Sprintf("%.8x", msg.xdr.getUInt())

	msgType := msg.xdr.getUInt()

	if msgType == RPC_CALL {

		// eat rpc version number
		msg.xdr.getUInt()

		rpcProg := msg.xdr.getUInt()
		rpcProgVers := msg.xdr.getUInt()
		rpcProc := msg.xdr.getUInt()

		if rpcProg == NFS_PROGRAM_NUMBER {

			// build event only if it's a nfs packet
			rpcInfo := common.MapStr{}
			rpcInfo["xid"] = xid
			rpcInfo["call_size"] = size

			auth_flavor := msg.xdr.getUInt()
			auth_opaque := msg.xdr.getDynamicOpaque()
			switch auth_flavor {
			case 0:
				rpcInfo["auth_flavor"] = "none"
			case 1:
				rpcInfo["auth_flavor"] = "unix"
				cred := common.MapStr{}
				credXdr := Xdr{data: auth_opaque, offset: 0}
				cred["stamp"] = credXdr.getUInt()
				cred["machinename"] = credXdr.getString()
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
			msg.xdr.getUInt()
			msg.xdr.getDynamicOpaque()

			event["type"] = "nfs"
			event["rpc"] = rpcInfo
			nfs := Nfs{ts: msg.ts, event: event, xdr: msg.xdr, vers: rpcProgVers, proc: rpcProc}
			nfs.getRequestInfo()

			// populate cache to trach request processing time
			calls_seen.Put(xid, &nfs)
		}

	} else {
		replyStatus := msg.xdr.getUInt()
		// we are interesed only in accepted rpc reply
		if replyStatus != 0 {
			return
		}

		// eat auth verifier
		msg.xdr.getUInt()
		msg.xdr.getDynamicOpaque()

		// get cached request
		v := calls_seen.Delete(xid)
		if v != nil {
			nfs := *(v.(*Nfs))
			rpcInfo := nfs.event["rpc"].(common.MapStr)
			rpcInfo["reply_size"] = size
			rpcTime := msg.ts.Sub(nfs.ts)
			rpcInfo["time"] = rpcTime
			// the same in human readable form
			rpcInfo["time_str"] = fmt.Sprintf("%v", rpcTime)
			acceptStatus := int(msg.xdr.getUInt())
			rpcInfo["status"] = ACCEPT_STATUS[acceptStatus]
			// populate nfs info for seccessfully executed requests
			if acceptStatus == 0 {
				nfs.getReplyInfo(&msg.xdr)
			}
			results.PublishTransaction(nfs.event)
		}
	}
}
