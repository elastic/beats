package cassandra

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	. "github.com/elastic/beats/packetbeat/protos/cassandra/internal/gocql"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

// cassandra application level protocol analyzer plugin
type cassandra struct {
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
	debugf = logp.MakeDebug("cassandra")
)

func init() {
	protos.Register("cassandra", New)
}

// New create and initializes a new cassandra protocol analyzer instance.
func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &cassandra{}
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

func (cassandra *cassandra) init(results publish.Transactions, config *cassandraConfig) error {
	if err := cassandra.setFromConfig(config); err != nil {
		return err
	}
	cassandra.pub.results = results
	return nil
}

func (cassandra *cassandra) setFromConfig(config *cassandraConfig) error {

	// set module configuration
	if err := cassandra.ports.Set(config.Ports); err != nil {
		return err
	}

	// set parser configuration
	parser := &cassandra.parserConfig
	parser.maxBytes = tcp.TCP_MAX_DATA_IN_STREAM

	// set parser's compressor, only `snappy` supported right now
	if config.Compressor == Snappy {
		parser.compressor = SnappyCompressor{}
	} else {
		parser.compressor = nil
	}

	// parsed ignored ops
	if len(config.OPsIgnored) > 0 {
		maps := map[FrameOp]bool{}
		for _, op := range config.OPsIgnored {
			maps[op] = true
		}
		parser.ignoredOps = maps
		debugf("parsed config IgnoredOPs: %v ", parser.ignoredOps)
	}

	// set transaction correlator configuration
	trans := &cassandra.transConfig
	trans.transactionTimeout = config.TransactionTimeout

	// set transaction publisher configuration
	pub := &cassandra.pub
	pub.sendRequest = config.SendRequest
	pub.sendResponse = config.SendResponse
	pub.sendRequestHeader = config.SendRequestHeader
	pub.sendResponseHeader = config.SendResponseHeader

	return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func (cassandra *cassandra) ConnectionTimeout() time.Duration {
	return cassandra.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func (cassandra *cassandra) GetPorts() []int {
	return cassandra.ports.Ports
}

// Parse processes a TCP packet. Return nil if connection
// state shall be dropped (e.g. parser not in sync with tcp stream)
func (cassandra *cassandra) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("Parse cassandra exception")

	conn := cassandra.ensureConnection(private)
	st := conn.streams[dir]
	if st == nil {
		st = &stream{}
		st.parser.init(&cassandra.parserConfig, func(msg *message) error {
			return conn.trans.onMessage(tcptuple.IPPort(), dir, msg)
		})
		conn.streams[dir] = st
	}

	if err := st.parser.feed(pkt.Ts, pkt.Payload); err != nil {
		debugf("%v, dropping TCP stream for error in direction %v.", err, dir)
		cassandra.onDropConnection(conn)
		return nil
	}
	return conn
}

// ReceivedFin handles TCP-FIN packet.
func (cassandra *cassandra) ReceivedFin(
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	return private
}

// GapInStream handles lost packets in tcp-stream.
func (cassandra *cassandra) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int,
	private protos.ProtocolData,
) (protos.ProtocolData, bool) {
	conn := getConnection(private)
	if conn != nil {
		cassandra.onDropConnection(conn)
	}

	return nil, true
}

// onDropConnection processes and optionally sends incomplete
// transaction in case of connection being dropped due to error
func (cassandra *cassandra) onDropConnection(conn *connection) {
}

func (cassandra *cassandra) ensureConnection(private protos.ProtocolData) *connection {
	conn := getConnection(private)
	if conn == nil {
		conn = &connection{}
		conn.trans.init(&cassandra.transConfig, cassandra.pub.onTransaction)
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
		logp.Warn("cassandra connection type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: cassandra connection data not set")
		return nil
	}
	return priv
}
