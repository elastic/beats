package dns

import (
	"encoding/binary"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"

	mkdns "github.com/miekg/dns"
)

const MaxDnsMessageSize = (1 << 16) - 1

// RFC 1035
// The 2 first bytes contain the length of the message
const DecodeOffset = 2

// DnsStream contains DNS data from one side of a TCP transmission. A pair
// of DnsStream's are used to represent the full conversation.
type DnsStream struct {
	tcpTuple    *common.TcpTuple
	rawData     []byte
	parseOffset int
	message     *DnsMessage
}

// dnsConnectionData contains two DnsStream's that hold data from a complete TCP
// transmission. Element zero contains the response data. Element one contains
// the request data.
// prevRequest (previous Request) is used to add Notes to a transaction when a failing answer is encountered
type dnsConnectionData struct {
	Data        [2]*DnsStream
	prevRequest *DnsMessage
}

func (dns *Dns) Parse(pkt *protos.Packet, tcpTuple *common.TcpTuple, dir uint8, private protos.ProtocolData) protos.ProtocolData {
	defer logp.Recover("Dns ParseTcp")

	debugf("Parsing packet addressed with %s of length %d.",
		pkt.Tuple.String(), len(pkt.Payload))

	conn := ensureDnsConnection(private)

	conn = dns.doParse(conn, pkt, tcpTuple, dir)
	if conn == nil {
		return nil
	}

	return conn
}

func ensureDnsConnection(private protos.ProtocolData) *dnsConnectionData {
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

func (dns *Dns) doParse(conn *dnsConnectionData, pkt *protos.Packet, tcpTuple *common.TcpTuple, dir uint8) *dnsConnectionData {
	stream := conn.Data[dir]
	payload := pkt.Payload

	if stream == nil {
		stream = newStream(pkt, tcpTuple)
		conn.Data[dir] = stream
	} else {
		if stream.message == nil { // nth message of the same stream
			stream.message = &DnsMessage{Ts: pkt.Ts, Tuple: pkt.Tuple}
		}

		stream.rawData = append(stream.rawData, payload...)
		if len(stream.rawData) > tcp.TCP_MAX_DATA_IN_STREAM {
			debugf("Stream data too large, dropping DNS stream")
			conn.Data[dir] = nil
			return conn
		}
	}
	decodedData, err := stream.handleTcpRawData()

	if err != nil {

		if err == IncompleteMsg {
			debugf("Waiting for more raw data")
			return conn
		}

		if dir == tcp.TcpDirectionReverse {
			dns.publishResponseError(conn, err)
		}

		debugf("%s addresses %s, length %d", err.Error(),
			tcpTuple.String(), len(stream.rawData))

		// This means that malformed requests or responses are being sent...
		// TODO: publish the situation also if Request
		conn.Data[dir] = nil
		return conn
	}

	dns.messageComplete(conn, tcpTuple, dir, decodedData)
	stream.PrepareForNewMessage()
	return conn
}

func newStream(pkt *protos.Packet, tcpTuple *common.TcpTuple) *DnsStream {
	return &DnsStream{
		tcpTuple: tcpTuple,
		rawData:  pkt.Payload,
		message:  &DnsMessage{Ts: pkt.Ts, Tuple: pkt.Tuple},
	}
}

func (dns *Dns) messageComplete(conn *dnsConnectionData, tcpTuple *common.TcpTuple, dir uint8, decodedData *mkdns.Msg) {
	dns.handleDns(conn, tcpTuple, decodedData, dir)
}

func (dns *Dns) handleDns(conn *dnsConnectionData, tcpTuple *common.TcpTuple, decodedData *mkdns.Msg, dir uint8) {
	message := conn.Data[dir].message
	dnsTuple := DnsTupleFromIpPort(&message.Tuple, TransportTcp, decodedData.Id)

	message.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcpTuple.IpPort())
	message.Data = decodedData
	message.Length += DecodeOffset

	if decodedData.Response {
		dns.receivedDnsResponse(&dnsTuple, message)
		conn.prevRequest = nil
	} else /* Query */ {
		dns.receivedDnsRequest(&dnsTuple, message)
		conn.prevRequest = message
	}
}

func (stream *DnsStream) PrepareForNewMessage() {
	stream.rawData = stream.rawData[stream.parseOffset:]
	stream.message = nil
	stream.parseOffset = 0
}

func (dns *Dns) ReceivedFin(tcpTuple *common.TcpTuple, dir uint8, private protos.ProtocolData) protos.ProtocolData {
	if private == nil {
		return nil
	}
	conn, ok := private.(*dnsConnectionData)
	if !ok {
		return private
	}
	stream := conn.Data[dir]

	if stream == nil || stream.message == nil {
		return conn
	}

	decodedData, err := stream.handleTcpRawData()

	if err == nil {
		dns.messageComplete(conn, tcpTuple, dir, decodedData)
		return conn
	}

	if dir == tcp.TcpDirectionReverse {
		dns.publishResponseError(conn, err)
	}

	debugf("%s addresses %s, length %d", err.Error(),
		tcpTuple.String(), len(stream.rawData))

	return conn
}

func (dns *Dns) GapInStream(tcpTuple *common.TcpTuple, dir uint8, nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	if private == nil {
		return private, true
	}
	conn, ok := private.(*dnsConnectionData)
	if !ok {
		return private, false
	}
	stream := conn.Data[dir]

	if stream == nil || stream.message == nil {
		return private, false
	}

	decodedData, err := stream.handleTcpRawData()

	if err == nil {
		dns.messageComplete(conn, tcpTuple, dir, decodedData)
		return private, true
	}

	if dir == tcp.TcpDirectionReverse {
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
func (dns *Dns) publishResponseError(conn *dnsConnectionData, err error) {
	streamOrigin := conn.Data[tcp.TcpDirectionOriginal]
	streamReverse := conn.Data[tcp.TcpDirectionReverse]

	if streamOrigin == nil || conn.prevRequest == nil || streamReverse == nil {
		return
	}

	dataOrigin := conn.prevRequest.Data
	dnsTupleOrigin := DnsTupleFromIpPort(&conn.prevRequest.Tuple, TransportTcp, dataOrigin.Id)
	hashDnsTupleOrigin := (&dnsTupleOrigin).Hashable()

	trans := dns.deleteTransaction(hashDnsTupleOrigin)

	if trans == nil { // happens if Parse, Gap or Fin already published the response error
		return
	}

	errDns, ok := err.(*DNSError)
	if !ok {
		return
	}
	trans.Notes = append(trans.Notes, errDns.ResponseError())

	// Should we publish the length (bytes_out) of the failed Response?
	//streamReverse.message.Length = len(streamReverse.rawData)
	//trans.Response = streamReverse.message

	dns.publishTransaction(trans)
	dns.deleteTransaction(hashDnsTupleOrigin)
}

// Manages data length prior to decoding the data and manages errors after decoding
func (stream *DnsStream) handleTcpRawData() (*mkdns.Msg, error) {
	rawData := stream.rawData
	messageLength := len(rawData)

	if messageLength < DecodeOffset {
		return nil, IncompleteMsg
	}

	if stream.message.Length == 0 {
		stream.message.Length = int(binary.BigEndian.Uint16(rawData[:DecodeOffset]))
		messageLength := stream.message.Length
		stream.parseOffset = messageLength + DecodeOffset

		// TODO: This means that malformed requests or responses are being sent or
		// that someone is attempting to the DNS port for non-DNS traffic.
		// We might want to publish this in the future, for security reasons
		if messageLength <= 0 {
			return nil, ZeroLengthMsg
		}
		if messageLength > MaxDnsMessageSize { // Should never be true though ...
			return nil, UnexpectedLengthMsg
		}
	}

	if messageLength < stream.parseOffset {
		return nil, IncompleteMsg
	}

	decodedData, err := decodeDnsData(TransportTcp, rawData[:stream.parseOffset])

	if err != nil {
		return nil, err
	}

	return decodedData, nil
}
