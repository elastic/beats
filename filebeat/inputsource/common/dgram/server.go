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
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const windowErrBuffer = "A message sent on a datagram socket was larger than the internal message" +
	" buffer or some other network limit, or the buffer used to receive a datagram into was smaller" +
	" than the datagram itself."

// HandlerFactory returns a ConnectionHandler func
type HandlerFactory func(config ListenerConfig) ConnectionHandler

type ConnectionHandler func(context.Context, net.PacketConn) error

type ListenerFactory func() (net.PacketConn, error)

// MetadataFunc defines callback executed when a line is read from the split handler.
type MetadataFunc func(net.Conn) inputsource.NetworkMetadata

type ListenerConfig struct {
	Timeout        time.Duration
	MaxMessageSize cfgtype.ByteSize
}

type Listener struct {
	log      *logp.Logger
	family   inputsource.Family
	config   *ListenerConfig
	listener ListenerFactory
	connect  HandlerFactory
	tg       unison.TaskGroup
}

func NewListener(
	f inputsource.Family,
	path string,
	connect HandlerFactory,
	listenerFactory ListenerFactory,
	config *ListenerConfig,
) *Listener {
	return &Listener{
		log:      logp.NewLogger(f.String()),
		family:   f,
		config:   config,
		listener: listenerFactory,
		connect:  connect,
		tg:       unison.TaskGroup{},
	}
}

// DatagramReaderFactory allows creation of a handler that has splitting capabilities.
func DatagramReaderFactory(
	family inputsource.Family,
	logger *logp.Logger,
	callback inputsource.NetworkFunc,
) HandlerFactory {
	return func(config ListenerConfig) ConnectionHandler {
		return ConnectionHandler(func(ctx context.Context, conn net.PacketConn) error {
			//metadata := metadataCallback(conn)
			for ctx.Err() == nil {

				buffer := make([]byte, config.MaxMessageSize)
				conn.SetDeadline(time.Now().Add(config.Timeout))

				// If you are using Windows and you are using a fixed buffer and you get a datagram which
				// is bigger than the specified size of the buffer, it will return an `err` and the buffer will
				// contains a subset of the data.
				//
				// On Unix based system, the buffer will be truncated but no error will be returned.
				length, addr, err := conn.ReadFrom(buffer)
				if err != nil {
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

func (l *Listener) Run(ctx context.Context) error {
	l.log.Info("Started listening for " + l.family.String() + " connection")

	for ctx.Err() == nil {
		conn, err := l.listener()
		if err != nil {
			l.log.Debugw("Cannot connect", "error", err)
			continue
		}
		connCtx, connCancel := ctxtool.WithFunc(ctx, func() {
			conn.Close()
		})

		err = l.run(connCtx, conn)
		if err != nil {
			l.log.Debugw("Error while processing input", "error", err)
			connCancel()
			continue
		}
		connCancel()
	}
	return nil
}

func (l *Listener) Start() error {
	l.log.Info("Started listening for " + l.family.String() + " connection")

	conn, err := l.listener()
	if err != nil {
		return err
	}

	l.tg.Go(func(ctx unison.Canceler) error {
		connCtx, connCancel := ctxtool.WithFunc(ctxtool.FromCanceller(ctx), func() {
			conn.Close()
		})
		defer connCancel()

		return l.run(ctxtool.FromCanceller(connCtx), conn)
	})
	return nil
}

func (l *Listener) run(ctx context.Context, conn net.PacketConn) error {
	handler := l.connect(*l.config)
	for ctx.Err() == nil {
		err := handler(ctx, conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Listener) Stop() {
	l.log.Debug("Stopping datagram socket server for " + l.family.String())
	err := l.tg.Stop()
	if err != nil {
		l.log.Errorf("Error while stopping datagram socket server: %v", err)
	}
}
