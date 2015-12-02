package memcache

// Types including Stringer interface implementation and constants for common
// codes.

// Memcache Command Classification
type commandTypeCode uint8

const (
	MemcacheUnknownType commandTypeCode = iota
	MemcacheLoadMsg
	MemcacheStoreMsg
	MemcacheDeleteMsg
	MemcacheCounterMsg
	MemcacheInfoMsg
	MemcacheSlabCtrlMsg
	MemcacheLruCrawlerMsg
	MemcacheStatsMsg
	MemcacheSuccessResp
	MemcacheFailResp
	MemcacheAuthMsg
)

var commandTypeCodeStrings = []string{
	"UNKNOWN",
	"Load",
	"Store",
	"Delete",
	"Counter",
	"Info",
	"SlabCtrl",
	"LRUCrawler",
	"Stats",
	"Success",
	"Fail",
	"Auth",
}

// commandCode defines shared text and binary protocol message type codes.
type commandCode uint8

const (
	MemcacheCmdUNKNOWN commandCode = iota
	MemcacheCmdNoOp
	MemcacheCmdSet
	MemcacheCmdAdd
	MemcacheCmdReplace
	MemcacheCmdAppend
	MemcacheCmdPrepend
	MemcacheCmdCas

	MemcacheCmdGet
	MemcacheCmdGets

	MemcacheCmdIncr
	MemcacheCmdDecr
	MemcacheResCounterOp

	MemcacheCmdTouch

	MemcacheCmdDelete

	MemcacheCmdSlabs
	MemcacheCmdSlabsReassign
	MemcacheCmdSlabsAutomove

	MemcacheCmdLru
	MemcacheCmdLruEnable
	MemcacheCmdLruDisable
	MemcacheCmdLruSleep
	MemcacheCmdLruToCrawl
	MemcacheCmdLruCrawl

	MemcacheCmdStats
	MemcacheResStat

	MemcacheCmdFlushAll
	MemcacheCmdVerbosity
	MemcacheCmdQuit
	MemcacheCmdVersion

	MemcacheCmdSaslList
	MemcacheCmdSaslAuth
	MemcacheCmdSaslStep

	MemcacheResOK
	MemcacheResValue
	MemcacheResEnd

	MemcacheResStored
	MemcacheResNotStored
	MemcacheResExists
	MemcacheResNotFound

	MemcacheResTouched

	MemcacheResDeleted

	MemcacheErrError
	MemcacheErrClientError
	MemcacheErrServerError
	MemcacheErrBusy
	MemcacheErrBadClass
	MemcacheErrNoSpare
	MemcacheErrNotFull
	MemcacheErrUnsafe
	MemcacheErrSame

	MemcacheResVersion
)

var commandCodeStrings = []string{
	"UNKNOWN",
	"noop",
	"set",
	"add",
	"replace",
	"append",
	"prepend",
	"cas",

	"get",
	"gets",

	"incr",
	"decr",
	"<counter_op_res>",

	"touch",

	"delete",

	"slabs",
	"slabs reassign",
	"slabs automove",

	"lru_crawler",
	"lru_crawler enable",
	"lru_crawler disable",
	"lru_crawler sleep",
	"lru_crawler tocrawl",
	"lru_crawler crawl",

	"stats",
	"STAT",

	"flush_all",
	"verbosity",
	"quit",
	"version",
	"sasl_list",
	"sasl_auth",
	"sasl_step",

	"OK",
	"VALUE",
	"END",

	"STORED",
	"NOT_STORED",
	"EXISTS",
	"NOT_FOUND",

	"TOCHED",

	"DELETED",

	"ERROR",
	"CLIENT_ERROR",
	"SERVER_ERROR",
	"BUSY",
	"BADCLASS",
	"NOSPARE",
	"NOTFULL",
	"UNSAFE",
	"SAME",

	"VERSION",
}

// memcacheOpcode stores memcache binary protocol opcodes.
type memcacheOpcode uint8

const (
	opcodeGet           memcacheOpcode = 0x00
	opcodeSet           memcacheOpcode = 0x01
	opcodeAdd           memcacheOpcode = 0x02
	opcodeReplace       memcacheOpcode = 0x03
	opcodeDelete        memcacheOpcode = 0x04
	opcodeIncrement     memcacheOpcode = 0x05
	opcodeDecrement     memcacheOpcode = 0x06
	opcodeQuit          memcacheOpcode = 0x07
	opcodeFlush         memcacheOpcode = 0x08
	opcodeGetQ          memcacheOpcode = 0x09
	opcodeNoOp          memcacheOpcode = 0x0a
	opcodeVersion       memcacheOpcode = 0x0b
	opcodeGetK          memcacheOpcode = 0x0c
	opcodeGetKQ         memcacheOpcode = 0x0d
	opcodeAppend        memcacheOpcode = 0x0e
	opcodePrepend       memcacheOpcode = 0x0f
	opcodeStat          memcacheOpcode = 0x10
	opcodeSetQ          memcacheOpcode = 0x11
	opcodeAddQ          memcacheOpcode = 0x12
	opcodeReplaceQ      memcacheOpcode = 0x13
	opcodeDeleteQ       memcacheOpcode = 0x14
	opcodeIncrementQ    memcacheOpcode = 0x15
	opcodeDecrementQ    memcacheOpcode = 0x16
	opcodeQuitQ         memcacheOpcode = 0x17
	opcodeFlushQ        memcacheOpcode = 0x18
	opcodeAppendQ       memcacheOpcode = 0x19
	opcodePrependQ      memcacheOpcode = 0x1a
	opcodeVerbosity     memcacheOpcode = 0x1b // not in memcached source
	opcodeTouch         memcacheOpcode = 0x1c
	opcodeGat           memcacheOpcode = 0x1d
	opcodeGatQ          memcacheOpcode = 0x1e
	opcodeSaslListMechs memcacheOpcode = 0x20
	opcodeSaslAuth      memcacheOpcode = 0x21
	opcodeSaslStep      memcacheOpcode = 0x22
	opcodeGatK          memcacheOpcode = 0x23
	opcodeGatKQ         memcacheOpcode = 0x24
	opcodeRGet          memcacheOpcode = 0x30 // range op not supported by memcached?
	opcodeRSet          memcacheOpcode = 0x31 // range op not supported by memcached?
	opcodeRSetQ         memcacheOpcode = 0x32 // range op not supported by memcached?
	opcodeRAppend       memcacheOpcode = 0x33 // range op not supported by memcached?
	opcodeRAppendQ      memcacheOpcode = 0x34 // range op not supported by memcached?
	opcodeRPrepend      memcacheOpcode = 0x35 // range op not supported by memcached?
	opcodeRPrependQ     memcacheOpcode = 0x36 // range op not supported by memcached?
	opcodeRDelete       memcacheOpcode = 0x37 // range op not supported by memcached?
	opcodeRDeleteQ      memcacheOpcode = 0x38 // range op not supported by memcached?
	opcodeRIncr         memcacheOpcode = 0x39 // range op not supported by memcached
	opcodeRIncrQ        memcacheOpcode = 0x3a // range op not supported by memcached?
	opcodeRDecr         memcacheOpcode = 0x3b // range op not supported by memcached?
	opcodeRDecrQ        memcacheOpcode = 0x3c // range op not supported by memcached?

	/* These codes have been found on wiki only:
	 *  https://code.google.com/p/memcached/wiki/BinaryProtocolRevamped
	 *
	 *  But no reference in source code
	 */

	opcodeSetVBucket         memcacheOpcode = 0x3d
	opcodeGetVBucket         memcacheOpcode = 0x3e
	opcodeDelVBucket         memcacheOpcode = 0x3f
	opcodeTapConnect         memcacheOpcode = 0x40
	opcodeTapMutation        memcacheOpcode = 0x41
	opcodeTapDelete          memcacheOpcode = 0x42
	opcodeTapFlush           memcacheOpcode = 0x43
	opcodeTapOpaque          memcacheOpcode = 0x44
	opcodeTapVBucketSet      memcacheOpcode = 0x45
	opcodeTapCheckpointStart memcacheOpcode = 0x46
	opcodeTapCheckpointEnd   memcacheOpcode = 0x47
)

var opcodeNames = map[memcacheOpcode]string{
	opcodeGet:                "Get",
	opcodeSet:                "Set",
	opcodeAdd:                "Add",
	opcodeReplace:            "Replace",
	opcodeDelete:             "Delete",
	opcodeIncrement:          "Increment",
	opcodeDecrement:          "Decrement",
	opcodeQuit:               "Quit",
	opcodeFlush:              "Flush",
	opcodeGetQ:               "GetQ",
	opcodeNoOp:               "No-op",
	opcodeVersion:            "Version",
	opcodeGetK:               "GetK",
	opcodeGetKQ:              "GetKQ",
	opcodeAppend:             "Append",
	opcodePrepend:            "Prepend",
	opcodeStat:               "Stat",
	opcodeSetQ:               "SetQ",
	opcodeAddQ:               "AddQ",
	opcodeReplaceQ:           "ReplaceQ",
	opcodeDeleteQ:            "DeleteQ",
	opcodeIncrementQ:         "IncrementQ",
	opcodeDecrementQ:         "DecrementQ",
	opcodeQuitQ:              "QuitQ",
	opcodeFlushQ:             "FlushQ",
	opcodeAppendQ:            "AppendQ",
	opcodePrependQ:           "PrependQ",
	opcodeVerbosity:          "Verbosity",
	opcodeTouch:              "Touch",
	opcodeGat:                "GAT",
	opcodeGatQ:               "GATQ",
	opcodeSaslListMechs:      "SASL list mechs",
	opcodeSaslAuth:           "SASL Auth",
	opcodeSaslStep:           "SASL Step",
	opcodeGatK:               "GatK",
	opcodeGatKQ:              "GatKQ",
	opcodeRGet:               "RGet",
	opcodeRSet:               "RSet",
	opcodeRSetQ:              "RSetQ",
	opcodeRAppend:            "RAppend",
	opcodeRAppendQ:           "RAppendQ",
	opcodeRPrepend:           "RPrepend",
	opcodeRPrependQ:          "RPrependQ",
	opcodeRDelete:            "RDelete",
	opcodeRDeleteQ:           "RDeleteQ",
	opcodeRIncr:              "RIncr",
	opcodeRIncrQ:             "RIncrQ",
	opcodeRDecr:              "RDecr",
	opcodeRDecrQ:             "RDecrQ",
	opcodeSetVBucket:         "Set VBucket",
	opcodeGetVBucket:         "Get VBucket",
	opcodeDelVBucket:         "Del VBucket",
	opcodeTapConnect:         "TAP Connect",
	opcodeTapMutation:        "TAP Mutation",
	opcodeTapDelete:          "TAP Delete",
	opcodeTapFlush:           "TAP Flush",
	opcodeTapOpaque:          "TAP Opaque",
	opcodeTapVBucketSet:      "TAP VBucket Set",
	opcodeTapCheckpointStart: "TAP Checkpoint Start",
	opcodeTapCheckpointEnd:   "TAP Checkpoint End",
}

type memcacheStatusCode uint16

const (
	statusCodeNoError                       memcacheStatusCode = 0x00
	statusCodeKeyNotFound                   memcacheStatusCode = 0x01
	statusCodeKeyExists                     memcacheStatusCode = 0x02
	statusCodeValueTooLarge                 memcacheStatusCode = 0x03
	statusCodeInvalidArguments              memcacheStatusCode = 0x04
	statusCodeItemNotStored                 memcacheStatusCode = 0x05
	statusCodeIncrDecrOnNonNumericValue     memcacheStatusCode = 0x06
	statusCodeVbucketBelongsToAnotherServer memcacheStatusCode = 0x07
	statusCodeAuthenticationError           memcacheStatusCode = 0x20 // doc says 0x08, but memcached headers say 0x20
	statusCodeAuthenticationContinue        memcacheStatusCode = 0x21 // doc says 0x09, but memcached headers say 0x21

	// error codes
	statusCodeUnknownCommand   memcacheStatusCode = 0x81
	statusCodeOutOfMemory      memcacheStatusCode = 0x82
	statusCodeNotSupported     memcacheStatusCode = 0x83
	statusCodeInternalError    memcacheStatusCode = 0x84
	statusCodeBusy             memcacheStatusCode = 0x85
	statusCodeTemporaryFailure memcacheStatusCode = 0x86
)

var statusCodeNames = map[memcacheStatusCode]string{
	statusCodeNoError:                       "Success",
	statusCodeKeyNotFound:                   "Key not found",
	statusCodeKeyExists:                     "Key exists",
	statusCodeValueTooLarge:                 "Value too large",
	statusCodeInvalidArguments:              "Invalid arguments",
	statusCodeItemNotStored:                 "Item not stored",
	statusCodeIncrDecrOnNonNumericValue:     "Incr/Decr on non-numeric value.",
	statusCodeVbucketBelongsToAnotherServer: "The vbucket belongs to another server",
	statusCodeAuthenticationError:           "Authentication error",
	statusCodeAuthenticationContinue:        "Authentication continue",
	statusCodeUnknownCommand:                "Unknown command",
	statusCodeOutOfMemory:                   "Out of memory",
	statusCodeNotSupported:                  "Not supported",
	statusCodeInternalError:                 "Internal error",
	statusCodeBusy:                          "Busy",
	statusCodeTemporaryFailure:              "Temporary failure",
}

func (c commandTypeCode) String() string {
	return commandTypeCodeStrings[c]
}

func (c commandCode) String() string {
	return commandCodeStrings[c]
}

func (c memcacheOpcode) String() string {
	return opcodeNames[c]
}

func (c memcacheStatusCode) String() string {
	return statusCodeNames[c]
}
