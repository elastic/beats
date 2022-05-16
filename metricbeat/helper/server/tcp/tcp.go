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
	"bufio"
	"fmt"
	"net"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper/server"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type TcpServer struct {
	tcpAddr           *net.TCPAddr
	listener          *net.TCPListener
	receiveBufferSize int
	done              chan struct{}
	eventQueue        chan server.Event
	delimiter         byte
}

type TcpEvent struct {
	event mapstr.M
}

func (m *TcpEvent) GetEvent() mapstr.M {
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
		delimiter:         byte(config.Delimiter[0]),
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

		go g.handle(conn)
	}
}

func (g *TcpServer) handle(conn net.Conn) {
	if conn == nil {
		return
	}
	logp.Debug("tcp", "Handling new connection...")

	// Close connection when this function ends
	defer conn.Close()

	// Get a new reader with buffer size as the same as receiveBufferSize
	bufReader := bufio.NewReaderSize(conn, g.receiveBufferSize)

	for {
		// Read tokens delimited by delimiter
		bytes, err := bufReader.ReadBytes(g.delimiter)
		if err != nil {
			logp.Debug("tcp", "unable to read bytes due to error: %v", err)
			return
		}

		// Truncate to max buffer size if too big of a payload
		if len(bytes) > g.receiveBufferSize {
			bytes = bytes[:g.receiveBufferSize]
		}

		// Drop the delimiter and send the data
		if len(bytes) > 0 {
			g.eventQueue <- &TcpEvent{
				event: mapstr.M{
					server.EventDataKey: bytes[:len(bytes)-1],
				},
			}
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
