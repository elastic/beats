package {{ cookiecutter.module }}

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

// {{ cookiecutter.plugin_type }} application level protocol analyzer plugin
type {{ cookiecutter.plugin_type }} struct {
	ports        protos.PortsConfig
	parserConfig parserConfig
	transConfig  transactionConfig
	pub          transPub
}

// Application Layer tcp stream data to be stored on tcp connection context.
type connection struct {
	streams [2]*stream
	trans   transactions
}

// Uni-directioal tcp stream state for parsing messages.
type stream struct {
	parser parser
}

var (
	debugf = logp.MakeDebug("{{ cookiecutter.module }}")

	// use isDebug/isDetailed to guard debugf/detailedf to minimize allocations
	// (garbage collection) when debug log is disabled.
	isDebug = false
)

func init() {
	protos.Register("{{ cookiecutter.module }}", New)
}

// New create and initializes a new {{ cookiecutter.protocol }} protocol analyzer instance.
func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &{{ cookiecutter.plugin_type }}{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) init(results publish.Transactions, config *{{ cookiecutter.module }}Config) error {
	if err := {{ cookiecutter.plugin_var }}.setFromConfig(config); err != nil {
		return err
	}
	{{ cookiecutter.plugin_var }}.pub.results = results

	isDebug = logp.IsDebug("http")
	return nil
}

func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) setFromConfig(config *{{ cookiecutter.module }}Config) error {

	// set module configuration
	if err := {{ cookiecutter.plugin_var }}.ports.Set(config.Ports); err != nil {
		return err
	}

	// set parser configuration
	parser := &{{ cookiecutter.plugin_var }}.parserConfig
	parser.maxBytes = tcp.TCP_MAX_DATA_IN_STREAM

	// set transaction correlator configuration
	trans := &{{ cookiecutter.plugin_var }}.transConfig
	trans.transactionTimeout = config.TransactionTimeout

	// set transaction publisher configuration
	pub := &{{ cookiecutter.plugin_var }}.pub
	pub.sendRequest = config.SendRequest
	pub.sendResponse = config.SendResponse

	return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) ConnectionTimeout() time.Duration {
	return {{ cookiecutter.plugin_var }}.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) GetPorts() []int {
	return {{ cookiecutter.plugin_var }}.ports.Ports
}

// Parse processes a TCP packet. Return nil if connection
// state shall be dropped (e.g. parser not in sync with tcp stream)
func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) Parse(
	pkt *protos.Packet,
	tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("Parse {{ cookiecutter.plugin_type }} exception")

	conn := {{ cookiecutter.plugin_var }}.ensureConnection(private)
	st := conn.streams[dir]
	if st == nil {
		st = &stream{}
		st.parser.init(&{{ cookiecutter.plugin_var }}.parserConfig, func(msg *message) error {
			return conn.trans.onMessage(tcptuple.IpPort(), dir, msg)
		})
		conn.streams[dir] = st
	}

	if err := st.parser.feed(pkt.Ts, pkt.Payload); err != nil {
		debugf("%v, dropping TCP stream for error in direction %v.", err, dir)
		{{ cookiecutter.plugin_var }}.onDropConnection(conn)
		return nil
	}
	return conn
}

// ReceivedFin handles TCP-FIN packet.
func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) ReceivedFin(
	tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	return private
}

// GapInStream handles lost packets in tcp-stream.
func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int,
	private protos.ProtocolData,
) (protos.ProtocolData, bool) {
	conn := getConnection(private)
	if conn != nil {
		{{ cookiecutter.plugin_var }}.onDropConnection(conn)
	}

	return nil, true
}

// onDropConnection processes and optionally sends incomplete
// transaction in case of connection being dropped due to error
func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) onDropConnection(conn *connection) {
}

func ({{ cookiecutter.plugin_var }} *{{ cookiecutter.plugin_type }}) ensureConnection(private protos.ProtocolData) *connection {
	conn := getConnection(private)
	if conn == nil {
		conn = &connection{}
		conn.trans.init(&{{ cookiecutter.plugin_var }}.transConfig, {{ cookiecutter.plugin_var }}.pub.onTransaction)
	}
	return conn
}

func (conn *connection) dropStreams() {
	conn.streams[0] = nil
	conn.streams[1] = nil
}

func getConnection(private protos.ProtocolData) *connection {
	if private == nil {
		return nil
	}

	priv, ok := private.(*connection)
	if !ok {
		logp.Warn("{{ cookiecutter.module }} connection type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: {{ cookiecutter.module }} connection data not set")
		return nil
	}
	return priv
}
