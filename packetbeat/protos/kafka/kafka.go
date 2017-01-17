package kafka

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

// kafkaPlugin application level protocol analyzer plugin
type kafkaPlugin struct {
	ports          protos.PortsConfig
	splitterConfig splitterConfig
	transConfig    transactionConfig

	parser *parser
	pub    transPub
}

// Application Layer tcp stream data to be stored on tcp connection context.
type connection struct {
	streams [2]*stream
	trans   transactions
}

// Uni-directioal tcp stream state for parsing messages.
type stream struct {
	splitter splitter
}

var (
	debugf = logp.MakeDebug("kafka")

	// use isDebug/isDetailed to guard debugf/detailedf to minimize allocations
	// (garbage collection) when debug log is disabled.
	isDebug = false
)

func init() {
	protos.Register("kafka", New)
}

// New create and initializes a new kafka protocol analyzer instance.
func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	fmt.Println("initialize kafka module")

	p := &kafkaPlugin{}
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

func (kp *kafkaPlugin) init(results publish.Transactions, config *kafkaConfig) error {
	if err := kp.setFromConfig(config); err != nil {
		return err
	}
	kp.pub.results = results

	isDebug = logp.IsDebug("kafka")
	return nil
}

func (kp *kafkaPlugin) setFromConfig(config *kafkaConfig) error {

	// set module configuration
	if err := kp.ports.Set(config.Ports); err != nil {
		return err
	}

	// set splitter configuration
	splitter := &kp.splitterConfig
	splitter.maxBytes = tcp.TCPMaxDataInStream

	// set transaction correlator configuration
	trans := &kp.transConfig
	trans.transactionTimeout = config.TransactionTimeout

	// set parser configuration
	kp.parser = newParser(kp.pub.onTransaction, &parserConfig{
		ignoreAPI: config.Ignore,
		detailed:  config.SendDetails,
	})

	// set transaction publisher configuration

	return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func (kp *kafkaPlugin) ConnectionTimeout() time.Duration {
	return kp.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func (kp *kafkaPlugin) GetPorts() []int {
	return kp.ports.Ports
}

// Parse processes a TCP packet. Return nil if connection
// state shall be dropped (e.g. parser not in sync with tcp stream)
func (kp *kafkaPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("Parse kafkaPlugin exception")

	debugf("parse kafka packet")

	conn := kp.ensureConnection(private)
	st := conn.streams[dir]
	if st == nil {
		st = &stream{}
		st.splitter.init(&kp.splitterConfig, func(msg *rawMessage) error {
			msg.endpoint.IP = tcptuple.SrcIP
			msg.endpoint.Port = tcptuple.SrcPort
			return conn.trans.onMessage(dir, msg)
		})
		conn.streams[dir] = st
	}

	if err := st.splitter.feed(pkt.Ts, pkt.Payload); err != nil {
		debugf("%v, dropping TCP stream for error in direction %v.", err, dir)
		kp.onDropConnection(conn)
		return conn
	}
	return conn
}

// ReceivedFin handles TCP-FIN packet.
func (kp *kafkaPlugin) ReceivedFin(
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	return private
}

// GapInStream handles lost packets in tcp-stream.
func (kp *kafkaPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int,
	private protos.ProtocolData,
) (protos.ProtocolData, bool) {
	conn := getConnection(private)
	if conn != nil {
		kp.onDropConnection(conn)
	}

	// internally dropped state, but don't drop TCP connection so
	// request/response order from last syncing can be re-used
	return conn, false
}

// onDropConnection processes and optionally sends incomplete
// transaction in case of connection being dropped due to error
func (kp *kafkaPlugin) onDropConnection(conn *connection) {
	conn.dropStreams()
}

func (kp *kafkaPlugin) ensureConnection(private protos.ProtocolData) *connection {
	conn := getConnection(private)
	if conn == nil {
		conn = &connection{}
		conn.trans.init(&kp.transConfig, kp.parser.onTransaction)
	}
	return conn
}

func (conn *connection) dropStreams() {
	conn.streams[0] = nil
	conn.streams[1] = nil
	conn.trans.reset()
}

func getConnection(private protos.ProtocolData) *connection {
	if private == nil {
		return nil
	}

	priv, ok := private.(*connection)
	if !ok {
		logp.Warn("kafka connection type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: kafka connection data not set")
		return nil
	}
	return priv
}
