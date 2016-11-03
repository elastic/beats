package protos

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/publish"
)

type ProtocolPlugin func(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (Plugin, error)

// Functions to be exported by a protocol plugin
type Plugin interface {
	// Called to return the configured ports
	GetPorts() []int
}

type TCPPlugin interface {
	Plugin

	// Called when TCP payload data is available for parsing.
	Parse(pkt *Packet, tcptuple *common.TCPTuple,
		dir uint8, private ProtocolData) ProtocolData

	// Called when the FIN flag is seen in the TCP stream.
	ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
		private ProtocolData) ProtocolData

	// Called when a packets are missing from the tcp
	// stream.
	GapInStream(tcptuple *common.TCPTuple, dir uint8, nbytes int,
		private ProtocolData) (priv ProtocolData, drop bool)

	// ConnectionTimeout returns the per stream connection timeout.
	// Return <=0 to set default tcp module transaction timeout.
	ConnectionTimeout() time.Duration
}

type UDPPlugin interface {
	Plugin

	// ParseUDP is invoked when UDP payload data is available for parsing.
	ParseUDP(pkt *Packet)
}

// Protocol identifier.
type Protocol uint16

// Protocol constants.
const (
	UnknownProtocol Protocol = iota
)

// Protocol names
var protocolNames = []string{
	"unknown",
}

func (p Protocol) String() string {
	if int(p) >= len(protocolNames) {
		return "impossible"
	}
	return protocolNames[p]
}

var (
	protocolPlugins = map[Protocol]ProtocolPlugin{}
	protocolSyms    = map[string]Protocol{}
)

func Lookup(name string) Protocol {
	if p, exists := protocolSyms[name]; exists {
		return p
	}
	return UnknownProtocol
}

func Register(name string, plugin ProtocolPlugin) {
	proto := Protocol(len(protocolNames))
	if p, exists := protocolSyms[name]; exists {
		// keep symbol table entries if plugin gets overwritten
		proto = p
	} else {
		protocolNames = append(protocolNames, name)
		protocolSyms[name] = proto
	}

	protocolPlugins[proto] = plugin
}
