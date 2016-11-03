package nfs

import (
	"github.com/elastic/beats/libbeat/common"
)

type NFS struct {
	vers  uint32
	proc  uint32
	event common.MapStr
}

func (nfs *NFS) getRequestInfo(xdr *Xdr) common.MapStr {

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

func (nfs *NFS) getNFSReplyStatus(xdr *Xdr) string {
	switch nfs.proc {
	case 0:
		return NFSStatus[0]
	default:
		stat := int(xdr.getUInt())
		return NFSStatus[stat]
	}
}
