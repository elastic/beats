// Package dns provides support for parsing DNS messages and reporting the
// results. This package supports the DNS protocol as defined by RFC 1034
// and RFC 1035. It does not have any special support for RFC 2671 (EDNS) or
// RFC 4035 (DNS Security Extensions), but since those specifications only
// add backwards compatible features there will be no issues handling the
// messages.

package dns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"

	mkdns "github.com/miekg/dns"
)

const MaxDnsTupleRawSize = 16 + 16 + 2 + 2 + 4 + 1

// Constants used to associate the DNS QR flag with a meaningful value.
const (
	Query    = false
	Response = true
)

// Transport protocol.
type Transport uint8

const (
	TransportTcp = iota
	TransportUdp
)

var TransportNames = []string{
	"tcp",
	"udp",
}

func (t Transport) String() string {
	if int(t) >= len(TransportNames) {
		return "impossible"
	}
	return TransportNames[t]
}

type HashableDnsTuple [MaxDnsTupleRawSize]byte

// DnsMessage contains a single DNS message.
type DnsMessage struct {
	Ts           time.Time          // Time when the message was received.
	Tuple        common.IpPortTuple // Source and destination addresses of packet.
	CmdlineTuple *common.CmdlineTuple
	Data         *mkdns.Msg // Parsed DNS packet data.
	Length       int        // Length of the DNS message in bytes (without DecodeOffset).
}

// DnsTuple contains source IP/port, destination IP/port, transport protocol,
// and DNS ID.
type DnsTuple struct {
	Ip_length          int
	Src_ip, Dst_ip     net.IP
	Src_port, Dst_port uint16
	Transport          Transport
	Id                 uint16

	raw    HashableDnsTuple // Src_ip:Src_port:Dst_ip:Dst_port:Transport:Id
	revRaw HashableDnsTuple // Dst_ip:Dst_port:Src_ip:Src_port:Transport:Id
}

func DnsTupleFromIpPort(t *common.IpPortTuple, trans Transport, id uint16) DnsTuple {
	tuple := DnsTuple{
		Ip_length: t.Ip_length,
		Src_ip:    t.Src_ip,
		Dst_ip:    t.Dst_ip,
		Src_port:  t.Src_port,
		Dst_port:  t.Dst_port,
		Transport: trans,
		Id:        id,
	}
	tuple.ComputeHashebles()

	return tuple
}

func (t DnsTuple) Reverse() DnsTuple {
	return DnsTuple{
		Ip_length: t.Ip_length,
		Src_ip:    t.Dst_ip,
		Dst_ip:    t.Src_ip,
		Src_port:  t.Dst_port,
		Dst_port:  t.Src_port,
		Transport: t.Transport,
		Id:        t.Id,
		raw:       t.revRaw,
		revRaw:    t.raw,
	}
}

func (t *DnsTuple) ComputeHashebles() {
	copy(t.raw[0:16], t.Src_ip)
	copy(t.raw[16:18], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
	copy(t.raw[18:34], t.Dst_ip)
	copy(t.raw[34:36], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})
	copy(t.raw[36:38], []byte{byte(t.Id >> 8), byte(t.Id)})
	t.raw[39] = byte(t.Transport)

	copy(t.revRaw[0:16], t.Dst_ip)
	copy(t.revRaw[16:18], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})
	copy(t.revRaw[18:34], t.Src_ip)
	copy(t.revRaw[34:36], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
	copy(t.revRaw[36:38], []byte{byte(t.Id >> 8), byte(t.Id)})
	t.revRaw[39] = byte(t.Transport)
}

func (t *DnsTuple) String() string {
	return fmt.Sprintf("DnsTuple src[%s:%d] dst[%s:%d] transport[%s] id[%d]",
		t.Src_ip.String(),
		t.Src_port,
		t.Dst_ip.String(),
		t.Dst_port,
		t.Transport,
		t.Id)
}

// Hashable returns a hashable value that uniquely identifies
// the DNS tuple.
func (t *DnsTuple) Hashable() HashableDnsTuple {
	return t.raw
}

// Hashable returns a hashable value that uniquely identifies
// the DNS tuple after swapping the source and destination.
func (t *DnsTuple) RevHashable() HashableDnsTuple {
	return t.revRaw
}

type Dns struct {
	// Configuration data.
	Ports               []int
	Send_request        bool
	Send_response       bool
	Include_authorities bool
	Include_additionals bool

	// Cache of active DNS transactions. The map key is the HashableDnsTuple
	// associated with the request.
	transactions       *common.Cache
	transactionTimeout time.Duration

	results publish.Transactions // Channel where results are pushed.
}

// getTransaction returns the transaction associated with the given
// HashableDnsTuple. The lookup key should be the HashableDnsTuple associated
// with the request (src is the requestor). Nil is returned if the entry
// does not exist.
func (dns *Dns) getTransaction(k HashableDnsTuple) *DnsTransaction {
	v := dns.transactions.Get(k)
	if v != nil {
		return v.(*DnsTransaction)
	}
	return nil
}

type DnsTransaction struct {
	ts           time.Time // Time when the request was received.
	tuple        DnsTuple  // Key used to track this transaction in the transactionsMap.
	ResponseTime int32     // Elapsed time in milliseconds between the request and response.
	Src          common.Endpoint
	Dst          common.Endpoint
	Transport    Transport
	Notes        []string

	Request  *DnsMessage
	Response *DnsMessage
}

func newTransaction(ts time.Time, tuple DnsTuple, cmd common.CmdlineTuple) *DnsTransaction {
	trans := &DnsTransaction{
		Transport: tuple.Transport,
		ts:        ts,
		tuple:     tuple,
	}
	trans.Src = common.Endpoint{
		Ip:   tuple.Src_ip.String(),
		Port: tuple.Src_port,
		Proc: string(cmd.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   tuple.Dst_ip.String(),
		Port: tuple.Dst_port,
		Proc: string(cmd.Dst),
	}
	return trans
}

// deleteTransaction deletes an entry from the transaction map and returns
// the deleted element. If the key does not exist then nil is returned.
func (dns *Dns) deleteTransaction(k HashableDnsTuple) *DnsTransaction {
	v := dns.transactions.Delete(k)
	if v != nil {
		return v.(*DnsTransaction)
	}
	return nil
}

func (dns *Dns) initDefaults() {
	dns.Send_request = false
	dns.Send_response = false
	dns.Include_authorities = false
	dns.Include_additionals = false
	dns.transactionTimeout = protos.DefaultTransactionExpiration
}

func (dns *Dns) setFromConfig(config config.Dns) error {

	dns.Ports = config.Ports

	if config.SendRequest != nil {
		dns.Send_request = *config.SendRequest
	}
	if config.SendResponse != nil {
		dns.Send_response = *config.SendResponse
	}
	if config.Include_authorities != nil {
		dns.Include_authorities = *config.Include_authorities
	}
	if config.Include_additionals != nil {
		dns.Include_additionals = *config.Include_additionals
	}
	if config.TransactionTimeout != nil && *config.TransactionTimeout > 0 {
		dns.transactionTimeout = time.Duration(*config.TransactionTimeout) * time.Second
	}

	return nil
}

func (dns *Dns) Init(test_mode bool, results publish.Transactions) error {
	dns.initDefaults()
	if !test_mode {
		dns.setFromConfig(config.ConfigSingleton.Protocols.Dns)
	}

	dns.transactions = common.NewCacheWithRemovalListener(
		dns.transactionTimeout,
		protos.DefaultTransactionHashSize,
		func(k common.Key, v common.Value) {
			trans, ok := v.(*DnsTransaction)
			if !ok {
				logp.Err("Expired value is not a *DnsTransaction.")
				return
			}
			dns.expireTransaction(trans)
		})
	dns.transactions.StartJanitor(dns.transactionTimeout)

	dns.results = results

	return nil
}

func (dns *Dns) GetPorts() []int {
	return dns.Ports
}

func (dns *Dns) ConnectionTimeout() time.Duration {
	return dns.transactionTimeout
}

func (dns *Dns) receivedDnsRequest(tuple *DnsTuple, msg *DnsMessage) {
	logp.Debug("dns", "Processing query. %s", tuple.String())

	trans := dns.deleteTransaction(tuple.Hashable())
	if trans != nil {
		// This happens if a client puts multiple requests in flight
		// with the same ID.
		trans.Notes = append(trans.Notes, DuplicateQueryMsg.Error())
		logp.Debug("dns", "%s %s", DuplicateQueryMsg.Error(), tuple.String())
		dns.publishTransaction(trans)
		dns.deleteTransaction(trans.tuple.Hashable())
	}

	trans = newTransaction(msg.Ts, *tuple, *msg.CmdlineTuple)
	dns.transactions.Put(tuple.Hashable(), trans)
	trans.Request = msg
}

func (dns *Dns) receivedDnsResponse(tuple *DnsTuple, msg *DnsMessage) {
	logp.Debug("dns", "Processing response. %s", tuple.String())

	trans := dns.getTransaction(tuple.RevHashable())
	if trans == nil {
		trans = newTransaction(msg.Ts, tuple.Reverse(), common.CmdlineTuple{
			Src: msg.CmdlineTuple.Dst, Dst: msg.CmdlineTuple.Src})
		trans.Notes = append(trans.Notes, OrphanedResponse.Error())
		logp.Debug("dns", "%s %s", OrphanedResponse.Error(), tuple.String())
	}

	trans.Response = msg
	dns.publishTransaction(trans)
	dns.deleteTransaction(trans.tuple.Hashable())
}

func (dns *Dns) publishTransaction(t *DnsTransaction) {
	if dns.results == nil {
		return
	}

	logp.Debug("dns", "Publishing transaction. %s", t.tuple.String())

	event := common.MapStr{}
	event["@timestamp"] = common.Time(t.ts)
	event["type"] = "dns"
	event["transport"] = t.Transport.String()
	event["src"] = &t.Src
	event["dst"] = &t.Dst
	event["status"] = common.ERROR_STATUS
	if len(t.Notes) == 1 {
		event["notes"] = t.Notes[0]
	} else if len(t.Notes) > 1 {
		event["notes"] = strings.Join(t.Notes, " ")
	}

	dnsEvent := common.MapStr{}
	event["dns"] = dnsEvent

	if t.Request != nil && t.Response != nil {
		event["bytes_in"] = t.Request.Length
		event["bytes_out"] = t.Response.Length
		event["responsetime"] = int32(t.Response.Ts.Sub(t.ts).Nanoseconds() / 1e6)
		event["method"] = dnsOpCodeToString(t.Request.Data.Opcode)
		if len(t.Request.Data.Question) > 0 {
			event["query"] = dnsQuestionToString(t.Request.Data.Question[0])
			event["resource"] = t.Request.Data.Question[0].Name
		}
		addDnsToMapStr(dnsEvent, t.Response.Data, dns.Include_authorities,
			dns.Include_additionals)

		if t.Response.Data.Rcode == 0 {
			event["status"] = common.OK_STATUS
		}

		if dns.Send_request {
			event["request"] = dnsToString(t.Request.Data)
		}
		if dns.Send_response {
			event["response"] = dnsToString(t.Response.Data)
		}
	} else if t.Request != nil {
		event["bytes_in"] = t.Request.Length
		event["method"] = dnsOpCodeToString(t.Request.Data.Opcode)
		if len(t.Request.Data.Question) > 0 {
			event["query"] = dnsQuestionToString(t.Request.Data.Question[0])
			event["resource"] = t.Request.Data.Question[0].Name
		}
		addDnsToMapStr(dnsEvent, t.Request.Data, dns.Include_authorities,
			dns.Include_additionals)

		if dns.Send_request {
			event["request"] = dnsToString(t.Request.Data)
		}
	} else if t.Response != nil {
		event["bytes_out"] = t.Response.Length
		event["method"] = dnsOpCodeToString(t.Response.Data.Opcode)
		if len(t.Response.Data.Question) > 0 {
			event["query"] = dnsQuestionToString(t.Response.Data.Question[0])
			event["resource"] = t.Response.Data.Question[0].Name
		}
		addDnsToMapStr(dnsEvent, t.Response.Data, dns.Include_authorities,
			dns.Include_additionals)
		if dns.Send_response {
			event["response"] = dnsToString(t.Response.Data)
		}
	}

	dns.results.PublishTransaction(event)
}

func (dns *Dns) expireTransaction(t *DnsTransaction) {
	t.Notes = append(t.Notes, NoResponse.Error())
	logp.Debug("dns", "%s %s", NoResponse.Error(), t.tuple.String())
	dns.publishTransaction(t)
}

// Adds the DNS message data to the supplied MapStr.
func addDnsToMapStr(m common.MapStr, dns *mkdns.Msg, authority bool, additional bool) {
	m["id"] = dns.Id
	m["op_code"] = dnsOpCodeToString(dns.Opcode)

	m["flags"] = common.MapStr{
		"authoritative":       dns.Authoritative,
		"truncated_response":  dns.Truncated,
		"recursion_desired":   dns.RecursionDesired,
		"recursion_available": dns.RecursionAvailable,
		"authentic_data":      dns.AuthenticatedData, // [RFC4035]
		"checking_disabled":   dns.CheckingDisabled,  // [RFC4035]
	}
	m["response_code"] = dnsResponseCodeToString(dns.Rcode)

	if len(dns.Question) > 0 {
		q := dns.Question[0]
		m["question"] = common.MapStr{
			"name":  q.Name,
			"type":  dnsTypeToString(q.Qtype),
			"class": dnsClassToString(q.Qclass),
		}
	}

	m["answers_count"] = len(dns.Answer)
	if len(dns.Answer) > 0 {
		m["answers"] = rrToMapStr(dns.Answer)
	}

	m["authorities_count"] = len(dns.Ns)
	if authority && len(dns.Ns) > 0 {
		m["authorities"] = rrToMapStr(dns.Ns)
	}

	m["additionals_count"] = len(dns.Extra)
	if additional && len(dns.Extra) > 0 {
		m["additionals"] = rrToMapStr(dns.Extra)
	}
}

// rrToMapStr converts an array of DNSResourceRecord's to an array of MapStr's.
func rrToMapStr(records []mkdns.RR) []common.MapStr {
	mapStrArray := make([]common.MapStr, len(records))
	for i, rr := range records {
		rrHeader := rr.Header()
		rrType := rrHeader.Rrtype

		mapStr := common.MapStr{
			"name":  rrHeader.Name,
			"type":  dnsTypeToString(rrType),
			"class": dnsClassToString(rrHeader.Class),
			"ttl":   strconv.FormatInt(int64(rrHeader.Ttl), 10),
		}
		mapStrArray[i] = mapStr

		switch x := rr.(type) {
		default:
			// We don't have special handling for this type
			logp.Debug("dns", "No special handling for RR type %s", dnsTypeToString(rrType))
			unsupportedRR := new(mkdns.RFC3597)
			err := unsupportedRR.ToRFC3597(x)
			if err == nil {
				rData, err := hexStringToString(unsupportedRR.Rdata)
				mapStr["data"] = rData
				if err != nil {
					logp.Debug("dns", "%s", err.Error())
				}
			} else {
				logp.Debug("dns", "Rdata for the unhandled RR type %s could not be fetched", dnsTypeToString(rrType))
			}
		case *mkdns.A:
			mapStr["data"] = x.A.String()
		case *mkdns.AAAA:
			mapStr["data"] = x.AAAA.String()
		case *mkdns.CNAME:
			mapStr["data"] = x.Target
		case *mkdns.MX:
			mapStr["preference"] = x.Preference
			mapStr["data"] = x.Mx
		case *mkdns.NS:
			mapStr["data"] = x.Ns
		case *mkdns.PTR:
			mapStr["data"] = x.Ptr
		case *mkdns.RFC3597:
			// Miekg/dns lib doesn't handle this type
			//TODO: write a test for this.
			logp.Debug("dns", "Unknown RR type %s", dnsTypeToString(rrType))
			rData, err := hexStringToString(x.Rdata)
			mapStr["data"] = rData
			if err != nil {
				logp.Debug("dns", "%s", err.Error())
			}
		case *mkdns.SOA:
			mapStr["rname"] = x.Mbox
			mapStr["serial"] = x.Serial
			mapStr["refresh"] = x.Refresh
			mapStr["retry"] = x.Retry
			mapStr["expire"] = x.Expire
			mapStr["minimum"] = x.Minttl
			mapStr["data"] = x.Ns
		case *mkdns.SRV:
			mapStr["priority"] = x.Priority
			mapStr["weight"] = x.Weight
			mapStr["port"] = x.Port
			mapStr["data"] = x.Target
		case *mkdns.TXT:
			mapStr["data"] = strings.Join(x.Txt, " ")
		}
	}

	return mapStrArray
}

// dnsQuestionToString converts a Question to a string.
func dnsQuestionToString(q mkdns.Question) string {
	name := q.Name

	return fmt.Sprintf("class %s, type %s, %s", dnsClassToString(q.Qclass),
		dnsTypeToString(q.Qtype), name)
}

// dnsResourceRecordToString converts a RR to a string.
func dnsResourceRecordToString(rr mkdns.RR) string {
	rrHeader := rr.Header()
	rrType := rrHeader.Rrtype

	var data string
	switch x := rr.(type) {
	default:
		// We don't have special handling for this type
		logp.Debug("dns", "No special handling for RR type %s", dnsTypeToString(rrType))
		unsupportedRR := new(mkdns.RFC3597)
		err := unsupportedRR.ToRFC3597(x)
		if err == nil {
			rData, err := hexStringToString(unsupportedRR.Rdata)
			data = rData
			if err != nil {
				logp.Debug("dns", "%s", err.Error())
			}
		} else {
			logp.Debug("dns", "Rdata for the unhandled RR type %s could not be fetched", dnsTypeToString(rrType))
		}
	case *mkdns.A:
		data = x.A.String()
	case *mkdns.AAAA:
		data = x.AAAA.String()
	case *mkdns.CNAME:
		data = x.Target
	case *mkdns.MX:
		data = fmt.Sprintf("preference %d, %s", x.Preference, x.Mx)
	case *mkdns.NS:
		data = x.Ns
	case *mkdns.PTR:
		data = x.Ptr
	case *mkdns.RFC3597:
		// Miekg/dns lib doesn't handle this type
		logp.Debug("dns", "Unknown RR type %s", dnsTypeToString(rrType))
		rData, err := hexStringToString(x.Rdata)
		data = rData
		if err != nil {
			logp.Debug("dns", "%s", err.Error())
		}
	case *mkdns.SOA:
		data = fmt.Sprintf("mname %s, rname %s, serial %d, refresh %d, "+
			"retry %d, expire %d, minimum %d", x.Ns, x.Mbox,
			x.Serial, x.Refresh, x.Retry, x.Expire,
			x.Minttl)
	case *mkdns.SRV:
		data = fmt.Sprintf("priority %d, weight %d, port %d, %s", x.Priority,
			x.Weight, x.Port, x.Target)
	case *mkdns.TXT:
		data = strings.Join(x.Txt, " ")
	}

	return fmt.Sprintf("%s: ttl %d, class %s, type %s, %s", rrHeader.Name,
		int(rrHeader.Ttl), dnsClassToString(rrHeader.Class),
		dnsTypeToString(rrType), data)
}

// dnsResourceRecordsToString converts an array of DNSResourceRecord's to a
// string.
func dnsResourceRecordsToString(r []mkdns.RR) string {
	var rrStrs []string
	for _, rr := range r {
		rrStrs = append(rrStrs, dnsResourceRecordToString(rr))
	}
	return strings.Join(rrStrs, "; ")
}

// dnsToString converts a DNS message to a string.
func dnsToString(dns *mkdns.Msg) string {
	var msgType string
	if dns.Response {
		msgType = "response"
	} else {
		msgType = "query"
	}

	var t []string
	if dns.Authoritative {
		t = append(t, "aa")
	}
	if dns.Truncated {
		t = append(t, "tc")
	}
	if dns.RecursionDesired {
		t = append(t, "rd")
	}
	if dns.RecursionAvailable {
		t = append(t, "ra")
	}
	if dns.AuthenticatedData {
		t = append(t, "ad")
	}
	if dns.CheckingDisabled {
		t = append(t, "cd")
	}
	flags := strings.Join(t, " ")

	var a []string
	a = append(a, fmt.Sprintf("ID %d; QR %s; OPCODE %s; FLAGS %s; RCODE %s",
		dns.Id, msgType, dnsOpCodeToString(dns.Opcode), flags,
		dnsResponseCodeToString(dns.Rcode)))

	if len(dns.Question) > 0 {
		t = []string{}
		for _, question := range dns.Question {
			t = append(t, dnsQuestionToString(question))
		}
		a = append(a, fmt.Sprintf("QUESTION %s", strings.Join(t, "; ")))
	}

	if len(dns.Answer) > 0 {
		a = append(a, fmt.Sprintf("ANSWER %s",
			dnsResourceRecordsToString(dns.Answer)))
	}

	if len(dns.Ns) > 0 {
		a = append(a, fmt.Sprintf("AUTHORITY %s",
			dnsResourceRecordsToString(dns.Ns)))
	}

	if len(dns.Extra) > 0 {
		a = append(a, fmt.Sprintf("ADDITIONAL %s",
			dnsResourceRecordsToString(dns.Extra)))
	}

	return strings.Join(a, "; ")
}

// decodeDnsData decodes a byte array into a DNS struct. If an error occurs
// then the returnd dns pointer will be nil. This method recovers from panics
// and is concurrency-safe.
// We do not handle Unpack ErrTruncated for now. See https://github.com/miekg/dns/pull/281
func decodeDnsData(transport Transport, rawData []byte) (dns *mkdns.Msg, err error) {
	var offset int
	if transport == TransportTcp {
		offset = DecodeOffset
	}

	// Recover from any panics that occur while parsing a packet.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	msg := &mkdns.Msg{}
	err = msg.Unpack(rawData[offset:])

	// Message should be more than 12 bytes.
	// The 12 bytes value corresponds to a message header length.
	// We use this check because Unpack does not return an error for some unvalid messages.
	// TODO: can a better solution be found?
	if msg.Len() <= 12 || err != nil {
		return nil, NonDnsMsg
	}
	return msg, nil
}
