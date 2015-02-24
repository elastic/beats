package protos

import (
	"packetbeat/common"
	"time"
)

// ProtocolData interface to represent an upper
// protocol private data. Used with types like
// HttpStream, MysqlStream, etc.
type ProtocolData interface{}

type Packet struct {
	Ts      time.Time
	Tuple   common.IpPortTuple
	Payload []byte
}

// Functions to be exported by a protocol plugin
type ProtocolPlugin interface {
	Init(test_mode bool, results chan common.MapStr) error
	SetFromConfig() error
	Parse(pkt *Packet, tcptuple *common.TcpTuple, dir uint8)
	ReceivedFin(tcptuple *common.TcpTuple, dir uint8)
	GapInStream(tcptuple *common.TcpTuple, dir uint8)
}

// Protocol identifier.
type Protocol uint16

// Protocol constants.
const (
	UnknownProtocol Protocol = iota
	HttpProtocol
	MysqlProtocol
	RedisProtocol
	PgsqlProtocol
	ThriftProtocol
)

// Protocol names
var ProtocolNames = []string{
	"unknown",
	"http",
	"mysql",
	"redis",
	"pgsql",
	"thrift",
}

func (p Protocol) String() string {
	if int(p) >= len(ProtocolNames) {
		return "impossible"
	}
	return ProtocolNames[p]
}

// list of protocol plugins
type Protocols struct {
	protos map[Protocol]*ProtocolPlugin
}

// Singleton of Protocols type.
var Protos Protocols

func (protocols Protocols) Get(proto Protocol) *ProtocolPlugin {
	ret, exists := protocols.protos[proto]
	if !exists {
		return nil
	}
	return ret
}

func (protos Protocols) Register(proto Protocol, plugin *ProtocolPlugin) {
	protos.protos[proto] = plugin
}

func Init() {
	Protos = Protocols{}
	Protos.protos = make(map[Protocol]*ProtocolPlugin)
}
