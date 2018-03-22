package graphite

import (
    "time"

    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/logp"

    "github.com/elastic/beats/packetbeat/protos"
    tcp1 "github.com/elastic/beats/packetbeat/protos/tcp"
    "github.com/elastic/beats/packetbeat/publish"
)

// TCP application level protocol analyzer plugin
type tcp struct {
    ports        protos.PortsConfig
    parserConfig parserConfig
    transConfig  transactionConfig
    pub          transPub
}

// Application Layer tcp stream data to be stored on tcp connection context.
type connection struct {
    streams[2] * stream
    trans   transactions
}

// JSON for Graphite application layer data
type JSON struct {
    MetricName      string
    MetricValue     float64
    MetricTimestamp int64
}

// Uni - directioal tcp stream state for parsing messages.
type stream struct {
    parser parser
}

var(
    debugf=logp.MakeDebug("graphite")

    // use isDebug / isDetailed to guard debugf / detailedf to minimize allocations
    // (garbage collection) when debug log is disabled.
    isDebug=false
)

func init() {
    protos.Register("graphite", New)
}

// New create and initializes a new tcp protocol analyzer instance.
func New(
    testMode bool,
    results publish.Transactions,
    cfg * common.Config,
)(protos.Plugin, error) {
    p: = &tcp{}
    config: = defaultConfig
    if !testMode {
        if err: = cfg.Unpack( & config)
        err != nil {
            return nil, err
        }
    }

    if err: = p.init(results, & config)
    err != nil {
        return nil, err
    }
    return p, nil
}

func(tcp * tcp) init(results publish.Transactions, config * graphiteConfig) error {
    if err: = tcp.setFromConfig(config)
    err != nil {
        return err
    }
    tcp.pub.results = results
    isDebug = logp.IsDebug("http")
    return nil
}

func(tcp * tcp) setFromConfig(config * graphiteConfig) error {

    // set module configuration
    if err: = tcp.ports.Set(config.Ports)
    err != nil {
        return err
    }

    // set parser configuration
    parser: = &tcp.parserConfig
    parser.maxBytes = tcp1.TCPMaxDataInStream

    // set transaction correlator configuration
    trans: = &tcp.transConfig
    trans.transactionTimeout = config.TransactionTimeout

    // set transaction publisher configuration
    pub: = &tcp.pub
    pub.sendRequest = config.SendRequest
    pub.sendResponse = config.SendResponse

    return nil
}

// ConnectionTimeout returns the per stream connection timeout.
// Return <= 0 to set default tcp module transaction timeout.
func(tcp * tcp) ConnectionTimeout() time.Duration {
    return tcp.transConfig.transactionTimeout
}

// GetPorts returns the ports numbers packets shall be processed for.
func(tcp * tcp) GetPorts()[]int {
    return tcp.ports.Ports
}

// Parse processes a TCP packet. Return nil if connection
// state shall be dropped(e.g. parser not in sync with tcp stream)
func(tcp * tcp) Parse(
    pkt * protos.Packet,
    tcptuple * common.TCPTuple, dir uint8,
    private protos.ProtocolData,
) protos.ProtocolData {
    defer logp.Recover("Parse tcp exception")

    conn: = tcp.ensureConnection(private)
    st: = conn.streams[dir]
    if st == nil {
        st = &stream{}
        st.parser.init( & tcp.parserConfig, func(msg * message) error {
            return conn.trans.onMessage(tcptuple.IPPort(), dir, msg)
        })
        conn.streams[dir] = st
    }

    if err: = st.parser.feed(pkt.Ts, pkt.Payload)
    err != nil {
        debugf("%v, dropping TCP stream for error in direction %v.", err, dir)
        tcp.onDropConnection(conn)
        return nil
    }
    return conn
}

// ReceivedFin handles TCP - FIN packet.
func(tcp * tcp) ReceivedFin(
    tcptuple * common.TCPTuple, dir uint8,
    private protos.ProtocolData,
) protos.ProtocolData {
    return private
}

// GapInStream handles lost packets in tcp - stream.
func(tcp * tcp) GapInStream(tcptuple * common.TCPTuple, dir uint8,
                            nbytes int,
                            private protos.ProtocolData,
                            )(protos.ProtocolData, bool) {
    conn: = getConnection(private)
    if conn != nil {
        tcp.onDropConnection(conn)
    }

    return nil, true
}

// onDropConnection processes and optionally sends incomplete
// transaction in case of connection being dropped due to error
func(tcp * tcp) onDropConnection(conn * connection) {

}

func(tcp * tcp) ensureConnection(private protos.ProtocolData) * connection {
    conn: = getConnection(private)
    if conn == nil {
        conn = &connection{}
            conn.trans.init(& tcp.transConfig, tcp.pub.onTransaction)
    }
    return conn
}

func(conn * connection) dropStreams() {
    conn.streams[0] = nil
    conn.streams[1] = nil
}

func getConnection(private protos.ProtocolData) * connection {
    if private == nil {
        return nil
    }
    priv, ok: = private.(*connection)
    if !ok {
        logp.Warn("graphite connection type error")
        return nil
    }
    if priv == nil {
        logp.Warn("Unexpected: graphite connection data not set")
        return nil
    }
    return priv
}
