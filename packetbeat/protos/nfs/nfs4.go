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

const (
	opAccess             = 3
	opClose              = 4
	opCommit             = 5
	opCreate             = 6
	opDelegpurge         = 7
	opDelegreturn        = 8
	opGetattr            = 9
	opGetfh              = 10
	opLink               = 11
	opLock               = 12
	opLockt              = 13
	opLocku              = 14
	opLookup             = 15
	opLookupp            = 16
	opNverify            = 17
	opOpen               = 18
	opOpenattr           = 19
	opOpenConfirm        = 20
	opOpenDowngrade      = 21
	opPutfh              = 22
	opPutpubfh           = 23
	opPutrootfh          = 24
	opRead               = 25
	opReaddir            = 26
	opReadlink           = 27
	opRemove             = 28
	opRename             = 29
	opRenew              = 30
	opRestorefh          = 31
	opSavefh             = 32
	opSecinfo            = 33
	opSetattr            = 34
	opSetclientid        = 35
	opSetclientidConfirm = 36
	opVerify             = 37
	opWrite              = 38
	opReleaseLockowner   = 39
	opBackchannelCtl     = 40
	opBindConnToSession  = 41
	opExchangeID         = 42
	opCreateSession      = 43
	opDestroySession     = 44
	opFreeStateid        = 45
	opGetDirDelegation   = 46
	opGetdeviceinfo      = 47
	opGetdevicelist      = 48
	opLayoutcommit       = 49
	opLayoutget          = 50
	opLayoutreturn       = 51
	opSecinfoNoName      = 52
	opSequence           = 53
	opSetSsv             = 54
	opTestStateid        = 55
	opWantDelegation     = 56
	opDestroyClientid    = 57
	opReclaimComplete    = 58
	opAllocate           = 59
	opCopy               = 60
	opCopyNotify         = 61
	opDeallocate         = 62
	opIoAdvise           = 63
	opLayoutError        = 64
	opLayoutStats        = 65
	opOffloadCancel      = 66
	opOffloadStatus      = 67
	opReadPlus           = 68
	opSeek               = 69
	opWriteSame          = 70
	opClone              = 71
	opIllegal            = 10044
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
	59:    "ALLOCATE",
	60:    "COPY",
	61:    "COPY_NOTIFY",
	62:    "DEALLOCATE",
	63:    "IO_ADVISE",
	64:    "LAYOUTERROR",
	65:    "LAYOUTSTATS",
	66:    "OFFLOAD_CANCEL",
	67:    "OFFLOAD_STATUS",
	68:    "READ_PLUS",
	69:    "SEEK",
	70:    "WRITE_SAME",
	71:    "CLONE",
	10044: "ILLEGAL",
}

func (nfs *nfs) eatData(op int, xdr *xdr) error {
	switch op {
	case opGetattr:
		_, err := xdr.getUIntVector()
		return err
	case opGetfh:
		return nil
	case opLookup:
		_, err := xdr.getDynamicOpaque()
		return err
	case opLookupp:
		return nil
	case opNverify:
		if _, err := xdr.getUIntVector(); err != nil {
			return err
		}
		_, err := xdr.getDynamicOpaque()
		return err
	case opPutfh:
		_, err := xdr.getDynamicOpaque()
		return err
	case opPutpubfh:
		return nil
	case opPutrootfh:
		return nil
	case opReadlink:
		return nil
	case opRenew:
		_, err := xdr.getUHyper()
		return err
	case opRestorefh:
		return nil
	case opSavefh:
		return nil
	case opSecinfo:
		_, err := xdr.getDynamicOpaque()
		return err
	case opVerify:
		if _, err := xdr.getUIntVector(); err != nil {
			return err
		}
		_, err := xdr.getDynamicOpaque()
		return err
	case opSequence:
		if _, err := xdr.getOpaque(16); err != nil {
			return err
		}
		for i := 0; i < 4; i++ {
			if _, err := xdr.getUInt(); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
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
func (nfs *nfs) findV4MainOpcode(xdr *xdr) (string, error) {
	// did we find a main operation opcode?
	found := false

	// default op code
	currentOpname := "ILLEGAL"

	opcountVal, err := xdr.getUInt()
	if err != nil {
		return "", err
	}
	opcount := int(opcountVal)
	for i := 0; !found && i < opcount; i++ {
		opVal, err := xdr.getUInt()
		if err != nil {
			return "", err
		}
		op := int(opVal)
		opname, ok := nfsOpnum4[op]

		if !ok {
			return "ILLEGAL", nil
		}
		currentOpname = opname

		switch op {
		// First class ops
		//
		// The first class ops usually the main operation in the compound.
		// NFS spec allows to build compound operation where multiple
		// first class ops are used, like OPEN->LOCK->WRITE->LOCKU->CLOSE,
		// but such construction are not used in the practice.
		case
			opAccess,
			opBackchannelCtl,
			opBindConnToSession,
			opClose,
			opCommit,
			opCreate,
			opCreateSession,
			opDelegpurge,
			opDelegreturn,
			opDestroyClientid,
			opDestroySession,
			opExchangeID,
			opFreeStateid,
			opGetdeviceinfo,
			opGetdevicelist,
			opGetDirDelegation,
			opLayoutcommit,
			opLayoutget,
			opLayoutreturn,
			opLink,
			opLock,
			opLockt,
			opLocku,
			opOpen,
			opOpenattr,
			opOpenConfirm,
			opOpenDowngrade,
			opRead,
			opReaddir,
			opReadlink,
			opReclaimComplete,
			opReleaseLockowner,
			opRemove,
			opRename,
			opSecinfoNoName,
			opSetattr,
			opSetclientid,
			opSetclientidConfirm,
			opSetSsv,
			opTestStateid,
			opWantDelegation,
			opWrite,
			opAllocate,
			opCopy,
			opCopyNotify,
			opDeallocate,
			opIoAdvise,
			opLayoutError,
			opLayoutStats,
			opOffloadCancel,
			opOffloadStatus,
			opReadPlus,
			opSeek,
			opWriteSame,
			opClone:

			found = true
		default:
			if err := nfs.eatData(op, xdr); err != nil {
				return "", err
			}
		}
	}
	return currentOpname, nil
}
