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
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/libbeat/logp"
)

// Name is the human readable name and identifier.
const Name = "udp"

const windowErrBuffer = "A message sent on a datagram socket was larger than the internal message" +
	" buffer or some other network limit, or the buffer used to receive a datagram into was smaller" +
	" than the datagram itself."

// Server creates a simple UDP Server and listen to a specific host:port and will send any
// event received to the callback method.
type Server struct {
	config   *Config
	callback inputsource.NetworkFunc
	Listener net.PacketConn
	log      *logp.Logger
	wg       sync.WaitGroup
	done     chan struct{}
}

// New returns a new UDPServer instance.
func New(config *Config, callback inputsource.NetworkFunc) *Server {
	return &Server{
		config:   config,
		callback: callback,
		log:      logp.NewLogger("udp").With("address", config.Host),
		done:     make(chan struct{}),
	}
}

// Start starts the UDP Server and listen to incoming events.
func (u *Server) Start() error {
	var err error
	u.Listener, err = net.ListenPacket("udp", u.config.Host)
	if err != nil {
		return err
	}
	u.log.Info("Started listening for UDP connection")
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		u.run()
	}()
	return nil
}

func (u *Server) run() {
	for {
		select {
		case <-u.done:
			return
		default:
		}

		buffer := make([]byte, u.config.MaxMessageSize)
		u.Listener.SetDeadline(time.Now().Add(u.config.Timeout))

		// If you are using Windows and you are using a fixed buffer and you get a datagram which
		// is bigger than the specified size of the buffer, it will return an `err` and the buffer will
		// contains a subset of the data.
		//
		// On Unix based system, the buffer will be truncated but no error will be returned.
		length, addr, err := u.Listener.ReadFrom(buffer)
		if err != nil {
			// don't log any deadline events.
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				continue
			}

			u.log.Errorw("Error reading from the socket", "error", err)

			// On Windows send the current buffer and mark it as truncated.
			// The buffer will have content but length will return 0, addr will be nil.
			if isLargerThanBuffer(err) {
				u.callback(buffer, inputsource.NetworkMetadata{RemoteAddr: addr, Truncated: true})
				continue
			}
		}

		if length > 0 {
			u.callback(buffer[:length], inputsource.NetworkMetadata{RemoteAddr: addr})
		}
	}
}

// Stop stops the current udp server.
func (u *Server) Stop() {
	u.log.Info("Stopping UDP server")
	u.Listener.Close()
	close(u.done)
	u.wg.Wait()
	u.log.Info("UDP server stopped")
}

func isLargerThanBuffer(err error) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	return strings.Contains(err.Error(), windowErrBuffer)
}
