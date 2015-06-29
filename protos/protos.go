package protos

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
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
	// Called to initialize the Plugin
	Init(test_mode bool, results chan common.MapStr) error

	// Called to return the configured ports
	GetPorts() []int

	// Called when payload data is available for parsing.
	Parse(pkt *Packet, tcptuple *common.TcpTuple,
		dir uint8, private ProtocolData) ProtocolData

	// Called when the FIN flag is seen in the TCP stream.
	ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
		private ProtocolData) ProtocolData

	// Called when a packets are missing from the tcp
	// stream.
	GapInStream(tcptuple *common.TcpTuple, dir uint8, nbytes int,
		private ProtocolData) (priv ProtocolData, drop bool)
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
	protos map[Protocol]ProtocolPlugin
}

// Singleton of Protocols type.
var Protos Protocols

func (protocols Protocols) Get(proto Protocol) ProtocolPlugin {
	ret, exists := protocols.protos[proto]
	if !exists {
		return nil
	}
	return ret
}

func (protocols Protocols) GetAll() map[Protocol]ProtocolPlugin {
	return protocols.protos
}

func (protos Protocols) Register(proto Protocol, plugin ProtocolPlugin) {
	protos.protos[proto] = plugin
}

func init() {
	logp.Debug("protos", "Initializing Protos")
	Protos = Protocols{}
	Protos.protos = make(map[Protocol]ProtocolPlugin)
}
