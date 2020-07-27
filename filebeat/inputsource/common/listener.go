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

package common

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
)

// Family represents the type of connection we're handling
type Family string

const (
	// FamilyUnix represents a unix socket listener
	FamilyUnix Family = "unix"
	// FamilyTCP represents a tcp socket listener
	FamilyTCP Family = "tcp"
)

func (f Family) String() string {
	return strings.ToUpper(string(f))
}

// ListenerFactory returns a net.Listener
type ListenerFactory func() (net.Listener, error)

// Listener represent a generic connected server
type Listener struct {
	Listener        net.Listener
	config          *ListenerConfig
	family          Family
	wg              sync.WaitGroup
	log             *logp.Logger
	ctx             context.Context
	cancel          context.CancelFunc
	clientsCount    atomic.Int
	handlerFactory  HandlerFactory
	listenerFactory ListenerFactory
}

// NewListener creates a new Listener
func NewListener(family Family, location string, handlerFactory HandlerFactory, listenerFactory ListenerFactory, config *ListenerConfig) *Listener {
	return &Listener{
		config:          config,
		family:          family,
		log:             logp.NewLogger(string(family)).With("address", location),
		handlerFactory:  handlerFactory,
		listenerFactory: listenerFactory,
	}
}

// Start listen to the socket.
func (l *Listener) Start() error {
	var err error
	l.Listener, err = l.listenerFactory()
	if err != nil {
		return err
	}

	l.ctx, l.cancel = context.WithCancel(context.Background())
	go func() {
		<-l.ctx.Done()
		l.Listener.Close()
	}()

	l.log.Info("Started listening for " + l.family.String() + " connection")

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.run()
	}()
	return nil
}

// Run start and run a new TCP listener to receive new data. When a new connection is accepted, the factory is used
// to create a ConnectionHandler. The ConnectionHandler takes the connection as input and handles the data that is
// being received via tha io.Reader. Most clients use the splitHandler which can take a bufio.SplitFunc and parse
// out each message into an appropriate event. The Close() of the ConnectionHandler can be used to clean up the
// connection either by client or server based on need.
func (l *Listener) run() {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			select {
			case <-l.ctx.Done():
				return
			default:
				l.log.Debugw("Can not accept the connection", "error", err)
				continue
			}
		}

		l.wg.Add(1)
		go func() {
			defer logp.Recover("recovering from a " + l.family.String() + " client crash")
			defer l.wg.Done()

			ctx, cancel := ctxtool.WithFunc(l.ctx, func() { conn.Close() })
			defer cancel()

			l.registerHandler()
			defer l.unregisterHandler()

			if l.family == FamilyUnix {
				// unix sockets have an empty `RemoteAddr` value, so no need to capture it
				l.log.Debugw("New client", "total", l.clientsCount.Load())
			} else {
				l.log.Debugw("New client", "remote_address", conn.RemoteAddr(), "total", l.clientsCount.Load())
			}

			handler := l.handlerFactory(*l.config)
			err := handler(ctx, conn)
			if err != nil {
				l.log.Debugw("client error", "error", err)
			}

			defer func() {
				if l.family == FamilyUnix {
					// unix sockets have an empty `RemoteAddr` value, so no need to capture it
					l.log.Debugw("client disconnected", "total", l.clientsCount.Load())
				} else {
					l.log.Debugw("client disconnected", "remote_address", conn.RemoteAddr(), "total", l.clientsCount.Load())
				}
			}()
		}()
	}
}

// Stop stops accepting new incoming connections and Close any active clients
func (l *Listener) Stop() {
	l.log.Info("Stopping" + l.family.String() + "server")
	l.cancel()
	l.wg.Wait()
	l.log.Info(l.family.String() + " server stopped")
}

func (l *Listener) registerHandler() {
	l.clientsCount.Inc()
}

func (l *Listener) unregisterHandler() {
	l.clientsCount.Dec()
}

// SplitFunc allows to create a `bufio.SplitFunc` based on a delimiter provided.
func SplitFunc(lineDelimiter []byte) bufio.SplitFunc {
	ld := []byte(lineDelimiter)
	if bytes.Equal(ld, []byte("\n")) {
		// This will work for most usecases and will also strip \r if present.
		// CustomDelimiter, need to match completely and the delimiter will be completely removed from
		// the returned byte slice
		return bufio.ScanLines
	}
	return FactoryDelimiter(ld)
}
