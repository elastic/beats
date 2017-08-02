package nfs

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type nfs struct {
	vers  uint32
	proc  uint32
	event beat.Event
}

func (nfs *nfs) getRequestInfo(xdr *xdr) common.MapStr {
	nfsInfo := common.MapStr{}
	nfsInfo["version"] = nfs.vers

	switch nfs.vers {
	case 3:
		nfsInfo["opcode"] = nfs.getV3Opcode(int(nfs.proc))
	case 4:
		switch nfs.proc {
		case 0:
			nfsInfo["opcode"] = "NULL"
		case 1:
			tag := xdr.getDynamicOpaque()
			nfsInfo["tag"] = string(tag)
			nfsInfo["minor_version"] = xdr.getUInt()
			nfsInfo["opcode"] = nfs.findV4MainOpcode(xdr)
		}
	}
	return nfsInfo
}

func (nfs *nfs) getNFSReplyStatus(xdr *xdr) string {
	switch nfs.proc {
	case 0:
		return nfsStatus[0]
	default:
		stat := int(xdr.getUInt())
		return nfsStatus[stat]
	}
}
