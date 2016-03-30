package nfs

import (
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type Nfs struct {
	xdr   Xdr
	vers  uint32
	proc  uint32
	event common.MapStr
	ts    time.Time
}

func (nfs *Nfs) getRequestInfo() {

	nfsInfo := common.MapStr{}
	nfsInfo["version"] = nfs.vers

	switch nfs.vers {
	case 3:
		nfsInfo["opcode"] = nfs.getV3Opcode()
	case 4:
		switch nfs.proc {
		case 0:
			nfsInfo["opcode"] = "NULL"
		case 1:
			tag := nfs.xdr.getDynamicOpaque()
			nfsInfo["tag"] = string(tag)
			nfsInfo["minor_version"] = nfs.xdr.getUInt()
			nfsInfo["opcode"] = nfs.getV4Opcode()
		}
	}
	nfs.event["nfs"] = nfsInfo
}

func (nfs *Nfs) getReplyInfo(xdr *Xdr) {
	nfsInfo := nfs.event["nfs"].(common.MapStr)
	stat := int(xdr.getUInt())
	nfsInfo["status"] = NFS_STATUS[stat]
}
