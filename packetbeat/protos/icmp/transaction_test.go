// +build !integration

package icmp

import (
	"testing"
	"time"

	"github.com/tsg/gopacket/layers"

	"github.com/stretchr/testify/assert"
)

func TestIcmpTransactionHasErrorICMPv4(t *testing.T) {
	tuple := icmpTuple{icmpVersion: 4}

	trans1 := icmpTransaction{tuple: tuple, request: nil}
	assert.True(t, trans1.HasError(), "request missing")

	trans2 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv4TypeDestinationUnreachable}}
	assert.True(t, trans2.HasError(), "response with error type")

	trans3 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv4TypeEchoReply}}
	assert.False(t, trans3.HasError(), "response with non-error type")

	trans4 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv4TypeEchoRequest}, response: nil}
	assert.True(t, trans4.HasError(), "transactional request without response")

	trans5 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv4TypeRedirect}, response: nil}
	assert.False(t, trans5.HasError(), "non-transactional request without response")
}

func TestIcmpTransactionHasErrorICMPv6(t *testing.T) {
	tuple := icmpTuple{icmpVersion: 6}

	trans1 := icmpTransaction{tuple: tuple, request: nil}
	assert.True(t, trans1.HasError(), "request missing")

	trans2 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv6TypeDestinationUnreachable}}
	assert.True(t, trans2.HasError(), "response with error type")

	trans3 := icmpTransaction{tuple: tuple, request: &icmpMessage{}, response: &icmpMessage{Type: layers.ICMPv6TypeEchoReply}}
	assert.False(t, trans3.HasError(), "response with non-error type")

	trans4 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv6TypeEchoRequest}, response: nil}
	assert.True(t, trans4.HasError(), "transactional request without response")

	trans5 := icmpTransaction{tuple: tuple, request: &icmpMessage{Type: layers.ICMPv6TypeRedirect}, response: nil}
	assert.False(t, trans5.HasError(), "non-transactional request without response")
}

func TestIcmpTransactionResponseTimeMillis(t *testing.T) {
	reqTime := time.Now()
	resTime := reqTime.Add(time.Duration(1) * time.Second)

	trans1 := icmpTransaction{request: &icmpMessage{ts: reqTime}, response: &icmpMessage{ts: resTime}}
	time1, hasTime1 := trans1.ResponseTimeMillis()
	assert.Equal(t, int32(1000), time1, "request with response")
	assert.True(t, hasTime1, "request with response")

	trans2 := icmpTransaction{request: &icmpMessage{ts: reqTime}}
	time2, hasTime2 := trans2.ResponseTimeMillis()
	assert.Equal(t, int32(0), time2, "request without response")
	assert.False(t, hasTime2, "request without response")

	trans3 := icmpTransaction{response: &icmpMessage{ts: resTime}}
	time3, hasTime3 := trans3.ResponseTimeMillis()
	assert.Equal(t, int32(0), time3, "response without request")
	assert.False(t, hasTime3, "response without request")
}
