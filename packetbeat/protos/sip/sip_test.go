// +build !integration

package sip

import (
    "testing"
    _"fmt"

    "github.com/stretchr/testify/assert"

    "github.com/elastic/beats/libbeat/common"
)

func TestConstTransportNumbers(t *testing.T){
    assert.Equal(t, 0, transportTCP, "Should be fixed magic number.")
    assert.Equal(t, 1, transportUDP, "Should be fixed magic number.")
}

func TestConstSipPacketRecievingStatus(t *testing.T){
    assert.Equal(t, 0, SIP_STATUS_RECEIVED         , "Should be fixed magic number.")
    assert.Equal(t, 1, SIP_STATUS_HEADER_RECEIVING , "Should be fixed magic number.")
    assert.Equal(t, 2, SIP_STATUS_BODY_RECEIVING   , "Should be fixed magic number.")
    assert.Equal(t, 3, SIP_STATUS_REJECTED         , "Should be fixed magic number.")
}

func TestGetLastElementStrArray(t *testing.T){
    var array []common.NetString

    array = append(array,common.NetString("test1"))
    array = append(array,common.NetString("test2"))
    array = append(array,common.NetString("test3"))
    array = append(array,common.NetString("test4"))

    assert.Equal(t, common.NetString("test4"), getLastElementStrArray(array), "Return last element of array")
}

func TestTypeOfTransport(t *testing.T){
    var trans transport

    trans=transportTCP
    assert.Equal(t, "tcp", trans.String(),        "String should be tcp")

    trans=transportUDP
    assert.Equal(t, "udp", trans.String(),        "String should be udp")

    trans=255
    assert.Equal(t, "impossible", trans.String(), "String should be impossible")
}
