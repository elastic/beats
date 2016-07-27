package nfs

import (
	"github.com/elastic/beats/libbeat/common"
)

type Nfs struct {
	vers  uint32
	proc  uint32
	event common.MapStr
}

func (nfs *Nfs) getRequestInfo(xdr *Xdr) common.MapStr {

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

func (nfs *Nfs) getNFSReplyStatus(xdr *Xdr) string {
	switch nfs.proc {
	case 0:
		return NFS_STATUS[0]
	default:
		stat := int(xdr.getUInt())
		return NFS_STATUS[stat]
	}
}
