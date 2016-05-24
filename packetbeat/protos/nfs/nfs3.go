package nfs

var nfs_opnum3 = [...]string{
	"NULL",
	"GETATTR",
	"SETATTR",
	"LOOKUP",
	"ACCESS",
	"READLINK",
	"READ",
	"WRITE",
	"CREATE",
	"MKDIR",
	"SYM_LINK",
	"MKNODE",
	"REMOVE",
	"RMDIR",
	"RENAME",
	"LINK",
	"READDIR",
	"READDIRPLUS",
	"FSSTAT",
	"FSINFO",
	"PATHINFO",
	"COMMIT",
}

func (nfs *Nfs) getV3Opcode(proc int) string {
	if proc < len(nfs_opnum3) {
		return nfs_opnum3[proc]
	} else {
		return "ILLEGAL"
	}
}
