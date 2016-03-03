package protos

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/publish"
	"github.com/urso/ucfg"
)

type ProtocolPlugin func(
	testMode bool,
	results publish.Transactions,
	cfg *ucfg.Config,
) (Plugin, error)

// Functions to be exported by a protocol plugin
type Plugin interface {
	// Called to return the configured ports
	GetPorts() []int
}

type TcpPlugin interface {
	Plugin

	// Called when TCP payload data is available for parsing.
	Parse(pkt *Packet, tcptuple *common.TcpTuple,
		dir uint8, private ProtocolData) ProtocolData

	// Called when the FIN flag is seen in the TCP stream.
	ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
		private ProtocolData) ProtocolData

	// Called when a packets are missing from the tcp
	// stream.
	GapInStream(tcptuple *common.TcpTuple, dir uint8, nbytes int,
		private ProtocolData) (priv ProtocolData, drop bool)

	// ConnectionTimeout returns the per stream connection timeout.
	// Return <=0 to set default tcp module transaction timeout.
	ConnectionTimeout() time.Duration
}

type UdpPlugin interface {
	Plugin

	// ParseUdp is invoked when UDP payload data is available for parsing.
	ParseUdp(pkt *Packet)
}

// Protocol identifier.
type Protocol uint16

// Protocol constants.
const (
	UnknownProtocol Protocol = iota
	AmqpProtocol
	HttpProtocol
	MysqlProtocol
	RedisProtocol
	PgsqlProtocol
	ThriftProtocol
	MongodbProtocol
	DnsProtocol
	MemcacheProtocol
)

// Protocol names
var ProtocolNames = []string{
	"unknown",
	"amqp",
	"http",
	"mysql",
	"redis",
	"pgsql",
	"thrift",
	"mongodb",
	"dns",
	"memcache",
}

func (p Protocol) String() string {
	if int(p) >= len(ProtocolNames) {
		return "impossible"
	}
	return ProtocolNames[p]
}

var (
	protocolPlugins = map[Protocol]ProtocolPlugin{}
)

func Register(proto Protocol, plugin ProtocolPlugin) {
	protocolPlugins[proto] = plugin
}
