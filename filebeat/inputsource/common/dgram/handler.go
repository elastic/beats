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

package dgram

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/elastic-agent-libs/logp"
)

// HandlerFactory returns a ConnectionHandler func
type HandlerFactory func(config ListenerConfig) ConnectionHandler

// ConnectionHandler is able to read from incoming connections.
type ConnectionHandler func(context.Context, net.PacketConn) error

// MetadataFunc defines callback executed when a line is read from the split handler.
type MetadataFunc func(net.Conn) inputsource.NetworkMetadata

// DatagramReaderFactory allows creation of a handler which can read packets from connections.
func DatagramReaderFactory(
	family inputsource.Family,
	logger *logp.Logger,
	callback inputsource.NetworkFunc,
) HandlerFactory {
	return func(config ListenerConfig) ConnectionHandler {
		return ConnectionHandler(func(ctx context.Context, conn net.PacketConn) error {
			for ctx.Err() == nil {

				buffer := make([]byte, config.MaxMessageSize)
				// conn.SetDeadline(time.Now().Add(config.Timeout))

				// If you are using Windows and you are using a fixed buffer and you get a datagram which
				// is bigger than the specified size of the buffer, it will return an `err` and the buffer will
				// contains a subset of the data.
				//
				// On Unix based system, the buffer will be truncated but no error will be returned.
				length, addr, err := conn.ReadFrom(buffer)
				if err != nil {
					if family == inputsource.FamilyUnix {
						fmt.Println("connection handler error", err)
					}
					// don't log any deadline events.
					e, ok := err.(net.Error)
					if ok && e.Timeout() {
						continue
					}

					// Closed network error string will never change in Go 1.X
					// https://github.com/golang/go/issues/4373
					opErr, ok := err.(*net.OpError)
					if ok && strings.Contains(opErr.Err.Error(), "use of closed network connection") {
						logger.Info("Connection has been closed")
						return nil
					}

					logger.Errorf("Error reading from the socket %s", err)

					// On Windows send the current buffer and mark it as truncated.
					// The buffer will have content but length will return 0, addr will be nil.
					if family == inputsource.FamilyUDP && isLargerThanBuffer(err) {
						callback(buffer, inputsource.NetworkMetadata{RemoteAddr: addr, Truncated: true})
						continue
					}
				}

				if length > 0 {
					callback(buffer[:length], inputsource.NetworkMetadata{RemoteAddr: addr})
				}
			}
			fmt.Println("end of connection handling")
			return nil
		})
	}
}

func isLargerThanBuffer(err error) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	return strings.Contains(err.Error(), windowErrBuffer)
}
