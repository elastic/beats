package tls

import (
	"crypto/x509"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/applayer"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

type stream struct {
	applayer.Stream
	parser       parser
	tcptuple     *common.TCPTuple
	cmdlineTuple *common.CmdlineTuple
}

type tlsConnectionData struct {
	streams [2]*stream

	handshakeCompleted int8
	eventSent          bool
	startTime, endTime time.Time
}

// TLS protocol plugin
type tlsPlugin struct {
	// config
	ports                  []int
	sendCertificates       bool
	includeRawCertificates bool
	transactionTimeout     time.Duration
	results                protos.Reporter
}

var (
	debugf  = logp.MakeDebug("tls")
	isDebug = false

	// ensure that tlsPlugin fulfills the TCPPlugin interface
	_ protos.TCPPlugin = &tlsPlugin{}
)

func init() {
	protos.Register("tls", New)
}

// New returns a new instance of the TLS plugin
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &tlsPlugin{}
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

func (plugin *tlsPlugin) init(results protos.Reporter, config *tlsConfig) error {
	plugin.setFromConfig(config)

	plugin.results = results
	isDebug = logp.IsDebug("tls")

	return nil
}

func (plugin *tlsPlugin) setFromConfig(config *tlsConfig) {
	plugin.ports = config.Ports
	plugin.sendCertificates = config.SendCertificates
	plugin.includeRawCertificates = config.IncludeRawCertificates
	plugin.transactionTimeout = config.TransactionTimeout
}

func (plugin *tlsPlugin) GetPorts() []int {
	return plugin.ports
}

func (plugin *tlsPlugin) ConnectionTimeout() time.Duration {
	return plugin.transactionTimeout
}

func (plugin *tlsPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseTLS exception")

	conn := ensureTLSConnection(private)
	if private == nil {
		conn.startTime = pkt.Ts
	}
	conn = plugin.doParse(conn, pkt, tcptuple, dir)
	if conn == nil {
		return nil
	}
	return conn
}

func ensureTLSConnection(private protos.ProtocolData) *tlsConnectionData {
	if private == nil {
		return &tlsConnectionData{}
	}

	priv, ok := private.(*tlsConnectionData)
	if !ok {
		logp.Warn("tls connection data type error, creating a new one")
		return &tlsConnectionData{}
	}

	return priv
}

func (plugin *tlsPlugin) doParse(
	conn *tlsConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
) *tlsConnectionData {

	// Ignore further traffic after the handshake is completed (encrypted connection)
	// TODO: request/response analysis
	if 0 != conn.handshakeCompleted&(1<<dir) {
		return conn
	}

	st := conn.streams[dir]
	if st == nil {
		st = newStream(tcptuple)
		st.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())
		conn.streams[dir] = st
	}

	if err := st.Append(pkt.Payload); err != nil {
		if isDebug {
			debugf("%v, dropping TCP stream", err)
		}
		return nil
	}

	state := resultOK
	for state == resultOK && st.Buf.Len() > 0 {

		state = st.parser.parse(&st.Buf)
		switch state {

		case resultOK, resultMore:
			// no-op

		case resultFailed:
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.streams[dir] = nil
			if isDebug {
				debugf("non-TLS message: TCP stream dropped. Try parsing with the next segment")
			}

		case resultEncrypted:
			conn.handshakeCompleted |= 1 << dir
			if conn.handshakeCompleted == 3 {
				conn.endTime = pkt.Ts
				plugin.sendEvent(conn)
			}
		}
	}

	return conn
}

func newStream(tcptuple *common.TCPTuple) *stream {
	s := &stream{
		tcptuple: tcptuple,
	}
	s.Stream.Init(tcp.TCPMaxDataInStream)
	return s
}

func (plugin *tlsPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	if conn := ensureTLSConnection(private); conn != nil {
		plugin.sendEvent(conn)
	}
	return private
}

func (plugin *tlsPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	if conn := ensureTLSConnection(private); conn != nil {
		plugin.sendEvent(conn)
	}
	return private, true
}

func (plugin *tlsPlugin) sendEvent(conn *tlsConnectionData) {
	if !conn.eventSent {
		conn.eventSent = true
		if conn.hasInfo() {
			event := plugin.createEvent(conn)
			plugin.results(event)
		}
	}
}

func (plugin *tlsPlugin) createEvent(conn *tlsConnectionData) beat.Event {
	status := common.OK_STATUS
	if conn.handshakeCompleted < 2 {
		status = common.ERROR_STATUS
	}

	emptyStream := &stream{}
	client := conn.streams[0]
	server := conn.streams[1]
	if client == nil {
		client = emptyStream
	}
	if server == nil {
		server = emptyStream
	}
	if client.parser.direction == dirServer || server.parser.direction == dirClient {
		client, server = server, client
	}

	tls := common.MapStr{
		"handshake_completed": conn.handshakeCompleted > 1,
	}

	fingerprints := common.MapStr{}
	emptyHello := &helloMessage{}
	var clientHello, serverHello *helloMessage
	if client.parser.hello != nil {
		clientHello = client.parser.hello
		tls["client_hello"] = clientHello.toMap()
		hash, str := getJa3Fingerprint(clientHello)
		ja3 := common.MapStr{
			"hash": hash,
			"str":  str,
		}
		fingerprints["ja3"] = ja3
	} else {
		clientHello = emptyHello
	}
	if server.parser.hello != nil {
		serverHello = server.parser.hello
		tls["server_hello"] = serverHello.toMap()
	} else {
		serverHello = emptyHello
	}
	if cert, chain := getCerts(client.parser.certificates, plugin.includeRawCertificates); cert != nil {
		tls["client_certificate"] = cert
		if chain != nil {
			tls["client_certificate_chain"] = chain
		}
	}
	if plugin.sendCertificates {
		if cert, chain := getCerts(server.parser.certificates, plugin.includeRawCertificates); cert != nil {
			tls["server_certificate"] = cert
			if chain != nil {
				tls["server_certificate_chain"] = chain
			}
		}
	}
	tls["client_certificate_requested"] = server.parser.certRequested

	// It is a bit tricky to detect the mechanism used for a resumed session. If the client offered a ticket, then
	// ticket is assumed as the method used for resumption even when a session ID is also used (as RFC-5077 requires).
	// It is not possible to tell whether the server accepted the ticket or the session ID.
	sessionIDMatch := len(clientHello.sessionID) != 0 && clientHello.sessionID == serverHello.sessionID
	ticketOffered := len(clientHello.ticket.value) != 0 && serverHello.ticket.present
	resumed := !client.parser.keyExchanged && !server.parser.keyExchanged && (sessionIDMatch || ticketOffered)

	tls["resumed"] = resumed
	if resumed {
		if ticketOffered {
			tls["resumption_method"] = "ticket"
		} else {
			tls["resumption_method"] = "id"
		}
	}

	numAlerts := len(client.parser.alerts) + len(server.parser.alerts)
	alerts := make([]common.MapStr, 0, numAlerts)
	alertTypes := make([]string, 0, numAlerts)
	for _, alert := range client.parser.alerts {
		alerts = append(alerts, alert.toMap("client"))
		alertTypes = append(alertTypes, alert.code.String())
	}
	for _, alert := range server.parser.alerts {
		alerts = append(alerts, alert.toMap("server"))
		alertTypes = append(alertTypes, alert.code.String())
	}
	if numAlerts != 0 {
		tls["alerts"] = alerts
		tls["alert_types"] = alertTypes
	}

	src := &common.Endpoint{}
	dst := &common.Endpoint{}

	tcptuple := client.tcptuple
	if tcptuple == nil {
		tcptuple = server.tcptuple
	}
	if tcptuple != nil {
		src.IP = tcptuple.SrcIP.String()
		src.Port = tcptuple.SrcPort
		dst.IP = tcptuple.DstIP.String()
		dst.Port = tcptuple.DstPort
	}

	if client.cmdlineTuple != nil {
		src.Proc = string(client.cmdlineTuple.Src)
		dst.Proc = string(client.cmdlineTuple.Dst)
	} else if server.cmdlineTuple != nil {
		src.Proc = string(server.cmdlineTuple.Dst)
		dst.Proc = string(server.cmdlineTuple.Src)
	}

	if len(fingerprints) > 0 {
		tls["fingerprints"] = fingerprints
	}
	fields := common.MapStr{
		"type":   "tls",
		"status": status,
		"tls":    tls,
		"src":    src,
		"dst":    dst,
	}
	// set "server" to SNI, if provided
	if value, ok := clientHello.extensions.Parsed["server_name_indication"]; ok {
		if list, ok := value.([]string); ok && len(list) > 0 {
			fields["server"] = list[0]
		}
	}

	// set "responsetime" if handshake completed
	responseTime := int32(conn.endTime.Sub(conn.startTime) / time.Millisecond)
	if responseTime >= 0 {
		fields["responsetime"] = responseTime
	}

	timestamp := time.Now()
	return beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	}
}

func getCerts(certs []*x509.Certificate, includeRaw bool) (common.MapStr, []common.MapStr) {
	if len(certs) == 0 {
		return nil, nil
	}
	cert := certToMap(certs[0], includeRaw)
	if len(certs) == 1 {
		return cert, nil
	}
	chain := make([]common.MapStr, len(certs)-1)
	for idx := 1; idx < len(certs); idx++ {
		chain[idx-1] = certToMap(certs[idx], includeRaw)
	}
	return cert, chain
}

func (conn *tlsConnectionData) hasInfo() bool {
	return (conn.streams[0] != nil && conn.streams[0].parser.hasInfo()) ||
		(conn.streams[1] != nil && conn.streams[1].parser.hasInfo())
}
