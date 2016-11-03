// Package dns provides support for parsing DNS messages and reporting the
// results. This package supports the DNS protocol as defined by RFC 1034
// and RFC 1035. It does not have any special support for RFC 2671 (EDNS) or
// RFC 4035 (DNS Security Extensions), but since those specifications only
// add backwards compatible features there will be no issues handling the
// messages.
package dns

import (
	"bytes"
	"expvar"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"

	mkdns "github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

var (
	debugf = logp.MakeDebug("dns")
)

const MaxDNSTupleRawSize = 16 + 16 + 2 + 2 + 4 + 1

// Constants used to associate the DNS QR flag with a meaningful value.
const (
	Query    = false
	Response = true
)

// Transport protocol.
type Transport uint8

var (
	unmatchedRequests  = expvar.NewInt("dns.unmatched_requests")
	unmatchedResponses = expvar.NewInt("dns.unmatched_responses")
)

const (
	TransportTCP = iota
	TransportUDP
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

type HashableDNSTuple [MaxDNSTupleRawSize]byte

// DnsMessage contains a single DNS message.
type DNSMessage struct {
	Ts           time.Time          // Time when the message was received.
	Tuple        common.IPPortTuple // Source and destination addresses of packet.
	CmdlineTuple *common.CmdlineTuple
	Data         *mkdns.Msg // Parsed DNS packet data.
	Length       int        // Length of the DNS message in bytes (without DecodeOffset).
}

// DnsTuple contains source IP/port, destination IP/port, transport protocol,
// and DNS ID.
type DNSTuple struct {
	IPLength         int
	SrcIP, DstIP     net.IP
	SrcPort, DstPort uint16
	Transport        Transport
	ID               uint16

	raw    HashableDNSTuple // Src_ip:Src_port:Dst_ip:Dst_port:Transport:Id
	revRaw HashableDNSTuple // Dst_ip:Dst_port:Src_ip:Src_port:Transport:Id
}

func DNSTupleFromIPPort(t *common.IPPortTuple, trans Transport, id uint16) DNSTuple {
	tuple := DNSTuple{
		IPLength:  t.IPLength,
		SrcIP:     t.SrcIP,
		DstIP:     t.DstIP,
		SrcPort:   t.SrcPort,
		DstPort:   t.DstPort,
		Transport: trans,
		ID:        id,
	}
	tuple.ComputeHashebles()

	return tuple
}

func (t DNSTuple) Reverse() DNSTuple {
	return DNSTuple{
		IPLength:  t.IPLength,
		SrcIP:     t.DstIP,
		DstIP:     t.SrcIP,
		SrcPort:   t.DstPort,
		DstPort:   t.SrcPort,
		Transport: t.Transport,
		ID:        t.ID,
		raw:       t.revRaw,
		revRaw:    t.raw,
	}
}

func (t *DNSTuple) ComputeHashebles() {
	copy(t.raw[0:16], t.SrcIP)
	copy(t.raw[16:18], []byte{byte(t.SrcPort >> 8), byte(t.SrcPort)})
	copy(t.raw[18:34], t.DstIP)
	copy(t.raw[34:36], []byte{byte(t.DstPort >> 8), byte(t.DstPort)})
	copy(t.raw[36:38], []byte{byte(t.ID >> 8), byte(t.ID)})
	t.raw[39] = byte(t.Transport)

	copy(t.revRaw[0:16], t.DstIP)
	copy(t.revRaw[16:18], []byte{byte(t.DstPort >> 8), byte(t.DstPort)})
	copy(t.revRaw[18:34], t.SrcIP)
	copy(t.revRaw[34:36], []byte{byte(t.SrcPort >> 8), byte(t.SrcPort)})
	copy(t.revRaw[36:38], []byte{byte(t.ID >> 8), byte(t.ID)})
	t.revRaw[39] = byte(t.Transport)
}

func (t *DNSTuple) String() string {
	return fmt.Sprintf("DnsTuple src[%s:%d] dst[%s:%d] transport[%s] id[%d]",
		t.SrcIP.String(),
		t.SrcPort,
		t.DstIP.String(),
		t.DstPort,
		t.Transport,
		t.ID)
}

// Hashable returns a hashable value that uniquely identifies
// the DNS tuple.
func (t *DNSTuple) Hashable() HashableDNSTuple {
	return t.raw
}

// Hashable returns a hashable value that uniquely identifies
// the DNS tuple after swapping the source and destination.
func (t *DNSTuple) RevHashable() HashableDNSTuple {
	return t.revRaw
}

type DNS struct {
	// Configuration data.
	Ports              []int
	SendRequest        bool
	SendResponse       bool
	IncludeAuthorities bool
	IncludeAdditionals bool

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
func (dns *DNS) getTransaction(k HashableDNSTuple) *DNSTransaction {
	v := dns.transactions.Get(k)
	if v != nil {
		return v.(*DNSTransaction)
	}
	return nil
}

type DNSTransaction struct {
	ts           time.Time // Time when the request was received.
	tuple        DNSTuple  // Key used to track this transaction in the transactionsMap.
	ResponseTime int32     // Elapsed time in milliseconds between the request and response.
	Src          common.Endpoint
	Dst          common.Endpoint
	Transport    Transport
	Notes        []string

	Request  *DNSMessage
	Response *DNSMessage
}

func init() {
	protos.Register("dns", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &DNS{}
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

func (dns *DNS) init(results publish.Transactions, config *dnsConfig) error {
	dns.setFromConfig(config)
	dns.transactions = common.NewCacheWithRemovalListener(
		dns.transactionTimeout,
		protos.DefaultTransactionHashSize,
		func(k common.Key, v common.Value) {
			trans, ok := v.(*DNSTransaction)
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

func (dns *DNS) setFromConfig(config *dnsConfig) error {
	dns.Ports = config.Ports
	dns.SendRequest = config.SendRequest
	dns.SendResponse = config.SendResponse
	dns.IncludeAuthorities = config.IncludeAuthorities
	dns.IncludeAdditionals = config.IncludeAdditionals
	dns.transactionTimeout = config.TransactionTimeout
	return nil
}

func newTransaction(ts time.Time, tuple DNSTuple, cmd common.CmdlineTuple) *DNSTransaction {
	trans := &DNSTransaction{
		Transport: tuple.Transport,
		ts:        ts,
		tuple:     tuple,
	}
	trans.Src = common.Endpoint{
		IP:   tuple.SrcIP.String(),
		Port: tuple.SrcPort,
		Proc: string(cmd.Src),
	}
	trans.Dst = common.Endpoint{
		IP:   tuple.DstIP.String(),
		Port: tuple.DstPort,
		Proc: string(cmd.Dst),
	}
	return trans
}

// deleteTransaction deletes an entry from the transaction map and returns
// the deleted element. If the key does not exist then nil is returned.
func (dns *DNS) deleteTransaction(k HashableDNSTuple) *DNSTransaction {
	v := dns.transactions.Delete(k)
	if v != nil {
		return v.(*DNSTransaction)
	}
	return nil
}

func (dns *DNS) GetPorts() []int {
	return dns.Ports
}

func (dns *DNS) ConnectionTimeout() time.Duration {
	return dns.transactionTimeout
}

func (dns *DNS) receivedDNSRequest(tuple *DNSTuple, msg *DNSMessage) {
	debugf("Processing query. %s", tuple.String())

	trans := dns.deleteTransaction(tuple.Hashable())
	if trans != nil {
		// This happens if a client puts multiple requests in flight
		// with the same ID.
		trans.Notes = append(trans.Notes, DuplicateQueryMsg.Error())
		debugf("%s %s", DuplicateQueryMsg.Error(), tuple.String())
		dns.publishTransaction(trans)
		dns.deleteTransaction(trans.tuple.Hashable())
	}

	trans = newTransaction(msg.Ts, *tuple, *msg.CmdlineTuple)

	if tuple.Transport == TransportUDP && (msg.Data.IsEdns0() != nil) && msg.Length > MaxDNSPacketSize {
		trans.Notes = append(trans.Notes, UDPPacketTooLarge.Error())
		debugf("%s", UDPPacketTooLarge.Error())
	}

	dns.transactions.Put(tuple.Hashable(), trans)
	trans.Request = msg
}

func (dns *DNS) receivedDNSResponse(tuple *DNSTuple, msg *DNSMessage) {
	debugf("Processing response. %s", tuple.String())

	trans := dns.getTransaction(tuple.RevHashable())
	if trans == nil {
		trans = newTransaction(msg.Ts, tuple.Reverse(), common.CmdlineTuple{
			Src: msg.CmdlineTuple.Dst, Dst: msg.CmdlineTuple.Src})
		trans.Notes = append(trans.Notes, OrphanedResponse.Error())
		debugf("%s %s", OrphanedResponse.Error(), tuple.String())
		unmatchedResponses.Add(1)
	}

	trans.Response = msg

	if tuple.Transport == TransportUDP {
		respIsEdns := msg.Data.IsEdns0() != nil
		if !respIsEdns && msg.Length > MaxDNSPacketSize {
			trans.Notes = append(trans.Notes, UDPPacketTooLarge.ResponseError())
			debugf("%s", UDPPacketTooLarge.ResponseError())
		}

		request := trans.Request
		if request != nil {
			reqIsEdns := request.Data.IsEdns0() != nil

			switch {
			case reqIsEdns && !respIsEdns:
				trans.Notes = append(trans.Notes, RespEdnsNoSupport.Error())
				debugf("%s %s", RespEdnsNoSupport.Error(), tuple.String())
			case !reqIsEdns && respIsEdns:
				trans.Notes = append(trans.Notes, RespEdnsUnexpected.Error())
				debugf("%s %s", RespEdnsUnexpected.Error(), tuple.String())
			}
		}
	}

	dns.publishTransaction(trans)
	dns.deleteTransaction(trans.tuple.Hashable())
}

func (dns *DNS) publishTransaction(t *DNSTransaction) {
	if dns.results == nil {
		return
	}

	debugf("Publishing transaction. %s", t.tuple.String())

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
		addDNSToMapStr(dnsEvent, t.Response.Data, dns.IncludeAuthorities,
			dns.IncludeAdditionals)

		if t.Response.Data.Rcode == 0 {
			event["status"] = common.OK_STATUS
		}

		if dns.SendRequest {
			event["request"] = dnsToString(t.Request.Data)
		}
		if dns.SendResponse {
			event["response"] = dnsToString(t.Response.Data)
		}
	} else if t.Request != nil {
		event["bytes_in"] = t.Request.Length
		event["method"] = dnsOpCodeToString(t.Request.Data.Opcode)
		if len(t.Request.Data.Question) > 0 {
			event["query"] = dnsQuestionToString(t.Request.Data.Question[0])
			event["resource"] = t.Request.Data.Question[0].Name
		}
		addDNSToMapStr(dnsEvent, t.Request.Data, dns.IncludeAuthorities,
			dns.IncludeAdditionals)

		if dns.SendRequest {
			event["request"] = dnsToString(t.Request.Data)
		}
	} else if t.Response != nil {
		event["bytes_out"] = t.Response.Length
		event["method"] = dnsOpCodeToString(t.Response.Data.Opcode)
		if len(t.Response.Data.Question) > 0 {
			event["query"] = dnsQuestionToString(t.Response.Data.Question[0])
			event["resource"] = t.Response.Data.Question[0].Name
		}
		addDNSToMapStr(dnsEvent, t.Response.Data, dns.IncludeAuthorities,
			dns.IncludeAdditionals)
		if dns.SendResponse {
			event["response"] = dnsToString(t.Response.Data)
		}
	}

	dns.results.PublishTransaction(event)
}

func (dns *DNS) expireTransaction(t *DNSTransaction) {
	t.Notes = append(t.Notes, NoResponse.Error())
	debugf("%s %s", NoResponse.Error(), t.tuple.String())
	dns.publishTransaction(t)
	unmatchedRequests.Add(1)
}

// Adds the DNS message data to the supplied MapStr.
func addDNSToMapStr(m common.MapStr, dns *mkdns.Msg, authority bool, additional bool) {
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
		qMapStr := common.MapStr{
			"name":  q.Name,
			"type":  dnsTypeToString(q.Qtype),
			"class": dnsClassToString(q.Qclass),
		}
		m["question"] = qMapStr

		eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(strings.TrimRight(q.Name, "."))
		if err == nil {
			qMapStr["etld_plus_one"] = eTLDPlusOne + "."
		}
	}

	rrOPT := dns.IsEdns0()
	if rrOPT != nil {
		m["opt"] = optToMapStr(rrOPT)
	}

	m["answers_count"] = len(dns.Answer)
	if len(dns.Answer) > 0 {
		m["answers"] = rrsToMapStrs(dns.Answer)
	}

	m["authorities_count"] = len(dns.Ns)
	if authority && len(dns.Ns) > 0 {
		m["authorities"] = rrsToMapStrs(dns.Ns)
	}

	if rrOPT != nil {
		m["additionals_count"] = len(dns.Extra) - 1
	} else {
		m["additionals_count"] = len(dns.Extra)
	}
	if additional && len(dns.Extra) > 0 {
		rrsMapStrs := rrsToMapStrs(dns.Extra)
		// We do not want OPT RR to appear in the 'additional' section,
		// that's why rrsMapStrs could be empty even though len(dns.Extra) > 0
		if len(rrsMapStrs) > 0 {
			m["additionals"] = rrsMapStrs
		}
	}

}

func optToMapStr(rrOPT *mkdns.OPT) common.MapStr {
	optMapStr := common.MapStr{
		"do":        rrOPT.Do(), // true if DNSSEC
		"version":   strconv.FormatUint(uint64(rrOPT.Version()), 10),
		"udp_size":  rrOPT.UDPSize(),
		"ext_rcode": dnsResponseCodeToString(rrOPT.ExtendedRcode()),
	}
	for _, o := range rrOPT.Option {
		switch o.(type) {
		case *mkdns.EDNS0_DAU:
			optMapStr["dau"] = o.String()
		case *mkdns.EDNS0_DHU:
			optMapStr["dhu"] = o.String()
		case *mkdns.EDNS0_EXPIRE:
			optMapStr["local"] = o.String()
		case *mkdns.EDNS0_LLQ:
			optMapStr["llq"] = o.String()
		case *mkdns.EDNS0_LOCAL:
			optMapStr["local"] = o.String()
		case *mkdns.EDNS0_N3U:
			optMapStr["n3u"] = o.String()
		case *mkdns.EDNS0_NSID:
			optMapStr["nsid"] = o.String()
		case *mkdns.EDNS0_SUBNET:
			var draft string
			if o.(*mkdns.EDNS0_SUBNET).DraftOption {
				draft = " draft"
			}
			optMapStr["subnet"] = o.String() + draft
		case *mkdns.EDNS0_UL:
			optMapStr["ul"] = o.String()
		}
	}
	return optMapStr
}

// rrsToMapStr converts an slice of RR's to an slice of MapStr's.
func rrsToMapStrs(records []mkdns.RR) []common.MapStr {
	mapStrSlice := make([]common.MapStr, 0, len(records))
	for _, rr := range records {
		rrHeader := rr.Header()

		mapStr := rrToMapStr(rr)
		if len(mapStr) == 0 { // OPT pseudo-RR returns an empty MapStr
			continue
		}
		mapStr["name"] = rrHeader.Name
		mapStr["type"] = dnsTypeToString(rrHeader.Rrtype)
		mapStr["class"] = dnsClassToString(rrHeader.Class)
		mapStr["ttl"] = strconv.FormatInt(int64(rrHeader.Ttl), 10)
		mapStrSlice = append(mapStrSlice, mapStr)
	}
	return mapStrSlice
}

// Convert all RDATA fields of a RR to a single string
// fields are ordered alphabetically with 'data' as the last element
//
// TODO An improvement would be to replace 'data' by the real field name
// It would require some changes in unit tests
func rrToString(rr mkdns.RR) string {
	var st string
	var keys []string

	mapStr := rrToMapStr(rr)
	data, ok := mapStr["data"]
	delete(mapStr, "data")

	for k := range mapStr {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	for _, k := range keys {
		v := mapStr[k]
		switch x := v.(type) {
		case int:
			fmt.Fprintf(&b, "%s %d, ", k, x)
		case string:
			fmt.Fprintf(&b, "%s %s, ", k, x)
		}
	}
	if !ok {
		st = strings.TrimSuffix(b.String(), ", ")
		return st
	}

	switch x := data.(type) {
	case int:
		fmt.Fprintf(&b, "%d", x)
	case string:
		fmt.Fprintf(&b, "%s", x)
	}
	return b.String()
}

func rrToMapStr(rr mkdns.RR) common.MapStr {
	mapStr := common.MapStr{}
	rrType := rr.Header().Rrtype

	switch x := rr.(type) {
	default:
		// We don't have special handling for this type
		debugf("No special handling for RR type %s", dnsTypeToString(rrType))
		unsupportedRR := new(mkdns.RFC3597)
		err := unsupportedRR.ToRFC3597(x)
		if err == nil {
			rData, err := hexStringToString(unsupportedRR.Rdata)
			mapStr["data"] = rData
			if err != nil {
				debugf("%s", err.Error())
			}
		} else {
			debugf("Rdata for the unhandled RR type %s could not be fetched", dnsTypeToString(rrType))
		}
	case *mkdns.A:
		mapStr["data"] = x.A.String()
	case *mkdns.AAAA:
		mapStr["data"] = x.AAAA.String()
	case *mkdns.CNAME:
		mapStr["data"] = x.Target
	case *mkdns.DNSKEY:
		mapStr["flags"] = strconv.Itoa(int(x.Flags))
		mapStr["protocol"] = strconv.Itoa(int(x.Protocol))
		mapStr["algorithm"] = dnsAlgorithmToString(x.Algorithm)
		mapStr["data"] = x.PublicKey
	case *mkdns.DS:
		mapStr["key_tag"] = strconv.Itoa(int(x.KeyTag))
		mapStr["algorithm"] = dnsAlgorithmToString(x.Algorithm)
		mapStr["digest_type"] = dnsHashToString(x.DigestType)
		mapStr["data"] = strings.ToUpper(x.Digest)
	case *mkdns.MX:
		mapStr["preference"] = x.Preference
		mapStr["data"] = x.Mx
	case *mkdns.NS:
		mapStr["data"] = x.Ns
	case *mkdns.NSEC:
		mapStr["type_bits"] = dnsTypeBitsMapToString(x.TypeBitMap)
		mapStr["data"] = x.NextDomain
	case *mkdns.NSEC3:
		mapStr["hash"] = dnsHashToString(x.Hash)
		mapStr["flags"] = strconv.Itoa(int(x.Flags))
		mapStr["iterations"] = strconv.Itoa(int(x.Iterations))
		mapStr["salt"] = dnsSaltToString(x.Salt)
		mapStr["type_bits"] = dnsTypeBitsMapToString(x.TypeBitMap)
		mapStr["data"] = x.NextDomain
	case *mkdns.NSEC3PARAM:
		mapStr["hash"] = dnsHashToString(x.Hash)
		mapStr["flags"] = strconv.Itoa(int(x.Flags))
		mapStr["iterations"] = strconv.Itoa(int(x.Iterations))
		mapStr["data"] = dnsSaltToString(x.Salt)
	case *mkdns.OPT: // EDNS [RFC6891]
		// OPT pseudo-RR is managed in addDnsToMapStr function
		return nil
	case *mkdns.PTR:
		mapStr["data"] = x.Ptr
	case *mkdns.RFC3597:
		// Miekg/dns lib doesn't handle this type
		debugf("Unknown RR type %s", dnsTypeToString(rrType))
		rData, err := hexStringToString(x.Rdata)
		mapStr["data"] = rData
		if err != nil {
			debugf("%s", err.Error())
		}
	case *mkdns.RRSIG:
		mapStr["type_covered"] = dnsTypeToString(x.TypeCovered)
		mapStr["algorithm"] = dnsAlgorithmToString(x.Algorithm)
		mapStr["labels"] = strconv.Itoa(int(x.Labels))
		mapStr["original_ttl"] = strconv.FormatInt(int64(x.OrigTtl), 10)
		mapStr["expiration"] = mkdns.TimeToString(x.Expiration)
		mapStr["inception"] = mkdns.TimeToString(x.Inception)
		mapStr["key_tag"] = strconv.Itoa(int(x.KeyTag))
		mapStr["signer_name"] = x.SignerName
		mapStr["data"] = x.Signature
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

	return mapStr
}

// dnsQuestionToString converts a Question to a string.
func dnsQuestionToString(q mkdns.Question) string {
	name := q.Name

	return fmt.Sprintf("class %s, type %s, %s", dnsClassToString(q.Qclass),
		dnsTypeToString(q.Qtype), name)
}

// rrsToString converts an array of RR's to a
// string.
func rrsToString(r []mkdns.RR) string {
	var rrStrs []string
	for _, rr := range r {
		rrStrs = append(rrStrs, rrToString(rr))
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
			rrsToString(dns.Answer)))
	}

	if len(dns.Ns) > 0 {
		a = append(a, fmt.Sprintf("AUTHORITY %s",
			rrsToString(dns.Ns)))
	}

	if len(dns.Extra) > 0 {
		a = append(a, fmt.Sprintf("ADDITIONAL %s",
			rrsToString(dns.Extra)))
	}

	return strings.Join(a, "; ")
}

// decodeDnsData decodes a byte array into a DNS struct. If an error occurs
// then the returned dns pointer will be nil. This method recovers from panics
// and is concurrency-safe.
// We do not handle Unpack ErrTruncated for now. See https://github.com/miekg/dns/pull/281
func decodeDNSData(transport Transport, rawData []byte) (dns *mkdns.Msg, err error) {
	var offset int
	if transport == TransportTCP {
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
		return nil, NonDNSMsg
	}
	return msg, nil
}
