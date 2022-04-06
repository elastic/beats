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
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/x509util"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
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
	includeDetailedFields  bool
	fingerprints           []*FingerprintAlgorithm
	transactionTimeout     time.Duration
	results                protos.Reporter
	watcher                procs.ProcessesWatcher
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
	watcher procs.ProcessesWatcher,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &tlsPlugin{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, watcher, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func (plugin *tlsPlugin) init(results protos.Reporter, watcher procs.ProcessesWatcher, config *tlsConfig) error {
	if err := plugin.setFromConfig(config); err != nil {
		return err
	}

	plugin.results = results
	plugin.watcher = watcher
	isDebug = logp.IsDebug("tls")

	return nil
}

func (plugin *tlsPlugin) setFromConfig(config *tlsConfig) error {
	plugin.ports = config.Ports
	plugin.sendCertificates = config.SendCertificates
	plugin.includeRawCertificates = config.IncludeRawCertificates
	plugin.includeDetailedFields = config.IncludeDetailedFields
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
	if conn.handshakeCompleted&(1<<dir) != 0 {
		return conn
	}

	st := conn.streams[dir]
	if st == nil {
		st = newStream(tcptuple)
		st.cmdlineTuple = plugin.watcher.FindProcessesTupleTCP(tcptuple.IPPort())
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
	private protos.ProtocolData,
) protos.ProtocolData {
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

	tls := ecs.Tls{
		Established: conn.handshakeCompleted > 1,
	}
	detailed := common.MapStr{}

	emptyHello := &helloMessage{}
	var clientHello, serverHello *helloMessage
	if client.parser.hello != nil {
		clientHello = client.parser.hello
		detailed["client_hello"] = clientHello.toMap()
		tls.ClientJa3, _ = getJa3Fingerprint(clientHello)
		tls.ClientSupportedCiphers = clientHello.supportedCiphers()
	} else {
		clientHello = emptyHello
	}
	if server.parser.hello != nil {
		serverHello = server.parser.hello
		detailed["server_hello"] = serverHello.toMap()
		tls.Cipher = serverHello.selected.cipherSuite.String()
	} else {
		serverHello = emptyHello
	}
	if server.parser.ocspResponseIsValid {
		detailed["ocsp_response"] = server.parser.ocspResponse.String()
	}
	if plugin.sendCertificates {
		if cert, chain := plugin.getCerts(client.parser.certificates); cert != nil {
			detailed["client_certificate"] = cert
			if chain != nil {
				detailed["client_certificate_chain"] = chain
			}
		}
		if cert, chain := plugin.getCerts(server.parser.certificates); cert != nil {
			detailed["server_certificate"] = cert
			if chain != nil {
				detailed["server_certificate_chain"] = chain
			}
		}
	}
	if plugin.includeRawCertificates {
		tls.ClientCertificateChain = getPEMCertChain(client.parser.certificates)
		tls.ServerCertificateChain = getPEMCertChain(server.parser.certificates)
	}
	if list := client.parser.certificates; len(list) > 0 {
		cert := list[0]
		hashCert(cert, plugin.fingerprints, map[string]*string{
			"md5":    &tls.ClientHashMd5,
			"sha1":   &tls.ClientHashSha1,
			"sha256": &tls.ClientHashSha256,
		})
		tls.ClientSubject = cert.Subject.String()
		tls.ClientIssuer = cert.Issuer.String()
		tls.ClientNotAfter = cert.NotAfter
		tls.ClientNotBefore = cert.NotBefore
	}
	if list := server.parser.certificates; len(list) > 0 {
		cert := list[0]
		hashCert(cert, plugin.fingerprints, map[string]*string{
			"md5":    &tls.ServerHashMd5,
			"sha1":   &tls.ServerHashSha1,
			"sha256": &tls.ServerHashSha256,
		})
		tls.ServerSubject = cert.Subject.String()
		tls.ServerIssuer = cert.Issuer.String()
		tls.ServerNotAfter = cert.NotAfter
		tls.ServerNotBefore = cert.NotBefore
	}
	detailed["client_certificate_requested"] = server.parser.certRequested

	// It is a bit tricky to detect the mechanism used for a resumed session. If the client offered a ticket, then
	// ticket is assumed as the method used for resumption even when a session ID is also used (as RFC-5077 requires).
	// It is not possible to tell whether the server accepted the ticket or the session ID.
	sessionIDMatch := len(clientHello.sessionID) != 0 && clientHello.sessionID == serverHello.sessionID
	ticketOffered := len(clientHello.ticket.value) != 0 && serverHello.ticket.present
	resumed := !client.parser.keyExchanged && !server.parser.keyExchanged && (sessionIDMatch || ticketOffered)

	tls.Resumed = resumed
	if resumed {
		if ticketOffered {
			detailed["resumption_method"] = "ticket"
		} else {
			detailed["resumption_method"] = "id"
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
		detailed["alerts"] = alerts
		detailed["alert_types"] = alertTypes
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

	// TLS version in use
	var version tlsVersion
	if !serverHello.version.IsZero() {
		var ok bool
		var raw []byte
		const supportedVersionsExt = 43
		if raw, ok = serverHello.extensions.Raw[supportedVersionsExt]; ok {
			version.major = raw[0]
			version.minor = raw[1]
		}
		if !ok {
			version = serverHello.version
		}
	} else if !clientHello.version.IsZero() {
		version = clientHello.version
	}
	detailed["version"] = version.String()
	pVer := version.GetProtocolVersion()
	tls.VersionProtocol, tls.Version = pVer.Protocol, pVer.Version

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

	// set "server.domain" to SNI, if provided
	if value, ok := clientHello.extensions.Parsed["server_name_indication"]; ok {
		if list, ok := value.([]string); ok && len(list) > 0 {
			pbf.Destination.Domain = list[0]
			tls.ClientServerName = list[0]
		}
	}
	// set next protocol from server's ALPN extension
	if value, ok := serverHello.extensions.Parsed["application_layer_protocol_negotiation"]; ok {
		if list, ok := value.([]string); ok && len(list) > 0 {
			tls.NextProtocol = list[0]
		}
	}

	// Serialize ECS TLS fields
	pb.MarshalStruct(fields, "tls", tls)
	if plugin.includeDetailedFields {
		if cert, ok := detailed["client_certificate"]; ok {
			fields.Put("tls.client.x509", cert)
			detailed.Delete("client_certificate")
		}
		if cert, ok := detailed["server_certificate"]; ok {
			fields.Put("tls.server.x509", cert)
			detailed.Delete("server_certificate")
		}
		fields.Put("tls.detailed", detailed)
	}

	if len(tls.ServerCertificateChain) > 0 {
		fields.Put("tls.server.certificate_chain", tls.ServerCertificateChain)
	}
	if len(tls.ClientCertificateChain) > 0 {
		fields.Put("tls.client.certificate_chain", tls.ClientCertificateChain)
	}
	if len(tls.ClientSupportedCiphers) > 0 {
		fields.Put("tls.client.supported_ciphers", tls.ClientSupportedCiphers)
	}
	// Enforce booleans (not serialized when false)
	if !tls.Established {
		fields.Put("tls.established", tls.Established)
	}
	if !tls.Resumed {
		fields.Put("tls.resumed", tls.Resumed)
	}
	return evt
}

func getPEMCertChain(certs []*x509.Certificate) (chain []string) {
	n := len(certs)
	if n == 0 {
		return
	}
	chain = make([]string, 0, n)
	for _, cert := range certs {
		chain = append(chain, x509util.CertToPEMString(cert))
	}
	return
}

func hashCert(cert *x509.Certificate, algos []*FingerprintAlgorithm, req map[string]*string) {
	for _, fp := range algos {
		if dst := req[fp.name]; dst != nil {
			*dst = strings.ToUpper(fp.algo.Hash(cert.Raw))
		}
	}
}

func (plugin *tlsPlugin) getCerts(certs []*x509.Certificate) (common.MapStr, []common.MapStr) {
	if len(certs) == 0 {
		return nil, nil
	}
	cert := certToMap(certs[0])
	if len(certs) == 1 {
		return cert, nil
	}
	chain := make([]common.MapStr, len(certs)-1)
	for idx := 1; idx < len(certs); idx++ {
		chain[idx-1] = certToMap(certs[idx])
	}
	return cert, chain
}

func (conn *tlsConnectionData) hasInfo() bool {
	return (conn.streams[0] != nil && conn.streams[0].parser.hasInfo()) ||
		(conn.streams[1] != nil && conn.streams[1].parser.hasInfo())
}
