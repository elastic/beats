package mongodb

// Represent a mongodb message being parsed
import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type mongodbMessage struct {
	Ts time.Time

	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
	Direction    uint8

	IsResponse      bool
	ExpectsResponse bool

	// Standard message header fields from mongodb wire protocol
	// see http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#standard-message-header
	messageLength int
	requestId     int
	responseTo    int
	opCode        opCode

	// deduced from content. Either an operation from the original wire protocol or the name of a command (passed through a query)
	// List of commands: http://docs.mongodb.org/manual/reference/command/
	// List of original protocol operations: http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#request-opcodes
	method    string
	error     string
	resource  string
	documents []interface{}
	params    map[string]interface{}

	// Other fields vary very much depending on operation type
	// lets just put them in a map
	event common.MapStr
}

// Represent a stream being parsed that contains a mongodb message
type stream struct {
	tcptuple *common.TcpTuple

	data    []byte
	message *mongodbMessage
}

// Parser moves to next message in stream
func (st *stream) PrepareForNewMessage() {
	st.data = st.data[st.message.messageLength:]
	st.message = nil
}

// The private data of a parser instance
// is composed of 2 potentially active streams: incoming, outgoing
type mongodbConnectionData struct {
	Streams [2]*stream
}

// Represent a full mongodb transaction (request/reply)
// These transactions are the end product of this parser
type transaction struct {
	Type         string
	tuple        common.TcpTuple
	cmdline      *common.CmdlineTuple
	Src          common.Endpoint
	Dst          common.Endpoint
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	BytesOut     int
	BytesIn      int

	Mongodb common.MapStr

	event     common.MapStr
	method    string
	resource  string
	error     string
	params    map[string]interface{}
	documents []interface{}
}

type opCode int32

const (
	opReply      opCode = 1
	opMsg        opCode = 1000
	opUpdate     opCode = 2001
	opInsert     opCode = 2002
	opReserved   opCode = 2003
	opQuery      opCode = 2004
	opGetMore    opCode = 2005
	opDelete     opCode = 2006
	opKillCursor opCode = 2007
)

// List of valid mongodb wire protocol operation codes
// see http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/#request-opcodes
var opCodeNames = map[opCode]string{
	1:    "OP_REPLY",
	1000: "OP_MSG",
	2001: "OP_UPDATE",
	2002: "OP_INSERT",
	2003: "RESERVED",
	2004: "OP_QUERY",
	2005: "OP_GET_MORE",
	2006: "OP_DELETE",
	2007: "OP_KILL_CURSORS",
}

func validOpcode(o opCode) bool {
	_, found := opCodeNames[o]
	return found
}

func (o opCode) String() string {
	return opCodeNames[o]
}

func awaitsReply(c opCode) bool {
	return c == opQuery || c == opGetMore
}

// List of mongodb user commands (send throuwh a query of the legacy protocol)
// see http://docs.mongodb.org/manual/reference/command/
//
// This list was obtained by calling db.listCommands() and some grepping.
// They are compared cased insensitive
var DatabaseCommands = []string{
	"getLastError",
	"connPoolSync",
	"top",
	"dropIndexes",
	"explain",
	"grantRolesToRole",
	"dropRole",
	"dropAllRolesFromDatabase",
	"listCommands",
	"replSetReconfig",
	"replSetFresh",
	"writebacklisten",
	"setParameter",
	"update",
	"replSetGetStatus",
	"find",
	"resync",
	"appendOplogNote",
	"revokeRolesFromRole",
	"compact",
	"createUser",
	"replSetElect",
	"getPrevError",
	"serverStatus",
	"getShardVersion",
	"updateRole",
	"replSetFreeze",
	"getCmdLineOpts",
	"applyOps",
	"count",
	"aggregate",
	"copydbsaslstart",
	"distinct",
	"repairDatabase",
	"profile",
	"replSetStepDown",
	"findAndModify",
	"_transferMods",
	"filemd5",
	"forceerror",
	"getnonce",
	"saslContinue",
	"clone",
	"saslStart",
	"_getUserCacheGeneration",
	"_recvChunkCommit",
	"whatsmyuri",
	"repairCursor",
	"validate",
	"dbHash",
	"planCacheListFilters",
	"touch",
	"mergeChunks",
	"cursorInfo",
	"_recvChunkStart",
	"unsetSharding",
	"revokePrivilegesFromRole",
	"logout",
	"group",
	"shardConnPoolStats",
	"listDatabases",
	"buildInfo",
	"availableQueryOptions",
	"_isSelf",
	"splitVector",
	"geoSearch",
	"dbStats",
	"connectionStatus",
	"currentOpCtx",
	"copydb",
	"insert",
	"reIndex",
	"moveChunk",
	"cleanupOrphaned",
	"driverOIDTest",
	"isMaster",
	"getParameter",
	"replSetHeartbeat",
	"ping",
	"listIndexes",
	"dropUser",
	"dropDatabase",
	"dataSize",
	"convertToCapped",
	"planCacheSetFilter",
	"usersInfo",
	"grantPrivilegesToRole",
	"handshake",
	"_mergeAuthzCollections",
	"mapreduce.shardedfinish",
	"_recvChunkAbort",
	"authSchemaUpgrade",
	"replSetGetConfig",
	"replSetSyncFrom",
	"collStats",
	"replSetMaintenance",
	"createRole",
	"copydbgetnonce",
	"cloneCollectionAsCapped",
	"_migrateClone",
	"parallelCollectionScan",
	"connPoolStats",
	"revokeRolesFromUser",
	"authenticate",
	"create",
	"shutdown",
	"invalidateUserCache",
	"shardingState",
	"renameCollection",
	"replSetGetRBID",
	"splitChunk",
	"createIndexes",
	"updateUser",
	"cloneCollection",
	"logRotate",
	"planCacheListPlans",
	"medianKey",
	"hostInfo",
	"geoNear",
	"fsync",
	"checkShardingIndex",
	"getShardMap",
	"planCacheClear",
	"listCollections",
	"collMod",
	"_recvChunkStatus",
	"planCacheListQueryShapes",
	"delete",
	"planCacheClearFilters",
	"mapReduce",
	"rolesInfo",
	"eval",
	"drop",
	"grantRolesToUser",
	"resetError",
	"getLog",
	"dropAllUsersFromDatabase",
	"diagLogging",
	"replSetUpdatePosition",
	"setShardVersion",
	"replSetInitiate",
}
