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

package streaming

import (
	"bufio"
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// HandlerFactory returns a ConnectionHandler func
type HandlerFactory func(config ListenerConfig) ConnectionHandler

// ConnectionHandler interface provides mechanisms for handling of incoming connections
type ConnectionHandler func(context.Context, net.Conn) error

// MetadataFunc defines callback executed when a line is read from the split handler.
type MetadataFunc func(net.Conn) inputsource.NetworkMetadata

// SplitHandlerFactory allows creation of a handler that has splitting capabilities.
func SplitHandlerFactory(family inputsource.Family, logger *logp.Logger, metadataCallback MetadataFunc, callback inputsource.NetworkFunc, splitFunc bufio.SplitFunc) HandlerFactory {
	return func(config ListenerConfig) ConnectionHandler {
		return ConnectionHandler(func(ctx context.Context, conn net.Conn) error {
			metadata := metadataCallback(conn)
			maxMessageSize := uint64(config.MaxMessageSize)

			var log *logp.Logger
			if family == inputsource.FamilyUnix {
				// unix sockets have an empty `RemoteAddr` value, so no need to capture it
				log = logger.With("handler", "split_client")
			} else {
				log = logger.With("handler", "split_client", "remote_addr", conn.RemoteAddr().String())
			}

			r := NewResetableLimitedReader(NewDeadlineReader(conn, config.Timeout), maxMessageSize)
			buf := bufio.NewReader(r)
			scanner := bufio.NewScanner(buf)
			scanner.Split(splitFunc)
			// 16 is ratio of MaxScanTokenSize/startBufSize
			buffer := make([]byte, maxMessageSize/16)
			scanner.Buffer(buffer, int(maxMessageSize))
			for {
				select {
				case <-ctx.Done():
					break
				default:
				}

				if !scanner.Scan() {
					break
				}

				err := scanner.Err()
				if err != nil {
					// This is a user defined limit and we should notify the user.
					if IsMaxReadBufferErr(err) {
						log.Errorw("split_client error", "error", err)
					}
					return errors.Wrap(err, string(family)+" split_client error")
				}
				r.Reset()
				callback(scanner.Bytes(), metadata)
			}

			// We are out of the scanner, either we reached EOF or another fatal error occurred.
			// like we failed to complete the TLS handshake or we are missing the splitHandler certificate when
			// mutual auth is on, which is the default.
			return scanner.Err()
		})
	}
}
