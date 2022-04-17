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

//go:build !integration
// +build !integration

package udp

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/helper/server"
)

func GetTestUdpServer(host string, port int) (server.Server, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))

	if err != nil {
		return nil, err
	}

	logp.Info("Started listening for UDP on: %s:%d", host, port)
	return &UdpServer{
		udpaddr:           addr,
		receiveBufferSize: 1024,
		done:              make(chan struct{}),
		eventQueue:        make(chan server.Event),
	}, nil
}

func TestUdpServer(t *testing.T) {
	host := "127.0.0.1"
	port := 2003
	svc, err := GetTestUdpServer(host, port)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	svc.Start()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	defer svc.Stop()
	writeToServer(t, "test1", host, port)
	msg := <-svc.GetEvents()

	assert.True(t, msg.GetEvent() != nil)
	ok, _ := msg.GetEvent().HasKey("data")
	assert.True(t, ok)
	bytes, _ := msg.GetEvent()["data"].([]byte)
	assert.True(t, string(bytes) == "test1")
}

func writeToServer(t *testing.T, message, host string, port int) {
	servAddr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("udp", servAddr)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	defer conn.Close()
	_, err = conn.Write([]byte(message))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
