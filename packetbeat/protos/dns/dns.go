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
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
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

	results publisher.Client // Channel where results are pushed.
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

func (dns *Dns) Init(test_mode bool, results publisher.Client) error {
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
		event["method"] = dnsOpCodeToString(t.Request.Data.OpCode)
		if len(t.Request.Data.Questions) > 0 {
			event["query"] = dnsQuestionToString(t.Request.Data.Questions[0])
			event["resource"] = nameToString(t.Request.Data.Questions[0].Name)
		}
		addDnsToMapStr(dnsEvent, t.Response.Data, dns.Include_authorities,
			dns.Include_additionals)

		if t.Response.Data.ResponseCode == 0 {
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
		event["method"] = dnsOpCodeToString(t.Request.Data.OpCode)
		if len(t.Request.Data.Questions) > 0 {
			event["query"] = dnsQuestionToString(t.Request.Data.Questions[0])
			event["resource"] = nameToString(t.Request.Data.Questions[0].Name)
		}
		addDnsToMapStr(dnsEvent, t.Request.Data, dns.Include_authorities,
			dns.Include_additionals)

		if dns.Send_request {
			event["request"] = dnsToString(t.Request.Data)
		}
	} else if t.Response != nil {
		event["bytes_out"] = t.Response.Length
		event["method"] = dnsOpCodeToString(t.Response.Data.OpCode)
		if len(t.Response.Data.Questions) > 0 {
			event["query"] = dnsQuestionToString(t.Response.Data.Questions[0])
			event["resource"] = nameToString(t.Response.Data.Questions[0].Name)
		}
		addDnsToMapStr(dnsEvent, t.Response.Data, dns.Include_authorities,
			dns.Include_additionals)
		if dns.Send_response {
			event["response"] = dnsToString(t.Response.Data)
		}
	}

	dns.results.PublishEvent(event)
}

func (dns *Dns) expireTransaction(t *DnsTransaction) {
	t.Notes = append(t.Notes, NoResponse.Error())
	logp.Debug("dns", "%s %s", NoResponse.Error(), t.tuple.String())
	dns.publishTransaction(t)
}

// Adds the DNS message data to the supplied MapStr.
func addDnsToMapStr(m common.MapStr, dns *layers.DNS, authority bool, additional bool) {
	m["id"] = dns.ID
	m["op_code"] = dnsOpCodeToString(dns.OpCode)

	m["flags"] = common.MapStr{
		"authoritative":      dns.AA,
		"truncated_response": dns.TC,
		"recursion_desired":  dns.RD,
		"recursion_allowed":  dns.RA,
		// Need to add RFC4035 flag parsing to gopacket.
		//"authentic_data":     dns.AD
		//"checking_disabled":  dns.CD
	}
	m["response_code"] = dnsResponseCodeToString(dns.ResponseCode)

	if len(dns.Questions) > 0 {
		q := dns.Questions[0]
		m["question"] = common.MapStr{
			"name":  nameToString(q.Name),
			"type":  dnsTypeToString(q.Type),
			"class": dnsClassToString(q.Class),
		}
	}

	m["answers_count"] = len(dns.Answers)
	if len(dns.Answers) > 0 {
		m["answers"] = rrToMapStr(dns.Answers)
	}

	m["authorities_count"] = len(dns.Authorities)
	if authority && len(dns.Authorities) > 0 {
		m["authorities"] = rrToMapStr(dns.Authorities)
	}

	m["additionals_count"] = len(dns.Additionals)
	if additional && len(dns.Additionals) > 0 {
		m["additionals"] = rrToMapStr(dns.Additionals)
	}
}

// rrToMapStr converts an array of DNSResourceRecord's to an array of MapStr's.
func rrToMapStr(records []layers.DNSResourceRecord) []common.MapStr {
	mapStrArray := make([]common.MapStr, len(records))
	for i, r := range records {
		mapStr := common.MapStr{
			"name":  nameToString(r.Name),
			"type":  dnsTypeToString(r.Type),
			"class": dnsClassToString(r.Class),
			"ttl":   r.TTL,
		}
		mapStrArray[i] = mapStr

		switch r.Type {
		default:
			// We don't have special handling for this type so use the same
			// encoding used for names to output the raw data for this type.
			mapStr["data"] = nameToString(r.Data)
		case layers.DNSTypeA, layers.DNSTypeAAAA:
			mapStr["data"] = r.IP.String()
		case layers.DNSTypeSOA:
			mapStr["rname"] = nameToString(r.SOA.RName)
			mapStr["serial"] = r.SOA.Serial
			mapStr["refresh"] = r.SOA.Refresh
			mapStr["retry"] = r.SOA.Retry
			mapStr["expire"] = r.SOA.Expire
			mapStr["minimum"] = r.SOA.Minimum
			mapStr["data"] = nameToString(r.SOA.MName)
		case layers.DNSTypeMX:
			mapStr["preference"] = r.MX.Preference
			mapStr["data"] = nameToString(r.MX.Name)
		case layers.DNSTypeSRV:
			mapStr["priority"] = r.SRV.Priority
			mapStr["weight"] = r.SRV.Weight
			mapStr["port"] = r.SRV.Port
			mapStr["data"] = nameToString(r.SRV.Name)
		case layers.DNSTypeCNAME:
			mapStr["data"] = nameToString(r.CNAME)
		case layers.DNSTypePTR:
			mapStr["data"] = nameToString(r.PTR)
		case layers.DNSTypeNS:
			mapStr["data"] = nameToString(r.NS)
		}
	}

	return mapStrArray
}

// dnsQuestionToString converts a DNSQuestion to a string.
func dnsQuestionToString(q layers.DNSQuestion) string {
	name := nameToString(q.Name)
	if len(name) == 0 {
		name = "Root"
	}
	return fmt.Sprintf("class %s, type %s, %s", dnsClassToString(q.Class),
		dnsTypeToString(q.Type), name)
}

// dnsResourceRecordToString converts a DNSResourceRecord to a string.
func dnsResourceRecordToString(rr *layers.DNSResourceRecord) string {
	name := nameToString(rr.Name)
	if len(name) == 0 {
		name = "Root"
	}
	var data string
	switch rr.Type {
	default:
		// We don't have special handling for this type so use the same
		// encoding used for names to output the raw data for this type.
		data = nameToString(rr.Data)
	case layers.DNSTypeA, layers.DNSTypeAAAA:
		data = rr.IP.String()
	case layers.DNSTypeSOA:
		data = fmt.Sprintf("mname %s, rname %s, serial %d, refresh %d, "+
			"retry %d, expire %d, minimum %d", rr.SOA.MName, rr.SOA.RName,
			rr.SOA.Serial, rr.SOA.Refresh, rr.SOA.Retry, rr.SOA.Expire,
			rr.SOA.Minimum)
	case layers.DNSTypeMX:
		data = fmt.Sprintf("preference %d, %s", rr.MX.Preference, rr.MX.Name)
	case layers.DNSTypeSRV:
		data = fmt.Sprintf("priority %d, weight %d, port %d, %s", rr.SRV.Priority,
			rr.SRV.Weight, rr.SRV.Port, rr.SRV.Name)
	case layers.DNSTypeCNAME:
		data = nameToString(rr.CNAME)
	case layers.DNSTypePTR:
		data = nameToString(rr.PTR)
	case layers.DNSTypeNS:
		data = nameToString(rr.NS)
	}

	return fmt.Sprintf("%s: ttl %d, class %s, type %s, %s", name,
		int(rr.TTL), dnsClassToString(rr.Class),
		dnsTypeToString(rr.Type), data)
}

// dnsResourceRecordsToString converts an array of DNSResourceRecord's to a
// string.
func dnsResourceRecordsToString(r []layers.DNSResourceRecord) string {
	var rrStrs []string
	for _, rr := range r {
		rrStrs = append(rrStrs, dnsResourceRecordToString(&rr))
	}
	return strings.Join(rrStrs, "; ")
}

// dnsToString converts a DNS message to a string.
func dnsToString(dns *layers.DNS) string {
	var msgType string
	if dns.QR == Query {
		msgType = "query"
	} else {
		msgType = "response"
	}

	var t []string
	if dns.AA {
		t = append(t, "aa")
	}
	if dns.TC {
		t = append(t, "tc")
	}
	if dns.RD {
		t = append(t, "rd")
	}
	if dns.RA {
		t = append(t, "ra")
	}
	// Need to add RFC4035 flag parsing to gopacket.
	//if dns.AD { t = append(t, "ad") }
	//if dns.CD { t = append(t, "cd") }
	flags := strings.Join(t, " ")

	var a []string
	a = append(a, fmt.Sprintf("ID %d; QR %s; OPCODE %s; FLAGS %s; RCODE %s",
		dns.ID, msgType, dnsOpCodeToString(dns.OpCode), flags,
		dnsResponseCodeToString(dns.ResponseCode)))

	if len(dns.Questions) > 0 {
		t = []string{}
		for _, question := range dns.Questions {
			t = append(t, dnsQuestionToString(question))
		}
		a = append(a, fmt.Sprintf("QUESTION %s", strings.Join(t, "; ")))
	}

	if len(dns.Answers) > 0 {
		a = append(a, fmt.Sprintf("ANSWER %s",
			dnsResourceRecordsToString(dns.Answers)))
	}

	if len(dns.Authorities) > 0 {
		a = append(a, fmt.Sprintf("AUTHORITY %s",
			dnsResourceRecordsToString(dns.Authorities)))
	}

	if len(dns.Additionals) > 0 {
		a = append(a, fmt.Sprintf("ADDITIONAL %s",
			dnsResourceRecordsToString(dns.Additionals)))
	}

	return strings.Join(a, "; ")
}

// nameToString converts bytes representing a domain name to a string. Bytes
// below 32 or above 126 are represented as an escaped base10 integer (\DDD).
// Back slashes and quotes are escaped. Tabs, carriage returns, and line feeds
// will be converted to \t, \r and \n respectively.
func nameToString(name []byte) string {
	var s []byte
	for _, value := range name {
		switch value {
		default:
			if value < 32 || value >= 127 {
				// Unprintable characters are written as \\DDD (e.g. \\012).
				s = append(s, []byte(fmt.Sprintf("\\%03d", int(value)))...)
			} else {
				s = append(s, value)
			}
		case '"', '\\':
			s = append(s, '\\', value)
		case '\t':
			s = append(s, '\\', 't')
		case '\r':
			s = append(s, '\\', 'r')
		case '\n':
			s = append(s, '\\', 'n')
		}
	}
	return string(s)
}

// decodeDnsData decodes a byte array into a DNS struct. If an error occurs
// then the returnd dns pointer will be nil. This method recovers from panics
// and is concurrency-safe.
func decodeDnsData(transport Transport, rawData []byte) (dns *layers.DNS, err error) {
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

	d := &layers.DNS{}
	err = d.DecodeFromBytes(rawData[offset:], gopacket.NilDecodeFeedback)
	if err != nil {
		return nil, NonDnsMsg
	}
	return d, nil
}
