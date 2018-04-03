package sip

import (
    "net"
    "testing"
    "time"
    "fmt"

    "github.com/stretchr/testify/assert"

    "github.com/elastic/beats/libbeat/beat"
    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/logp"
    "github.com/elastic/beats/packetbeat/protos"
)


func TestInit(t *testing.T) {
    // TODO: Is it need test implementation?
}

func TestSetFromConfig(t *testing.T) {
    sip:=sipPlugin{}
    cfg:=sipConfig{}
    cfg.Ports=[]int{5060,5061}

    sip.setFromConfig(&cfg)
    assert.Equal(t, 5060 , sip.ports[0]                   , "There should be included 5060." )
    assert.Equal(t, 5061 , sip.ports[1]                   , "There should be included 5061." )
}

func TestGetPorts(t *testing.T) {
    sip:=sipPlugin{}
    sip.ports=[]int{5060,5061,1123,5555}
    ports:=sip.GetPorts()

    assert.Equal(t, 5060 , ports[0]                   , "There should be included 5060." )
    assert.Equal(t, 5061 , ports[1]                   , "There should be included 5061." )
    assert.Equal(t, 1123 , ports[2]                   , "There should be included 5061." )
    assert.Equal(t, 5555 , ports[3]                   , "There should be included 5061." )
}

func TestPublishMessage(t *testing.T) {
    sip:=sipPlugin{}

    raw_text   :="test raw string"
    method_text:="INVITE"
    phrase_text:="OK"
    ipTuple := common.NewIPPortTuple(4,
        net.ParseIP("10.0.0.1"), 1111,
        net.ParseIP("10.0.0.2"), 2222)
    msg:=sipMessage{transport:0, raw:common.NetString(raw_text),
                    tuple: ipTuple , method: common.NetString(method_text),
                    requestUri: common.NetString("sip:test"),
                    statusCode: uint16(200), statusPhrase: common.NetString(phrase_text),
                    from: common.NetString("from"), to: common.NetString("to"),
                    cseq: common.NetString("cseq"), callid: common.NetString("callid"),
                    contentlength: 10}

    // avoid to sip.results initialization error
    sip.publishMessage(&msg)
    assert.Nil(t, sip.results , "sip.results should still nil." )

    store:= &eventStore{}

    sip.results=store.publish
    sip.publishMessage(&msg)
    assert.Equal(t, 1, store.size() , "There should be added one packet in store after publish." )
    assert.Equal(t, phrase_text, store.events[0].Fields["sip.status-phrase"], "Compare published packet and stored data." )
    assert.Equal(t, nil        , store.events[0].Fields["sip.method"]       , "Compare published packet and stored data." )
    assert.Equal(t, raw_text   , store.events[0].Fields["sip.raw"]          , "Compare published packet and stored data." )
}

func TestCreateSIPMessage(t *testing.T) {
    sip:=sipPlugin{}
    var trans transport
    trans=transportTCP
    garbage := []byte( "Go is an open source programming language "   +
                       "that makes it easy to build simple, reliable, "+
                       "and efficient software.")
    sipMsg,err:=sip.createSIPMessage(trans,garbage)

    assert.Nil(t, err, "Should be no errors." )
    assert.Equal(t, trans, sipMsg.transport, "Compare transport value." )
    assert.Equal(t, garbage, sipMsg.raw, "Compare packet raw message." )
    assert.Equal(t, -1, sipMsg.hdr_start    , "Initialization check." )
    assert.Equal(t, -1, sipMsg.hdr_len      , "Initialization check." )
    assert.Equal(t, -1, sipMsg.bdy_start    , "Initialization check." )
    assert.Equal(t, -1, sipMsg.contentlength, "Initialization check." )
}


// Test Cases migrated from sip_test.go 2018-03-03
// Test Constants
const (
    serverIP   = "192.168.0.1"
    serverPort = 5060
    clientIP   = "10.0.0.1"
    clientPort = 5060
)

// Request and response addresses.
var (
    forward = common.NewIPPortTuple(4,
        net.ParseIP(serverIP), serverPort,
        net.ParseIP(clientIP), clientPort)
    reverse = common.NewIPPortTuple(4,
        net.ParseIP(clientIP), clientPort,
        net.ParseIP(serverIP), serverPort)
)

type eventStore struct {
    events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
    e.events = append(e.events, event)
}

func (e *eventStore) empty() bool {
    return len(e.events) == 0
}

func (e *eventStore) size() int {
    return len(e.events)
}

func newSIP(store *eventStore, verbose bool) *sipPlugin {
    level := logp.WarnLevel
    if verbose {
        level = logp.DebugLevel
    }
    logp.DevelopmentSetup(
        logp.WithLevel(level),
        logp.WithSelectors("sip"),
    )

    callback := func(beat.Event) {}
    if store != nil {
        callback = store.publish
    }

    cfg, _ := common.NewConfigFrom(map[string]interface{}{
        "ports":               []int{serverPort},
    })
    sip, err := New(false, callback, cfg)
    if err != nil {
        panic(err)
    }

    return sip.(*sipPlugin)
}

func newPacket(t common.IPPortTuple, payload []byte) *protos.Packet {
    return &protos.Packet{
        Ts:      time.Now(),
        Tuple:   t,
        Payload: payload,
    }
}

// Verify that an empty packet is safely handled (no panics).
func TestParseUdp_emptyPacket(t *testing.T) {
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    packet := newPacket(forward, []byte{})
    sip.ParseUDP(packet)

    assert.Equal(t, 0, store.size(), "There should be one message published.")
}

// Verify that a malformed packet is safely handled (no panics).
func TestParseUdp_malformedPacket(t *testing.T) {
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    garbage := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)

    assert.Equal(t, 0, store.size(), "There should be one message published.")
}

func TestParseUdp_requestPacketWithSDP(t *testing.T){
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    garbage := []byte( "INVITE sip:0312345678@192.168.0.1;user=phone SIP/2.0\r\n"   +
                       "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK81075720\r\n"  +
                       "From: <sip:sipurl@192.168.0.1>;tag=269050131\r\n"           +
                       "To: <sip:0312341234@192.168.0.1;user=phone>\r\n"            +
                       "Contact: <sip:301234123@10.0.0.1;user=phone>\r\n"           +
                       "Call-ID: hogehoge@192.168.0.1\r\n"                          +
                       "CSeq: 1 INVITE\r\n"                                         +
                       "Max-Forwards: 70\r\n"                                       +
                       "Allow: INVITE, ACK, CANCEL, BYE, UPDATE, PRACK\r\n"         +
                       "Supported: 100rel,timer\r\n"                                +
                       "Session-Expires: 300\r\n"                                   +
                       "Privacy: none\r\n"                                          +
                       "P-Preferred-Identity: <tel:0387654321>\r\n"                 +
                       "Content-Type: application/sdp\r\n"                          +
                       "Content-Length: 107\r\n"                                    +
                       "\r\n"                                                       +
                       "v=0\r\n"                                                    +
                       "o=- 0 0 IN IP4 10.0.0.1\r\n"                                +
                       "s=-\r\n"                                                    +
                       "c=IN IP4 10.0.0.1\r\n"                                      +
                       "t=0 0\r\n"                                                  +
                       "m=audio 5012 RTP/AVP 0\r\n"                                 +
                       "a=rtpmap:0 PCMU/8000\r\n")
    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)
    assert.Equal(t, 1, store.size(), "There should be one message published.")
}

func TestParseUdp_requestPacketWithoutSDP(t *testing.T){
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    garbage := []byte(  "ACK sip:0312345678@192.168.0.1:5060 SIP/2.0\r\n"               +
                        "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK3408987398\r\n"   +
                        "From: <sip:hogehoge@example.com>;tag=5408647717\r\n"           +
                        "To: <sip:0312345678@192.168.0.1>;tag=3713480994\r\n"           +
                        "Call-ID: hogehoge@10.0.0.1\r\n"                                +
                        "CSeq: 1 ACK\r\n"                                               +
                        "Content-Length: 0\r\n"                                         +
                        "Max-Forwards: 70\r\n"                                          +
                        "\r\n")
    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)
    assert.Equal(t, 1, store.size(), "There should be one message published.")
}

func TestParseUdp_requestPacketBeforeStartCRLF(t *testing.T){
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    garbage := []byte(  "\r\n"                                                          +
                        "\r\n"                                                          +
                        "\r\n"                                                          +
                        "\r\n"                                                          +
                        "\r\n"                                                          +
                        "ACK sip:0312345678@192.168.0.1:5060 SIP/2.0\r\n"               +
                        "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK3408987398\r\n"   +
                        "From: <sip:hogehoge@example.com>;tag=5408647717\r\n"           +
                        "To: <sip:0312345678@192.168.0.1>;tag=3713480994\r\n"           +
                        "Call-ID: hogehoge@10.0.0.1\r\n"                                +
                        "CSeq: 1 ACK\r\n"                                               +
                        "Content-Length: 0\r\n"                                         +
                        "Max-Forwards: 70\r\n"                                          +
                        "\r\n")
    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)
    assert.Equal(t, 1, store.size(), "There should be one message published.")
}


func TestParseUdp_responsePacketWithSDP(t *testing.T){
    store := &eventStore{}
    assert.Equal(t, 0, store.size(), "There should be one message published.")
    sip := newSIP(store, testing.Verbose())
    garbage := []byte( "SIP/2.0 183 Session Progess\r\n"                                                      +
                       "Via: SIP/2.0/UDP cw-aio:5060;rport;branch=z9hG4bKPjsRBrmG2vdijbHibFAGTin3eIn6pWysl1," +
                       " SIP/2.0/TCP 192.168.0.1:5052;rport=55196;received=192.168.0.1;"                      +
                       "branch=z9hG4bKPjzjRVAiigVbR6RMhBFOgNh6BXHP80-aBf,"                                    +
                       " SIP/2.0/TCP 192.168.0.1:5058;rport=34867;received=192.168.0.1;"                      +
                       "branch=z9hG4bKPjkp40B7iQTntn1rf9TuASHKtyhPss8fh5,"                                    +
                       " SIP/2.0/UDP 10.0.0.1:5060;received=10.0.0.1;branch=z9hG4bK-2363-1-0\r\n"             +
                       "From: \"sipp\" <sip:sipp@10.0.0.1>;tag=2363SIPpTag001\r\n"                            +
                       "To: \"sut\" <sip:6505550252@192.168.0.1>;tag=16489SIPpTag012\r\n"                     +
                       "Call-ID: 1-2363@10.0.0.1\r\n"                                                         +
                       "CSeq: 1 INVITE\r\n"                                                                   +
                       "Contact: <sip:192.168.0.1:5060;transport=UDP>\r\n"                                    +
                       "Content-Type: application/sdp\r\n"                                                    +
                       "Content-Length: 114\r\n"                                                              +
                       "\r\n"                                                                                 +
                       "v=0\r\n"                                                                              +
                       "o=- 0 0 IN IP4 192.168.0.1\r\n"                                                       +
                       "s=-\r\n"                                                                              +
                       "c=IN IP4 192.168.0.1\r\n"                                                             +
                       "t=0 0\r\n"                                                                            +
                       "m=audio 65000 RTP/AVP 0\r\n"                                                          +
                       "a=rtpmap:0 PCMU/8000\r\n")

    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)
    assert.Equal(t, 1, store.size(), "There should be one message published.")
    if store.size() == 1{
        fields:=store.events[0].Fields
        headers,_:=fields["sip.headers"].(common.MapStr)
        // mandatories
        assert.Equal(t, "Session Progess",
                        fields["sip.status-phrase"], 
                        "There should be [Session Progress].")

        assert.Equal(t, 183, 
                        fields["sip.status-code"], 
                        "There should be 183.")

        assert.Equal(t, "1-2363@10.0.0.1" ,
                        fields["sip.call-id"], 
                        "There should be [1-2363@10.0.0.1].")

        assert.Equal(t, "\"sipp\" <sip:sipp@10.0.0.1>;tag=2363SIPpTag001",
                        fields["sip.from"], 
                        "There should be [\"sipp\" <sip:sipp@10.0.0.1>;tag=2363SIPpTag001].")

        assert.Equal(t, "\"sut\" <sip:6505550252@192.168.0.1>;tag=16489SIPpTag012",
                        fields["sip.to"],
                        "There should be [\"sut\" <sip:6505550252@192.168.0.1>;tag=16489SIPpTag012].")

        assert.Equal(t, "1 INVITE",
                        fields["sip.cseq"],
                        "There should be [1 INVITE].")
        // headers
        assert.Equal(t, "application/sdp",
                        fmt.Sprintf("%s",(headers["content-type"].([]common.NetString))[0]),
                        "There should be [application/sdp].")

        via0:="SIP/2.0/UDP cw-aio:5060;rport;branch=z9hG4bKPjsRBrmG2vdijbHibFAGTin3eIn6pWysl1"
        assert.Equal(t, via0,
                        fmt.Sprintf("%s",(headers["via"].([]common.NetString))[0]),
                        fmt.Sprintf("There should be [%s].",via0))

        via1:="SIP/2.0/TCP 192.168.0.1:5052;rport=55196;received=192.168.0.1;" +
              "branch=z9hG4bKPjzjRVAiigVbR6RMhBFOgNh6BXHP80-aBf"
        assert.Equal(t, via1,
                        fmt.Sprintf("%s",(headers["via"].([]common.NetString))[1]),
                        fmt.Sprintf("There should be [%s].",via1))
via2:="SIP/2.0/TCP 192.168.0.1:5058;rport=34867;received=192.168.0.1;" +
              "branch=z9hG4bKPjkp40B7iQTntn1rf9TuASHKtyhPss8fh5"
        assert.Equal(t, via2,
                        fmt.Sprintf("%s",(headers["via"].([]common.NetString))[2]),
                        fmt.Sprintf("There should be [%s].",via2))

        via3:="SIP/2.0/UDP 10.0.0.1:5060;received=10.0.0.1;branch=z9hG4bK-2363-1-0"
        assert.Equal(t, via3,
                        fmt.Sprintf("%s",(headers["via"].([]common.NetString))[3]),
                        fmt.Sprintf("There should be [%s].",via3))
    }
}

func TestParseUdp_responsePacketWithoutSDP(t *testing.T){
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    garbage := []byte( "SIP/2.0 407 Proxy Authentication Required\r\n"              +
                       "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK3408987398\r\n"+
                       "From: <sip:hogehoge@10.0.0.1>;tag=5408647717\r\n"           +
                       "To: <sip:0312345678@192.168.0.1>;tag=3713480994\r\n"        +
                       "Call-ID: hogehoge@192.168.0.1\r\n"                          +
                       "CSeq: 1 INVITE\r\n"                                         +
                       "Content-Length: 0\r\n"                                      +
                       "Date: Mon, 04 Sep 2017 02:29:54 GMT\r\n"                    +
                       "Proxy-Authenticate: Digest realm=\"example.com\","          +
                       " domain=\"sip:192.168.0.1\", nonce=\"15044921123142536\","  +
                       " opaque=\"\", stale=FALSE, algorithm=MD5\r\n"               +
                       "\r\n")
    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)

    assert.Equal(t, 1, store.size(), "There should be one message published.")
}

func TestParseUdp_IncompletePacketInBody(t *testing.T) {
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())
    garbage := []byte(  "INVITE sip:0312345678@192.168.0.1:5060 SIP/2.0\r\n"                        +
                         "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK1701109339\r\n"               +
                         "From: <sip:hogehoge@example.cm>;tag=1451088881\r\n"                        +
                         "To: <sip:0312345678@192.168.0.1>\r\n"                                      +
                         "Call-ID: hogehoge@10.0.0.1\r\n"                                            +
                         "CSeq: 2 INVITE\r\n"                                                        +
                         "Contact: <sip:1833176976@10.0.0.1:5060;transport=udp>\r\n"                 +
                         "Supported: 100rel, timer\r\n"                                              +
                         "Allow: INVITE, ACK, CANCEL, BYE, UPDATE, PRACK\r\n"                        +
                         "Content-Length: 134\r\n"                                                   +
                         "Session-Expires: 180\r\n"                                                  +
                         "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK1701109339\r\n"               +
                         "Max-Forwards: 70\r\n"                                                      +
                         "Content-Type: application/sdp\r\n"                                         +
                         "Privacy: none\r\n"                                                         +
                         "P-Preferred-Identity: <sip:hogehoge@example.com>\r\n"                      +
                         "User-Agent: Some User-Agent\r\n"                                           +
                         "Proxy-Authorization: Digest username=\"hogehoge\", realm=\"example.com\"," +
                         " nonce=\"15044921123142536\", uri=\"sip:0312345678@192.168.0.1:5060\","    +
                         " response=\"358a640a266ad4eb3ed82f0746c82dfd\"\r\n"                        +
                         "\r\n"                                                                      +
                         "v=0\r\n" )

    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)
    assert.Equal(t, 1, store.size(), "There should be one message published.")

    fields:=store.events[0].Fields
    notes:=fields["sip.notes"]
    assert.Contains(t, fmt.Sprintf("%s",notes) ,"Incompleted message","There should be contained." )
}

func TestParseUdp_IncompletePacketInHeader(t *testing.T) {
    store := &eventStore{}
    sip := newSIP(store, testing.Verbose())

    garbage := []byte(  "INVITE sip:0312345678@192.168.0.1:5060 SIP/2.0\r\n"                        +
                         "Via: SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK1701109339\r\n"               +
                         "From: <sip:hogehoge@example.cm>;tag=1451088881\r\n"                        +
                         "To: <sip:0312345678@192.168.0.1>\r\n"                                      +
                         "Call-ID: hogehoge@10.0.0.1\r\n"                                            +
                         "CSeq: 2 INVITE\r\n"                                                        +
                         "Contact: <sip:1833176976@10.0.0.1:5060;transport=udp>\r\n"                 +
                         "Supported: 100rel, timer\r\n"                                              +
                         "Allow: INVITE, ACK, CANCEL, BYE, UPDATE, PRACK\r\n"                        +
                         "Content-Length: 134\r\n"                                                   +
                         "Session-Expires: 180\r\n")

    packet := newPacket(forward, garbage)
    sip.ParseUDP(packet)
    assert.Equal(t, 1, store.size(), "There should be one message published.")

    fields:=store.events[0].Fields
    notes:=fields["sip.notes"]
    assert.Contains(t, fmt.Sprintf("%s",notes) ,"Incompleted message","There should be contained." )
}

