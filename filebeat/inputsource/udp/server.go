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
	"net"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/common/dgram"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Name is the human readable name and identifier.
const Name = "udp"

// Server creates a simple UDP Server and listen to a specific host:port and will send any
// event received to the callback method.
type Server struct {
	*dgram.Listener
	config *Config

	localaddress string
}

// New returns a new UDPServer instance.
func New(config *Config, callback inputsource.NetworkFunc) *Server {
	server := &Server{config: config}
	log := logp.NewLogger("udp").With("address", config.Host)
	factory := dgram.DatagramReaderFactory(inputsource.FamilyUDP, log, callback)
	server.Listener = dgram.NewListener(inputsource.FamilyUDP, config.Host, factory, server.createConn, &dgram.ListenerConfig{
		Timeout:        config.Timeout,
		MaxMessageSize: config.MaxMessageSize,
	})
	return server
}

func (u *Server) createConn() (net.PacketConn, error) {
	var err error
	udpAdddr, err := net.ResolveUDPAddr("udp", u.config.Host)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenUDP("udp", udpAdddr)
	if err != nil {
		return nil, err
	}
	socketSize := int(u.config.ReadBuffer) * humanize.KiByte
	if socketSize != 0 {
		if err := listener.SetReadBuffer(int(u.config.ReadBuffer)); err != nil {
			return nil, err
		}
	}
	u.localaddress = listener.LocalAddr().String()

	return listener, err
}
