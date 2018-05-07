package hl7v2

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"strings"
	"time"
)

// hl7v2Plugin application level protocol analyzer plugin
type hl7v2Plugin struct {
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
	debugf = logp.MakeDebug("hl7v2")

	// use isDebug/isDetailed to guard debugf/detailedf to minimize allocations
	// (garbage collection) when debug log is disabled.
	isDebug = false
)

func init() {
	protos.Register("hl7v2", New)
}

// New create and initializes a new hl7v2 protocol analyzer instance.
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &hl7v2Plugin{}
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

func (hp *hl7v2Plugin) init(results protos.Reporter, config *hl7v2Config) error {
	if err := hp.setFromConfig(config); err != nil {
		return err
	}
	hp.pub.results = results

	isDebug = logp.IsDebug("hl7v2")
	return nil
}

func (hp *hl7v2Plugin) setFromConfig(config *hl7v2Config) error {

	// set module configuration
	if err := hp.ports.Set(config.Ports); err != nil {
		return err
	}

	// set parser configuration
	parser := &hp.parserConfig
	parser.maxBytes = tcp.TCPMaxDataInStream
	parser.NewLineChars = strings.Replace(strings.Replace(config.NewLineChars, `\r`, "\r", -1), `\n`, "\n", -1)
	if len(parser.NewLineChars) == 0 {
		parser.NewLineChars = "\r"
	}

	// set transaction correlator configuration
	trans := &hp.transConfig
	trans.transactionTimeout = config.TransactionTimeout

	// set transaction publisher configuration
	pub := &hp.pub
	pub.sendRequest = config.SendRequest
	pub.sendResponse = config.SendResponse
	pub.NewLineChars = strings.Replace(strings.Replace(config.NewLineChars, `\r`, "\r", -1), `\n`, "\n", -1)
	if len(pub.NewLineChars) == 0 {
		pub.NewLineChars = "\r"
	}
	pub.SegmentSelectionMode = config.SegmentSelectionMode
	pub.FieldSelectionMode = config.FieldSelectionMode

	segmentsmap := make(map[string]bool)
	if len(config.Segments) > 0 {
		for _, segment := range config.Segments {
			segmentsmap[segment] = true
		}
		pub.segmentsmap = segmentsmap
	}
	fieldsmap := make(map[string]bool)
	if len(config.Fields) > 0 {
		for _, field := range config.Fields {
			fieldsmap[field] = true
		}
		pub.fieldsmap = fieldsmap
	}
	componentsmap := make(map[string]bool)
	if len(config.Components) > 0 {
		for _, component := range config.Components {
			componentsmap[component] = true
		}
		pub.componentsmap = componentsmap
	}

	fieldmappingmap := make(map[string]string)
	if len(config.FieldMappingMap) > 0 {
		for mappings := range config.FieldMappingMap {
			for field, mapping := range config.FieldMappingMap[mappings] {
				fieldmappingmap[field] = mapping
			}
		}
		pub.fieldmappingmap = fieldmappingmap
	}

	return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <=0 to set default tcp module transaction timeout.
func (hp *hl7v2Plugin) ConnectionTimeout() time.Duration {
	return hp.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func (hp *hl7v2Plugin) GetPorts() []int {
	return hp.ports.Ports
}

// Parse processes a TCP packet. Return nil if connection
// state shall be dropped (e.g. parser not in sync with tcp stream)
func (hp *hl7v2Plugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("Parse hl7v2Plugin exception")

	conn := hp.ensureConnection(private)
	st := conn.streams[dir]
	if st == nil {
		st = &stream{}
		st.parser.init(&hp.parserConfig, func(msg *message) error {
			return conn.trans.onMessage(tcptuple.IPPort(), dir, msg)
		})
		conn.streams[dir] = st
	}

	if err := st.parser.feed(pkt.Ts, pkt.Payload); err != nil {
		debugf("%v, dropping TCP stream for error in direction %v.", err, dir)
		hp.onDropConnection(conn)
		return nil
	}
	return conn
}

// ReceivedFin handles TCP-FIN packet.
func (hp *hl7v2Plugin) ReceivedFin(
	tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	return private
}

// GapInStream handles lost packets in tcp-stream.
func (hp *hl7v2Plugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int,
	private protos.ProtocolData,
) (protos.ProtocolData, bool) {
	conn := getConnection(private)
	if conn != nil {
		hp.onDropConnection(conn)
	}

	return nil, true
}

// onDropConnection processes and optionally sends incomplete
// transaction in case of connection being dropped due to error
func (hp *hl7v2Plugin) onDropConnection(conn *connection) {
}

func (hp *hl7v2Plugin) ensureConnection(private protos.ProtocolData) *connection {
	conn := getConnection(private)
	if conn == nil {
		conn = &connection{}
		conn.trans.init(&hp.transConfig, hp.pub.onTransaction)
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
		logp.Warn("hl7v2 connection type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: hl7v2 connection data not set")
		return nil
	}
	return priv
}
