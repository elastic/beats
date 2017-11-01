package smtp

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

// smtpPlugin application level protocol analyzer plugin
type smtpPlugin struct {
	ports        protos.PortsConfig
	parserConfig parserConfig
	transConfig  transactionConfig
	pub          transPub
}

// Application Layer tcp stream data to be stored on tcp connection context.
type connection struct {
	streams [2]*stream
	trans   transactions
	syncer  syncer
}

// Uni-directioal tcp stream state for parsing messages.
type stream struct {
	parser parser
}

type parseState uint8

const (
	stateUnsynced parseState = iota
	stateCommand
	// Request only state
	stateData
)

var (
	debugf = logp.MakeDebug("smtp")

	// use isDebug to guard debugf to minimize allocations (garbage
	// collection) when debug log is disabled.
	isDebug = false
)

func init() {
	protos.Register("smtp", New)
}

// New create and initializes a new smtp protocol analyzer instance.
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &smtpPlugin{}
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

func (smtp *smtpPlugin) init(results protos.Reporter, config *smtpConfig) error {
	if err := smtp.setFromConfig(config); err != nil {
		return err
	}
	smtp.pub.results = results

	isDebug = logp.IsDebug("smtp")
	return nil
}

func (smtp *smtpPlugin) setFromConfig(config *smtpConfig) error {

	// set module configuration
	if err := smtp.ports.Set(config.Ports); err != nil {
		return err
	}

	// set parser configuration
	parser := &smtp.parserConfig
	parser.maxBytes = tcp.TCPMaxDataInStream

	// set transaction correlator configuration
	trans := &smtp.transConfig
	trans.transactionTimeout = config.TransactionTimeout

	// set transaction publisher configuration
	pub := &smtp.pub
	pub.sendRequest = config.SendRequest
	pub.sendResponse = config.SendResponse
	pub.sendDataHeaders = config.SendDataHeaders
	pub.sendDataBody = config.SendDataBody

	return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func (smtp *smtpPlugin) ConnectionTimeout() time.Duration {
	return smtp.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func (smtp *smtpPlugin) GetPorts() []int {
	return smtp.ports.Ports
}

// Parse processes a TCP payload. Returns nil if connection state
// should be dropped (e.g. parser not in sync with tcp stream)
func (smtp *smtpPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("Parse SMTP exception")

	var err error
	errMsg := "Error in direction %d: %s"

	conn := smtp.ensureConnection(private)
	conn.ensureStream(dir, smtp, tcptuple)

	st := conn.streams[dir]

	if err = st.parser.append(pkt.Payload); err != nil {
		debugf(errMsg, dir, err)
		return nil
	}

	if conn.syncer.done {
		err = st.parser.process(pkt.Ts)
	} else {
		err = conn.syncer.process(pkt.Ts, dir)
	}

	if err != nil {
		debugf(errMsg, dir, err)
		return nil
	}

	return conn
}

// ReceivedFin handles TCP-FIN packet.
func (smtp *smtpPlugin) ReceivedFin(
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	return private
}

// GapInStream handles lost packets in tcp-stream.
func (smtp *smtpPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int,
	private protos.ProtocolData,
) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(smtp) exception")

	conn := getConnection(private)
	if conn != nil {
		debugf("Loss of synchronization due to gap in TCP stream")
	}

	// Drop state and let the parsers re-sync
	return nil, true
}

func (smtp *smtpPlugin) ensureConnection(
	private protos.ProtocolData,
) *connection {
	conn := getConnection(private)
	if conn == nil {
		conn = &connection{}
		conn.trans.init(&smtp.transConfig, smtp.pub.onTransaction)
	}
	return conn
}

func (conn *connection) dropStreams() {
	conn.streams[0] = nil
	conn.streams[1] = nil
}

func (conn *connection) ensureStream(
	dir uint8,
	smtp *smtpPlugin,
	tcptuple *common.TCPTuple,
) {
	st := conn.streams[dir]
	if st == nil {
		st = &stream{}
		st.parser.init(
			&smtp.parserConfig,
			&smtp.pub,
			func(msg *message) error {
				return conn.trans.onMessage(tcptuple.IPPort(), dir, msg)
			},
		)
		conn.streams[dir] = st
		conn.syncer.parsers[dir] = &st.parser
	}
}

func getConnection(private protos.ProtocolData) *connection {
	if private == nil {
		return nil
	}

	priv, ok := private.(*connection)
	if !ok {
		logp.Warn("smtp connection type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: smtp connection data not set")
		return nil
	}
	return priv
}
