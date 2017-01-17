package kafka

import (
	"net"
	"time"

	"github.com/elastic/beats/packetbeat/protos/kafka/internal/kafka"
)

type rawMessage struct {
	// packet data
	TS       time.Time
	endpoint endpoint
	payload  []byte

	// list element use by 'transactions' for correlation
	next *rawMessage
}

type requestMessage struct {
	ts       time.Time
	endpoint endpoint
	header   kafka.RequestHeader
	payload  []byte
	size     int
}

type responseMessage struct {
	ts       time.Time
	endpoint endpoint
	header   kafka.ResponseHeader
	payload  []byte
	size     int
}

type endpoint struct {
	IP   net.IP
	Port uint16
}
