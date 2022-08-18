package tds

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// tdsPlugin application level protocol analyzer plugin
type tdsPlugin struct {
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

// Uni-directional tcp stream state for parsing messages.
type stream struct {
	parser parser
}

var (
	debugf = logp.MakeDebug("tds")

	// use isDebug/isDetailed to guard debugf/detailedf to minimize allocations
	// (garbage collection) when debug log is disabled.
	isDebug = true
)

func init() {
	protos.Register("tds", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	watcher procs.ProcessesWatcher,
	cfg *conf.C,
) (protos.Plugin, error) {
	p := &tdsPlugin{}
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

func (tp *tdsPlugin) init(results protos.Reporter, config *tdsConfig) error {
	logp.Info("tds.init(results, config)")
	if err := tp.setFromConfig(config); err != nil {
		return err
	}
	tp.pub.results = results

	isDebug = logp.IsDebug("http")
	return nil
}
func (tp *tdsPlugin) setFromConfig(config *tdsConfig) error {

	logp.Info("tds.setFromConfig(config)")

	// set module configuration
	if err := tp.ports.Set(config.Ports); err != nil {
		return err
	}

	// set parser configuration
	parser := &tp.parserConfig
	parser.maxBytes = tcp.TCPMaxDataInStream

	// set transaction correlator configuration
	trans := &tp.transConfig
	trans.transactionTimeout = config.TransactionTimeout

	// set transaction publisher configuration
	pub := &tp.pub
	pub.sendRequest = config.SendRequest
	pub.sendResponse = config.SendResponse

	return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func (tp *tdsPlugin) ConnectionTimeout() time.Duration {
	logp.Info("tds.ConnectionTimeout()")
	return tp.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func (tp *tdsPlugin) GetPorts() []int {
	logp.Info("tds.GetPorts()")
	logp.Info("- Ports: ~v", tp.ports.Ports)
	return tp.ports.Ports
}

// Parse processes a TCP packet. Return nil if connection
// state shall be dropped (e.g. parser not in sync with tcp stream)
func (tp *tdsPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	logp.Info("tds.Parse()")
	defer logp.Recover("Parse tdsPlugin exception")

	conn := tp.ensureConnection(private)
	st := conn.streams[dir]
	if st == nil {
		st = &stream{}
		st.parser.init(&tp.parserConfig, func(msg *message) error {
			return conn.trans.onMessage(tcptuple.IPPort(), dir, msg)
		})
		conn.streams[dir] = st
	}

	if err := st.parser.feed(pkt.Ts, pkt.Payload); err != nil {
		debugf("%v, dropping TCP stream for error in direction %v.", err, dir)
		tp.onDropConnection(conn)
		return nil
	}
	return conn
}

// ReceivedFin handles TCP-FIN packet.
func (tp *tdsPlugin) ReceivedFin(
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	logp.Info("tds.ReceivedFin()")
	return private
}

// GapInStream handles lost packets in tcp-stream.
func (tp *tdsPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int,
	private protos.ProtocolData,
) (protos.ProtocolData, bool) {
	logp.Info("tds.GapInStream()")
	conn := getConnection(private)
	if conn != nil {
		tp.onDropConnection(conn)
	}

	return nil, true
}

// onDropConnection processes and optionally sends incomplete
// transaction in case of connection being dropped due to error
func (tp *tdsPlugin) onDropConnection(conn *connection) {
	logp.Info("tds.onDropConnection()")
}

func (tp *tdsPlugin) ensureConnection(private protos.ProtocolData) *connection {
	logp.Info("tds.ensureConnection()")
	conn := getConnection(private)
	if conn == nil {
		conn = &connection{}
		conn.trans.init(&tp.transConfig, tp.pub.onTransaction)
	}
	return conn
}

func (conn *connection) dropStreams() {
	logp.Info("tds.dropStreams()")
	conn.streams[0] = nil
	conn.streams[1] = nil
}

func getConnection(private protos.ProtocolData) *connection {
	logp.Info("tds.getConnection()")
	if private == nil {
		return nil
	}

	priv, ok := private.(*connection)
	if !ok {
		logp.Warn("tds connection type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: tds connection data not set")
		return nil
	}
	return priv
}
