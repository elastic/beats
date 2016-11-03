package nfs

import "fmt"

const (
	OpAccess             = 3
	OpClose              = 4
	OpCommit             = 5
	OpCreate             = 6
	OpDelegpurge         = 7
	OpDelegreturn        = 8
	OpGetattr            = 9
	OpGetfh              = 10
	OpLink               = 11
	OpLock               = 12
	OpLockt              = 13
	OpLocku              = 14
	OpLookup             = 15
	OpLookupp            = 16
	OpNverify            = 17
	OpOpen               = 18
	OpOpenattr           = 19
	OpOpenConfirm        = 20
	OpOpenDowngrade      = 21
	OpPutfh              = 22
	OpPutpubfh           = 23
	OpPutrootfh          = 24
	OpRead               = 25
	OpReaddir            = 26
	OpReadlink           = 27
	OpRemove             = 28
	OpRename             = 29
	OpRenew              = 30
	OpRestorefh          = 31
	OpSavefh             = 32
	OpSecinfo            = 33
	OpSetattr            = 34
	OpSetclientid        = 35
	OpSetclientidConfirm = 36
	OpVerify             = 37
	OpWrite              = 38
	OpReleaseLockowner   = 39
	OpBackchannelCtl     = 40
	OpBindConnToSession  = 41
	OpExchangeID         = 42
	OpCreateSession      = 43
	OpDestroySession     = 44
	OpFreeStateid        = 45
	OpGetDirDelegation   = 46
	OpGetdeviceinfo      = 47
	OpGetdevicelist      = 48
	OpLayoutcommit       = 49
	OpLayoutget          = 50
	OpLayoutreturn       = 51
	OpSecinfoNoName      = 52
	OpSequence           = 53
	OpSetSsv             = 54
	OpTestStateid        = 55
	OpWantDelegation     = 56
	OpDestroyClientid    = 57
	OpReclaimComplete    = 58
	OpIllegal            = 10044
)

var nfsOpnum4 = map[int]string{
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

func (nfs *NFS) eatData(op int, xdr *Xdr) {

	switch op {
	case OpGetattr:
		xdr.getUIntVector()
	case OpGetfh:
		// nothing to eat
	case OpLookup:
		xdr.getDynamicOpaque()
	case OpLookupp:
		// nothing to eat
	case OpNverify:
		xdr.getUIntVector()
		xdr.getDynamicOpaque()
	case OpPutfh:
		xdr.getDynamicOpaque()
	case OpPutpubfh:
		// nothing to eat
	case OpPutrootfh:
		// nothing to eat
	case OpReadlink:
		// nothing to eat
	case OpRenew:
		xdr.getUHyper()
	case OpRestorefh:
		// nothing to eat
	case OpSavefh:
		// nothing to eat
	case OpSecinfo:
		xdr.getDynamicOpaque()
	case OpVerify:
		xdr.getUIntVector()
		xdr.getDynamicOpaque()
	case OpSequence:
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
func (nfs *NFS) findV4MainOpcode(xdr *Xdr) string {

	// did we find a main operation opcode?
	found := false

	// default op code
	currentOpname := "ILLEGAL"

	opcount := int(xdr.getUInt())
	for i := 0; !found && i < opcount; i++ {
		op := int(xdr.getUInt())
		opname, ok := nfsOpnum4[op]

		if !ok {
			return fmt.Sprintf("ILLEGAL (%d)", op)
		}
		currentOpname = opname

		switch op {
		// First class ops
		//
		// The first class ops usually the main operation in the compound.
		// NFS spec allowes to build compound opertion where multiple
		// first class ops are used, like OPEN->LOCK->WRITE->LOCKU->CLOSE,
		// but such construnction are not used in the practice.
		case
			OpAccess,
			OpBackchannelCtl,
			OpBindConnToSession,
			OpClose,
			OpCommit,
			OpCreate,
			OpCreateSession,
			OpDelegpurge,
			OpDelegreturn,
			OpDestroyClientid,
			OpDestroySession,
			OpExchangeID,
			OpFreeStateid,
			OpGetdeviceinfo,
			OpGetdevicelist,
			OpGetDirDelegation,
			OpLayoutcommit,
			OpLayoutget,
			OpLayoutreturn,
			OpLink,
			OpLock,
			OpLockt,
			OpLocku,
			OpOpen,
			OpOpenattr,
			OpOpenConfirm,
			OpOpenDowngrade,
			OpRead,
			OpReaddir,
			OpReadlink,
			OpReclaimComplete,
			OpReleaseLockowner,
			OpRemove,
			OpRename,
			OpSecinfoNoName,
			OpSetattr,
			OpSetclientid,
			OpSetclientidConfirm,
			OpSetSsv,
			OpTestStateid,
			OpWantDelegation,
			OpWrite:

			found = true
		default:
			nfs.eatData(op, xdr)
		}
	}
	return currentOpname
}
