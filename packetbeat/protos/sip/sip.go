package sip

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/beats/packetbeat/protos"
)

var (
	debugf = logp.MakeDebug("sip")
)

// Packetbeats monitoring metrics
var (
	messageIgnored = monitoring.NewInt(nil, "sip.message_ignored")
)

const (
	transportTCP = 0
	transportUDP = 1
)

// MessegeStatus
const (
	SipStatusReceived        = 0
	SipStatusHeaderReceiving = 1
	SipStatusBodyReceiving   = 2
	SipStatusRejected        = 3
)

// Detail parse mode
const (
	SipDetailURI          = 1 // ex. sip:bob@example.com
	SipDetailNameAddr     = 2 // ex. "Bob"<sip:bob@example.com>
	SipDetailInt          = 3 // ex. 123
	SipDetailIntMethod    = 4 // ex. 123 INVITE
	SipDetailIntIntMethod = 5 // ex. 123 123 INVITE
	SipDetailIntString    = 6 // ex. 123 INVITE
	SipDetailIntInt       = 7 // ex. 123 123
	SipDetailIntIntString = 8 // ex. 123 123 INVITE
)

func init() {
	// Memo: Secound argment*New* is below New function.
	protos.Register("sip", New)
}

// New create a sip plugin
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &sipPlugin{}
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

func getLastElementStrArray(array []common.NetString) common.NetString {
	return array[len(array)-1]
}

/**
 ******************************************************************
 * transport
 *******************************************************************
 **/

// Transport protocol.
// transport=0 tcp, transport=1, udp
type transport uint8

func (t transport) String() string {

	transportNames := []string{
		"tcp",
		"udp",
	}

	if int(t) >= len(transportNames) {
		return "impossible"
	}
	return transportNames[t]
}
