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

package tcp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/filebeat/input/inputtest"
	"github.com/menderesk/beats/v7/filebeat/inputsource"
	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestCreateEvent(t *testing.T) {
	hello := "hello world"
	ip := "127.0.0.1"
	parsedIP := net.ParseIP(ip)
	addr := &net.IPAddr{IP: parsedIP, Zone: ""}

	message := []byte(hello)
	mt := inputsource.NetworkMetadata{RemoteAddr: addr}

	event := createEvent(message, mt)

	m, err := event.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, string(message), m)

	from, _ := event.GetValue("log.source.address")
	assert.Equal(t, ip, from)
}

func TestNewInputDone(t *testing.T) {
	config := common.MapStr{
		"host": ":0",
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}
