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
	"fmt"
	"net"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/mb"
)

type TcpServer struct {
	tcpAddr           *net.TCPAddr
	listener          *net.TCPListener
	receiveBufferSize int
	done              chan struct{}
	eventQueue        chan server.Event
}

type TcpEvent struct {
	event common.MapStr
}

func (m *TcpEvent) GetEvent() common.MapStr {
	return m.event
}

func (m *TcpEvent) GetMeta() server.Meta {
	return server.Meta{}
}

func NewTcpServer(base mb.BaseMetricSet) (server.Server, error) {
	config := defaultTcpConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))

	if err != nil {
		return nil, err
	}

	return &TcpServer{
		tcpAddr:           addr,
		receiveBufferSize: config.ReceiveBufferSize,
		done:              make(chan struct{}),
		eventQueue:        make(chan server.Event),
	}, nil
}

func (g *TcpServer) Start() error {
	listener, err := net.ListenTCP("tcp", g.tcpAddr)
	if err != nil {
		return errors.Wrap(err, "failed to start TCP server")
	}
	g.listener = listener
	logp.Info("Started listening for TCP on: %s", g.tcpAddr.String())

	go g.watchMetrics()
	return nil
}

func (g *TcpServer) watchMetrics() {
	buffer := make([]byte, g.receiveBufferSize)
	for {
		select {
		case <-g.done:
			return
		default:
		}

		conn, err := g.listener.Accept()
		if err != nil {
			logp.Err("Unable to accept connection due to error: %v", err)
			continue
		}
		defer func() {
			if conn != nil {
				conn.Close()
			}
		}()

		length, err := conn.Read(buffer)
		if err != nil {
			logp.Err("Error reading from buffer: %v", err.Error())
			continue
		}
		g.eventQueue <- &TcpEvent{
			event: common.MapStr{
				server.EventDataKey: buffer[:length],
			},
		}
	}
}

func (g *TcpServer) GetEvents() chan server.Event {
	return g.eventQueue
}

func (g *TcpServer) Stop() {
	close(g.done)
	g.listener.Close()
	close(g.eventQueue)
}
