package nfs

import "fmt"

const (
	OP_ACCESS               = 3
	OP_CLOSE                = 4
	OP_COMMIT               = 5
	OP_CREATE               = 6
	OP_DELEGPURGE           = 7
	OP_DELEGRETURN          = 8
	OP_GETATTR              = 9
	OP_GETFH                = 10
	OP_LINK                 = 11
	OP_LOCK                 = 12
	OP_LOCKT                = 13
	OP_LOCKU                = 14
	OP_LOOKUP               = 15
	OP_LOOKUPP              = 16
	OP_NVERIFY              = 17
	OP_OPEN                 = 18
	OP_OPENATTR             = 19
	OP_OPEN_CONFIRM         = 20
	OP_OPEN_DOWNGRADE       = 21
	OP_PUTFH                = 22
	OP_PUTPUBFH             = 23
	OP_PUTROOTFH            = 24
	OP_READ                 = 25
	OP_READDIR              = 26
	OP_READLINK             = 27
	OP_REMOVE               = 28
	OP_RENAME               = 29
	OP_RENEW                = 30
	OP_RESTOREFH            = 31
	OP_SAVEFH               = 32
	OP_SECINFO              = 33
	OP_SETATTR              = 34
	OP_SETCLIENTID          = 35
	OP_SETCLIENTID_CONFIRM  = 36
	OP_VERIFY               = 37
	OP_WRITE                = 38
	OP_RELEASE_LOCKOWNER    = 39
	OP_BACKCHANNEL_CTL      = 40
	OP_BIND_CONN_TO_SESSION = 41
	OP_EXCHANGE_ID          = 42
	OP_CREATE_SESSION       = 43
	OP_DESTROY_SESSION      = 44
	OP_FREE_STATEID         = 45
	OP_GET_DIR_DELEGATION   = 46
	OP_GETDEVICEINFO        = 47
	OP_GETDEVICELIST        = 48
	OP_LAYOUTCOMMIT         = 49
	OP_LAYOUTGET            = 50
	OP_LAYOUTRETURN         = 51
	OP_SECINFO_NO_NAME      = 52
	OP_SEQUENCE             = 53
	OP_SET_SSV              = 54
	OP_TEST_STATEID         = 55
	OP_WANT_DELEGATION      = 56
	OP_DESTROY_CLIENTID     = 57
	OP_RECLAIM_COMPLETE     = 58
	OP_ILLEGAL              = 10044
)

var nfs_opnum4 = map[int]string{
	3:     "ACCESS",
	4:     "CLOSE",
	5:     "COMMIT",
	6:     "CREATE",
	7:     "DELEGPURGE",
	8:     "DELEGRETURN",
	9:     "GETATTR",
	10:    "GETFH",
	11:    "LINK",
	12:    "LOCK",
	13:    "LOCKT",
	14:    "LOCKU",
	15:    "LOOKUP",
	16:    "LOOKUPP",
	17:    "NVERIFY",
	18:    "OPEN",
	19:    "OPENATTR",
	20:    "OPEN_CONFIRM",
	21:    "OPEN_DOWNGRADE",
	22:    "PUTFH",
	23:    "PUTPUBFH",
	24:    "PUTROOTFH",
	25:    "READ",
	26:    "READDIR",
	27:    "READLINK",
	28:    "REMOVE",
	29:    "RENAME",
	30:    "RENEW",
	31:    "RESTOREFH",
	32:    "SAVEFH",
	33:    "SECINFO",
	34:    "SETATTR",
	35:    "SETCLIENTID",
	36:    "SETCLIENTID_CONFIRM",
	37:    "VERIFY",
	38:    "WRITE",
	39:    "RELEASE_LOCKOWNER",
	40:    "BACKCHANNEL_CTL",
	41:    "BIND_CONN_TO_SESSION",
	42:    "EXCHANGE_ID",
	43:    "CREATE_SESSION",
	44:    "DESTROY_SESSION",
	45:    "FREE_STATEID",
	46:    "GET_DIR_DELEGATION",
	47:    "GETDEVICEINFO",
	48:    "GETDEVICELIST",
	49:    "LAYOUTCOMMIT",
	50:    "LAYOUTGET",
	51:    "LAYOUTRETURN",
	52:    "SECINFO_NO_NAME",
	53:    "SEQUENCE",
	54:    "SET_SSV",
	55:    "TEST_STATEID",
	56:    "WANT_DELEGATION",
	57:    "DESTROY_CLIENTID",
	58:    "RECLAIM_COMPLETE",
	10044: "ILLEGAL",
}

func (nfs *Nfs) eatData(op int, xdr *Xdr) {

	switch op {
	case OP_GETATTR:
		xdr.getUIntVector()
	case OP_GETFH:
		// nothing to eat
	case OP_LOOKUP:
		xdr.getDynamicOpaque()
	case OP_LOOKUPP:
		// nothing to eat
	case OP_NVERIFY:
		xdr.getUIntVector()
		xdr.getDynamicOpaque()
	case OP_PUTFH:
		xdr.getDynamicOpaque()
	case OP_PUTPUBFH:
		// nothing to eat
	case OP_PUTROOTFH:
		// nothing to eat
	case OP_READLINK:
		// nothing to eat
	case OP_RENEW:
		xdr.getUHyper()
	case OP_RESTOREFH:
		// nothing to eat
	case OP_SAVEFH:
		// nothing to eat
	case OP_SECINFO:
		xdr.getDynamicOpaque()
	case OP_VERIFY:
		xdr.getUIntVector()
		xdr.getDynamicOpaque()
	case OP_SEQUENCE:
		xdr.getOpaque(16)
		xdr.getUInt()
		xdr.getUInt()
		xdr.getUInt()
		xdr.getUInt()

	}
}

// findV4MainOpcode finds the main operation in a compound call. If no main operation can be found, the last operation
// in compound call is returned.
//
// Compound requests group multiple nfs operations into a single request. Nevertheless, all compound requests are
// triggered by end-user activity, like 'ls', 'open', 'stat' and IO calls. Depending on which operations are combined
// the main operation can be different. For example, in compound:
//
// PUTFH + READDIR + GETATTR
//
// READDIR is the main operation. while in
//
// PUTFH + GETATTR
//
// GETATTR is the main operation.
func (nfs *Nfs) findV4MainOpcode(xdr *Xdr) string {

	// did we find a main operation opcode?
	found := false

	// default op code
	current_opname := "ILLEGAL"

	opcount := int(xdr.getUInt())
	for i := 0; !found && i < opcount; i++ {
		op := int(xdr.getUInt())
		opname, ok := nfs_opnum4[op]

		if !ok {
			return fmt.Sprintf("ILLEGAL (%d)", op)
		}
		current_opname = opname

		switch op {
		// First class ops
		//
		// The first class ops usually the main operation in the compound.
		// NFS spec allowes to build compound opertion where multiple
		// first class ops are used, like OPEN->LOCK->WRITE->LOCKU->CLOSE,
		// but such construnction are not used in the practice.
		case
			OP_ACCESS,
			OP_BACKCHANNEL_CTL,
			OP_BIND_CONN_TO_SESSION,
			OP_CLOSE,
			OP_COMMIT,
			OP_CREATE,
			OP_CREATE_SESSION,
			OP_DELEGPURGE,
			OP_DELEGRETURN,
			OP_DESTROY_CLIENTID,
			OP_DESTROY_SESSION,
			OP_EXCHANGE_ID,
			OP_FREE_STATEID,
			OP_GETDEVICEINFO,
			OP_GETDEVICELIST,
			OP_GET_DIR_DELEGATION,
			OP_LAYOUTCOMMIT,
			OP_LAYOUTGET,
			OP_LAYOUTRETURN,
			OP_LINK,
			OP_LOCK,
			OP_LOCKT,
			OP_LOCKU,
			OP_OPEN,
			OP_OPENATTR,
			OP_OPEN_CONFIRM,
			OP_OPEN_DOWNGRADE,
			OP_READ,
			OP_READDIR,
			OP_READLINK,
			OP_RECLAIM_COMPLETE,
			OP_RELEASE_LOCKOWNER,
			OP_REMOVE,
			OP_RENAME,
			OP_SECINFO_NO_NAME,
			OP_SETATTR,
			OP_SETCLIENTID,
			OP_SETCLIENTID_CONFIRM,
			OP_SET_SSV,
			OP_TEST_STATEID,
			OP_WANT_DELEGATION,
			OP_WRITE:

			found = true
		default:
			nfs.eatData(op, xdr)
		}
	}
	return current_opname
}
