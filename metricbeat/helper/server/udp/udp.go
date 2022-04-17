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

package udp

import (
	"fmt"
	"net"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/helper/server"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

type UdpServer struct {
	udpaddr           *net.UDPAddr
	listener          *net.UDPConn
	receiveBufferSize int
	done              chan struct{}
	eventQueue        chan server.Event
}

type UdpEvent struct {
	event common.MapStr
	meta  server.Meta
}

func (u *UdpEvent) GetEvent() common.MapStr {
	return u.event
}

func (u *UdpEvent) GetMeta() server.Meta {
	return u.meta
}

func NewUdpServer(base mb.BaseMetricSet) (server.Server, error) {
	config := defaultUdpConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.Host, config.Port))

	if err != nil {
		return nil, err
	}

	return &UdpServer{
		udpaddr:           addr,
		receiveBufferSize: config.ReceiveBufferSize,
		done:              make(chan struct{}),
		eventQueue:        make(chan server.Event),
	}, nil
}

func (g *UdpServer) GetHost() string {
	return g.udpaddr.String()
}

func (g *UdpServer) Start() error {
	listener, err := net.ListenUDP("udp", g.udpaddr)
	if err != nil {
		return errors.Wrap(err, "failed to start UDP server")
	}

	logp.Info("Started listening for UDP on: %s", g.udpaddr.String())
	g.listener = listener

	go g.watchMetrics()
	return nil
}

func (g *UdpServer) watchMetrics() {
	buffer := make([]byte, g.receiveBufferSize)
	for {
		select {
		case <-g.done:
			return
		default:
		}

		length, addr, err := g.listener.ReadFromUDP(buffer)
		if err != nil {
			logp.Err("Error reading from buffer: %v", err.Error())
			continue
		}

		bufCopy := make([]byte, length)
		copy(bufCopy, buffer)

		g.eventQueue <- &UdpEvent{
			event: common.MapStr{
				server.EventDataKey: bufCopy,
			},
			meta: server.Meta{
				"client_ip": addr.IP.String(),
			},
		}
	}
}

func (g *UdpServer) GetEvents() chan server.Event {
	return g.eventQueue
}

func (g *UdpServer) Stop() {
	close(g.done)
	g.listener.Close()
	close(g.eventQueue)
}
