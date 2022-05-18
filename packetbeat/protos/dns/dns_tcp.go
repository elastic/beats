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

package dns

import (
	"encoding/binary"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"

	mkdns "github.com/miekg/dns"
)

const maxDNSMessageSize = (1 << 16) - 1

// RFC 1035
// The 2 first bytes contain the length of the message
const decodeOffset = 2

// DnsStream contains DNS data from one side of a TCP transmission. A pair
// of DnsStream's are used to represent the full conversation.
type dnsStream struct {
	tcpTuple    *common.TCPTuple
	rawData     []byte
	parseOffset int
	message     *dnsMessage
}

// dnsConnectionData contains two DnsStream's that hold data from a complete TCP
// transmission. Element zero contains the response data. Element one contains
// the request data.
// prevRequest (previous Request) is used to add Notes to a transaction when a failing answer is encountered
type dnsConnectionData struct {
	data        [2]*dnsStream
	prevRequest *dnsMessage
}

func (dns *dnsPlugin) Parse(pkt *protos.Packet, tcpTuple *common.TCPTuple, dir uint8, private protos.ProtocolData) protos.ProtocolData {
	defer logp.Recover("Dns ParseTcp")

	debugf("Parsing packet addressed with %s of length %d.",
		pkt.Tuple.String(), len(pkt.Payload))

	conn := ensureDNSConnection(private)

	conn = dns.doParse(conn, pkt, tcpTuple, dir)
	if conn == nil {
		return nil
	}

	return conn
}

func ensureDNSConnection(private protos.ProtocolData) *dnsConnectionData {
	if private == nil {
		return &dnsConnectionData{}
	}

	conn, ok := private.(*dnsConnectionData)
	if !ok {
		logp.Warn("Dns connection data type error, create new one")
		return &dnsConnectionData{}
	}
	if conn == nil {
		logp.Warn("Unexpected: dns connection data not set, create new one")
		return &dnsConnectionData{}
	}

	return conn
}

func (dns *dnsPlugin) doParse(conn *dnsConnectionData, pkt *protos.Packet, tcpTuple *common.TCPTuple, dir uint8) *dnsConnectionData {
	stream := conn.data[dir]
	payload := pkt.Payload

	if stream == nil {
		stream = newStream(pkt, tcpTuple)
		conn.data[dir] = stream
	} else {
		if stream.message == nil { // nth message of the same stream
			stream.message = &dnsMessage{ts: pkt.Ts, tuple: pkt.Tuple}
		}

		stream.rawData = append(stream.rawData, payload...)
		if len(stream.rawData) > tcp.TCPMaxDataInStream {
			debugf("Stream data too large, dropping DNS stream")
			conn.data[dir] = nil
			return conn
		}
	}
	decodedData, err := stream.handleTCPRawData()
	if err != nil {

		if err == incompleteMsg {
			debugf("Waiting for more raw data")
			return conn
		}

		if dir == tcp.TCPDirectionReverse {
			dns.publishResponseError(conn, err)
		}

		debugf("%s addresses %s, length %d", err.Error(),
			tcpTuple.String(), len(stream.rawData))

		// This means that malformed requests or responses are being sent...
		// TODO: publish the situation also if Request
		conn.data[dir] = nil
		return conn
	}

	dns.messageComplete(conn, tcpTuple, dir, decodedData)
	stream.prepareForNewMessage()
	return conn
}

func newStream(pkt *protos.Packet, tcpTuple *common.TCPTuple) *dnsStream {
	return &dnsStream{
		tcpTuple: tcpTuple,
		rawData:  pkt.Payload,
		message:  &dnsMessage{ts: pkt.Ts, tuple: pkt.Tuple},
	}
}

func (dns *dnsPlugin) messageComplete(conn *dnsConnectionData, tcpTuple *common.TCPTuple, dir uint8, decodedData *mkdns.Msg) {
	dns.handleDNS(conn, tcpTuple, decodedData, dir)
}

func (dns *dnsPlugin) handleDNS(conn *dnsConnectionData, tcpTuple *common.TCPTuple, decodedData *mkdns.Msg, dir uint8) {
	message := conn.data[dir].message
	dnsTuple := dnsTupleFromIPPort(&message.tuple, transportTCP, decodedData.Id)

	message.cmdlineTuple = dns.watcher.FindProcessesTupleTCP(tcpTuple.IPPort())
	message.data = decodedData
	message.length += decodeOffset

	if decodedData.Response {
		dns.receivedDNSResponse(&dnsTuple, message)
		conn.prevRequest = nil
	} else /* Query */ {
		dns.receivedDNSRequest(&dnsTuple, message)
		conn.prevRequest = message
	}
}

func (stream *dnsStream) prepareForNewMessage() {
	stream.rawData = stream.rawData[stream.parseOffset:]
	stream.message = nil
	stream.parseOffset = 0
}

func (dns *dnsPlugin) ReceivedFin(tcpTuple *common.TCPTuple, dir uint8, private protos.ProtocolData) protos.ProtocolData {
	if private == nil {
		return nil
	}
	conn, ok := private.(*dnsConnectionData)
	if !ok {
		return private
	}
	stream := conn.data[dir]

	if stream == nil || stream.message == nil {
		return conn
	}

	decodedData, err := stream.handleTCPRawData()

	if err == nil {
		dns.messageComplete(conn, tcpTuple, dir, decodedData)
		return conn
	}

	if dir == tcp.TCPDirectionReverse {
		dns.publishResponseError(conn, err)
	}

	debugf("%s addresses %s, length %d", err.Error(),
		tcpTuple.String(), len(stream.rawData))

	return conn
}

func (dns *dnsPlugin) GapInStream(tcpTuple *common.TCPTuple, dir uint8, nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	if private == nil {
		return private, true
	}
	conn, ok := private.(*dnsConnectionData)
	if !ok {
		return private, false
	}
	stream := conn.data[dir]

	if stream == nil || stream.message == nil {
		return private, false
	}

	decodedData, err := stream.handleTCPRawData()

	if err == nil {
		dns.messageComplete(conn, tcpTuple, dir, decodedData)
		return private, true
	}

	if dir == tcp.TCPDirectionReverse {
		dns.publishResponseError(conn, err)
	}

	debugf("%s addresses %s, length %d", err.Error(),
		tcpTuple.String(), len(stream.rawData))
	debugf("Dropping the stream %s", tcpTuple.String())

	// drop the stream because it is binary Data and it would be unexpected to have a decodable message later
	return private, true
}

// Add Notes to the transaction about a failure in the response
// Publish and remove the transaction
func (dns *dnsPlugin) publishResponseError(conn *dnsConnectionData, err error) {
	streamOrigin := conn.data[tcp.TCPDirectionOriginal]
	streamReverse := conn.data[tcp.TCPDirectionReverse]

	if streamOrigin == nil || conn.prevRequest == nil || streamReverse == nil {
		return
	}

	dataOrigin := conn.prevRequest.data
	dnsTupleOrigin := dnsTupleFromIPPort(&conn.prevRequest.tuple, transportTCP, dataOrigin.Id)
	hashDNSTupleOrigin := (&dnsTupleOrigin).hashable()

	trans := dns.deleteTransaction(hashDNSTupleOrigin)

	if trans == nil { // happens if Parse, Gap or Fin already published the response error
		return
	}

	errDNS, ok := err.(*dnsError)
	if !ok {
		return
	}
	trans.notes = append(trans.notes, errDNS.responseError())

	// Should we publish the length (bytes_out) of the failed Response?
	// streamReverse.message.Length = len(streamReverse.rawData)
	// trans.Response = streamReverse.message

	dns.publishTransaction(trans)
	dns.deleteTransaction(hashDNSTupleOrigin)
}

// Manages data length prior to decoding the data and manages errors after decoding
func (stream *dnsStream) handleTCPRawData() (*mkdns.Msg, error) {
	rawData := stream.rawData
	messageLength := len(rawData)

	if messageLength < decodeOffset {
		return nil, incompleteMsg
	}

	if stream.message.length == 0 {
		stream.message.length = int(binary.BigEndian.Uint16(rawData[:decodeOffset]))
		messageLength := stream.message.length
		stream.parseOffset = messageLength + decodeOffset

		// TODO: This means that malformed requests or responses are being sent or
		// that someone is attempting to the DNS port for non-DNS traffic.
		// We might want to publish this in the future, for security reasons
		if messageLength <= 0 {
			return nil, zeroLengthMsg
		}
		if messageLength > maxDNSMessageSize { // Should never be true though ...
			return nil, unexpectedLengthMsg
		}
	}

	if messageLength < stream.parseOffset {
		return nil, incompleteMsg
	}

	decodedData, err := decodeDNSData(transportTCP, rawData[:stream.parseOffset])
	if err != nil {
		return nil, err
	}

	return decodedData, nil
}
