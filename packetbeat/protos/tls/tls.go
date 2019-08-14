// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tls

import (
	"crypto/x509"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/pb"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/applayer"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

type stream struct {
	applayer.Stream
	parser       parser
	tcptuple     *common.TCPTuple
	cmdlineTuple *common.ProcessTuple
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
	fingerprints           []*FingerprintAlgorithm
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
	if err := plugin.setFromConfig(config); err != nil {
		return err
	}

	plugin.results = results
	isDebug = logp.IsDebug("tls")

	return nil
}

func (plugin *tlsPlugin) setFromConfig(config *tlsConfig) error {
	plugin.ports = config.Ports
	plugin.sendCertificates = config.SendCertificates
	plugin.includeRawCertificates = config.IncludeRawCertificates
	plugin.transactionTimeout = config.TransactionTimeout
	for _, hashName := range config.Fingerprints {
		algo, err := GetFingerprintAlgorithm(hashName)
		if err != nil {
			return err
		}
		plugin.fingerprints = append(plugin.fingerprints, algo)
	}
	return nil
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
		st.cmdlineTuple = procs.ProcWatcher.FindProcessesTupleTCP(tcptuple.IPPort())
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
	if cert, chain := plugin.getCerts(client.parser.certificates); cert != nil {
		tls["client_certificate"] = cert
		if chain != nil {
			tls["client_certificate_chain"] = chain
		}
	}
	if plugin.sendCertificates {
		if cert, chain := plugin.getCerts(server.parser.certificates); cert != nil {
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
	cmdlineTuple := client.cmdlineTuple
	if cmdlineTuple == nil {
		cmdlineTuple = server.cmdlineTuple
	}
	if tcptuple != nil && cmdlineTuple != nil {
		source, destination := common.MakeEndpointPair(tcptuple.BaseTuple, cmdlineTuple)
		src, dst = &source, &destination
	}

	if len(fingerprints) > 0 {
		tls["fingerprints"] = fingerprints
	}

	// TLS version in use
	if conn.handshakeCompleted > 1 {
		var version string
		if serverHello != nil {
			var ok bool
			if value, exists := serverHello.extensions.Parsed["supported_versions"]; exists {
				version, ok = value.(string)
			}
			if !ok {
				version = serverHello.version.String()
			}
		} else if clientHello != nil {
			version = clientHello.version.String()
		}
		tls["version"] = version
	}

	evt, pbf := pb.NewBeatEvent(conn.startTime)
	pbf.SetSource(src)
	pbf.SetDestination(dst)
	pbf.Event.Start = conn.startTime
	pbf.Event.End = conn.endTime
	pbf.Network.Transport = "tcp"
	pbf.Network.Protocol = "tls"

	fields := evt.Fields
	fields["type"] = pbf.Network.Protocol
	fields["status"] = status
	fields["tls"] = tls

	// set "server.domain" to SNI, if provided
	if value, ok := clientHello.extensions.Parsed["server_name_indication"]; ok {
		if list, ok := value.([]string); ok && len(list) > 0 {
			pbf.Destination.Domain = list[0]
		}
	}

	return evt
}

func (plugin *tlsPlugin) getCerts(certs []*x509.Certificate) (common.MapStr, []common.MapStr) {
	if len(certs) == 0 {
		return nil, nil
	}
	cert := certToMap(certs[0], plugin.includeRawCertificates, plugin.fingerprints)
	if len(certs) == 1 {
		return cert, nil
	}
	chain := make([]common.MapStr, len(certs)-1)
	for idx := 1; idx < len(certs); idx++ {
		chain[idx-1] = certToMap(certs[idx], plugin.includeRawCertificates, plugin.fingerprints)
	}
	return cert, chain
}

func (conn *tlsConnectionData) hasInfo() bool {
	return (conn.streams[0] != nil && conn.streams[0].parser.hasInfo()) ||
		(conn.streams[1] != nil && conn.streams[1].parser.hasInfo())
}
