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

package pipe

import (
	"context"
	"errors"
	"net"
	"sync"
)

// errListenerClosed is the error returned by the Accept
// and DialContext methods of a closed listener.
var errListenerClosed = errors.New("listener is closed")

// Listener is a net.Listener that uses net.Pipe
// It is only relevant for the APM Server instrumentation of itself
type Listener struct {
	conns     chan net.Conn
	closeOnce sync.Once
	closed    chan struct{}
}

// NewListener returns a new Listener.
func NewListener() *Listener {
	l := &Listener{
		conns:  make(chan net.Conn),
		closed: make(chan struct{}),
	}
	return l
}

// Close closes the listener.
// This is part of the net.Listener interface.
func (l *Listener) Close() error {
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

// Addr returns the listener's network address.
// This is part of the net.Listener interface.
//
// The returned address's network and value are always both
// "pipe", the same as the addresses returned by net.Pipe
// connections.
func (l *Listener) Addr() net.Addr {
	return pipeAddr{}
}

// Accept waits for and returns the next connection to the listener.
// This is part of the net.Listener address.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, errListenerClosed
	case conn := <-l.conns:
		return conn, nil
	}
}

// DialContext dials a connection to the listener, blocking until
// a paired Accept call is made, the listener is closed, or the
// context is canceled/expired.
func (l *Listener) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case <-l.closed:
		client.Close()
		server.Close()
		return nil, errListenerClosed
	case <-ctx.Done():
		client.Close()
		server.Close()
		return nil, ctx.Err()
	case l.conns <- server:
		return client, nil
	}
}

type pipeAddr struct{}

func (pipeAddr) Network() string {
	return "pipe"
}

func (pipeAddr) String() string {
	return "pipe"
}
