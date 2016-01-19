package protos

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/publish"
)

const (
	DefaultTransactionHashSize                 = 2 ^ 16
	DefaultTransactionExpiration time.Duration = 10 * time.Second
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

var ErrInvalidPort = errors.New("port number out of range")

// Protocol Plugin Port configuration with validation on init
type PortsConfig struct {
	Ports []int
}

func (p *PortsConfig) Init(ports ...int) error {
	return p.Set(ports)
}

func (p *PortsConfig) Set(ports []int) error {
	if err := validatePorts(ports); err != nil {
		return err
	}
	p.Ports = ports
	return nil
}

func validatePorts(ports []int) error {
	for port := range ports {
		if port < 0 || port > 65535 {
			return ErrInvalidPort
		}
	}
	return nil
}

// Functions to be exported by a protocol plugin
type ProtocolPlugin interface {
	// Called to initialize the Plugin
	Init(test_mode bool, results publish.Transactions) error

	// Called to return the configured ports
	GetPorts() []int
}

type TcpProtocolPlugin interface {
	ProtocolPlugin

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

type UdpProtocolPlugin interface {
	ProtocolPlugin

	// ParseUdp is invoked when UDP payload data is available for parsing.
	ParseUdp(pkt *Packet)
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
	MongodbProtocol
	DnsProtocol
	MemcacheProtocol
)

// Protocol names
var ProtocolNames = []string{
	"unknown",
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

type Protocols interface {
	BpfFilter(with_vlans bool, with_icmp bool) string
	GetTcp(proto Protocol) TcpProtocolPlugin
	GetUdp(proto Protocol) UdpProtocolPlugin
	GetAll() map[Protocol]ProtocolPlugin
	GetAllTcp() map[Protocol]TcpProtocolPlugin
	GetAllUdp() map[Protocol]UdpProtocolPlugin
	Register(proto Protocol, plugin ProtocolPlugin)
}

// list of protocol plugins
type ProtocolsStruct struct {
	all map[Protocol]ProtocolPlugin
	tcp map[Protocol]TcpProtocolPlugin
	udp map[Protocol]UdpProtocolPlugin
}

// Singleton of Protocols type.
var Protos ProtocolsStruct

func (protocols ProtocolsStruct) GetTcp(proto Protocol) TcpProtocolPlugin {
	plugin, exists := protocols.tcp[proto]
	if !exists {
		return nil
	}

	return plugin
}

func (protocols ProtocolsStruct) GetUdp(proto Protocol) UdpProtocolPlugin {
	plugin, exists := protocols.udp[proto]
	if !exists {
		return nil
	}

	return plugin
}

func (protocols ProtocolsStruct) GetAll() map[Protocol]ProtocolPlugin {
	return protocols.all
}

func (protocols ProtocolsStruct) GetAllTcp() map[Protocol]TcpProtocolPlugin {
	return protocols.tcp
}

func (protocols ProtocolsStruct) GetAllUdp() map[Protocol]UdpProtocolPlugin {
	return protocols.udp
}

// BpfFilter returns a Berkeley Packer Filter (BFP) expression that
// will match against packets for the registered protocols. If with_vlans is
// true the filter will match against both IEEE 802.1Q VLAN encapsulated
// and unencapsulated packets
func (protocols ProtocolsStruct) BpfFilter(with_vlans bool, with_icmp bool) string {
	// Sort the protocol IDs so that the return value is consistent.
	var protos []int
	for proto := range protocols.all {
		protos = append(protos, int(proto))
	}
	sort.Ints(protos)

	var expressions []string
	for _, key := range protos {
		proto := Protocol(key)
		plugin := protocols.all[proto]
		for _, port := range plugin.GetPorts() {
			has_tcp := false
			has_udp := false

			if _, present := protocols.tcp[proto]; present {
				has_tcp = true
			}
			if _, present := protocols.udp[proto]; present {
				has_udp = true
			}

			var expr string
			if has_tcp && !has_udp {
				expr = "tcp port %d"
			} else if !has_tcp && has_udp {
				expr = "udp port %d"
			} else {
				expr = "port %d"
			}

			expressions = append(expressions, fmt.Sprintf(expr, port))
		}
	}

	filter := strings.Join(expressions, " or ")
	if with_icmp {
		filter = fmt.Sprintf("%s or icmp or icmp6", filter)
	}
	if with_vlans {
		filter = fmt.Sprintf("%s or (vlan and (%s))", filter, filter)
	}
	return filter
}

func (protos ProtocolsStruct) Register(proto Protocol, plugin ProtocolPlugin) {
	protos.all[proto] = plugin
	if tcp, ok := plugin.(TcpProtocolPlugin); ok {
		protos.tcp[proto] = tcp
	}
	if udp, ok := plugin.(UdpProtocolPlugin); ok {
		protos.udp[proto] = udp
	}
}

func init() {
	logp.Debug("protos", "Initializing Protos")
	Protos = ProtocolsStruct{}
	Protos.all = make(map[Protocol]ProtocolPlugin)
	Protos.tcp = make(map[Protocol]TcpProtocolPlugin)
	Protos.udp = make(map[Protocol]UdpProtocolPlugin)
}
