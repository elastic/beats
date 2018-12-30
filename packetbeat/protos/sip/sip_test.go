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

// +build !integration

package sip

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestConstTransportNumbers(t *testing.T) {
	assert.Equal(t, 0, transportTCP, "Should be fixed magic number.")
	assert.Equal(t, 1, transportUDP, "Should be fixed magic number.")
}

func TestConstSipPacketRecievingStatus(t *testing.T) {
	assert.Equal(t, 0, SipStatusReceived, "Should be fixed magic number.")
	assert.Equal(t, 1, SipStatusHeaderReceiving, "Should be fixed magic number.")
	assert.Equal(t, 2, SipStatusBodyReceiving, "Should be fixed magic number.")
	assert.Equal(t, 3, SipStatusRejected, "Should be fixed magic number.")
}

func TestGetLastElementStrArray(t *testing.T) {
	var array []common.NetString

	array = append(array, common.NetString("test1"))
	array = append(array, common.NetString("test2"))
	array = append(array, common.NetString("test3"))
	array = append(array, common.NetString("test4"))

	assert.Equal(t, common.NetString("test4"), getLastElementStrArray(array), "Return last element of array")
}

func TestTypeOfTransport(t *testing.T) {
	var trans transport

	trans = transportTCP
	assert.Equal(t, "tcp", trans.String(), "String should be tcp")

	trans = transportUDP
	assert.Equal(t, "udp", trans.String(), "String should be udp")

	trans = 255
	assert.Equal(t, "impossible", trans.String(), "String should be impossible")
}
