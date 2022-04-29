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
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
)

// ListenerFactory returns a net.Listener
type ListenerFactory func() (net.Listener, error)

// Listener represent a generic connected server
type Listener struct {
	Listener        net.Listener
	config          *ListenerConfig
	family          inputsource.Family
	wg              sync.WaitGroup
	log             *logp.Logger
	ctx             ctxtool.CancelContext
	clientsCount    atomic.Int
	handlerFactory  HandlerFactory
	listenerFactory ListenerFactory
}

// FramingType are supported framing options for the SplitFunc
type FramingType int

const (
	FramingDelimiter = iota
	FramingRFC6587
)

var (
	framingTypes = map[string]FramingType{
		"delimiter": FramingDelimiter,
		"rfc6587":   FramingRFC6587,
	}

	availableFramingTypesErrFormat string
)

func init() {
	framingTypeNames := make([]string, 0, len(framingTypes))
	for t := range framingTypes {
		framingTypeNames = append(framingTypeNames, t)
	}

	availableFramingTypesErrFormat = fmt.Sprintf("invalid framing type %%q, "+
		"the supported types are [%v]", strings.Join(framingTypeNames, ", "))
}

// Unpack unpacks the FramingType string value.
func (f *FramingType) Unpack(value string) error {
	value = strings.ToLower(value)

	ft, ok := framingTypes[value]
	if !ok {
		return fmt.Errorf(availableFramingTypesErrFormat, value)
	}

	*f = ft
	return nil
}

// NewListener creates a new Listener
func NewListener(family inputsource.Family, location string, handlerFactory HandlerFactory, listenerFactory ListenerFactory, config *ListenerConfig) *Listener {
	return &Listener{
		config:          config,
		family:          family,
		log:             logp.NewLogger(string(family)).With("address", location),
		handlerFactory:  handlerFactory,
		listenerFactory: listenerFactory,
	}
}

// Start listening to the socket and accepting connections. The method is
// non-blocking and starts a goroutine to service the socket. Stop must be
// called to ensure proper cleanup.
func (l *Listener) Start() error {
	if err := l.initListen(context.Background()); err != nil {
		return err
	}

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
func (l *Listener) Run(ctx context.Context) error {
	if err := l.initListen(ctx); err != nil {
		return err
	}

	l.wg.Add(1)
	defer l.wg.Done()
	l.run()
	return nil
}

func (l *Listener) initListen(ctx context.Context) error {
	var err error
	l.Listener, err = l.listenerFactory()
	if err != nil {
		return err
	}

	l.ctx = ctxtool.WrapCancel(ctxtool.WithFunc(ctx, func() {
		l.Listener.Close()
	}))
	return nil
}

func (l *Listener) run() {
	l.log.Debug("Start accepting connections")
	defer func() {
		l.Listener.Close()
		l.log.Debug("Stopped accepting connections")
	}()

	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			if l.ctx.Err() != nil {
				// Shutdown.
				return
			}

			if errors.Is(err, net.ErrClosed) {
				return
			}

			l.log.Debugw("Cannot accept new connection", "error", err)
			continue
		}

		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.handleConnection(conn)
		}()
	}
}

func (l *Listener) handleConnection(conn net.Conn) {
	log := l.log
	if remoteAddr := conn.RemoteAddr().String(); remoteAddr != "" {
		log = log.With("remote_address", remoteAddr)
	}
	defer log.Recover("Panic in connection handler")

	// Ensure accepted connection is closed on return and at shutdown.
	connCtx, cancel := ctxtool.WithFunc(l.ctx, func() {
		conn.Close()
	})
	defer cancel()

	// Track number of clients.
	l.clientsCount.Inc()
	log.Debugw("New client connection.", "active_clients", l.clientsCount.Load())
	defer func() {
		l.clientsCount.Dec()
		log.Debugw("Client disconnected.", "active_clients", l.clientsCount.Load())
	}()

	handler := l.handlerFactory(*l.config)
	if err := handler(connCtx, conn); err != nil {
		log.Debugw("Client error", "error", err)
		return
	}
}

// Stop stops accepting new incoming connections and closes all active clients.
func (l *Listener) Stop() {
	l.log.Debugw("Stopping socket listener. Waiting for active connections to close.", "active_clients", l.clientsCount.Load())
	l.ctx.Cancel()
	l.wg.Wait()
	l.log.Info("Socket listener stopped")
}

// SplitFunc allows to create a `bufio.SplitFunc` based on a framing and
// delimiter provided.
func SplitFunc(framing FramingType, lineDelimiter []byte) (bufio.SplitFunc, error) {
	if len(lineDelimiter) == 0 {
		return nil, fmt.Errorf("line delimiter required")
	}
	switch framing {
	case FramingDelimiter:
		// This will work for most usecases and will also
		// strip \r if present.  CustomDelimiter, need to
		// match completely and the delimiter will be
		// completely removed from the returned byte slice
		if bytes.Equal(lineDelimiter, []byte("\n")) {
			return bufio.ScanLines, nil
		}
		return FactoryDelimiter(lineDelimiter), nil
	case FramingRFC6587:
		return FactoryRFC6587Framing(lineDelimiter), nil
	default:
		return nil, fmt.Errorf("unknown SplitFunc for framing %d and line delimiter %q", framing, lineDelimiter)
	}
}
