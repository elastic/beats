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

package transport

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	l := NewPipeListener()
	assert.NotNil(t, l)
	defer l.Close()
	assert.Implements(t, new(net.Listener), l)
}

func TestAddr(t *testing.T) {
	l := NewPipeListener()
	assert.NotNil(t, l)
	defer l.Close()

	addr := l.Addr()
	assert.NotNil(t, addr)
	assert.Equal(t, "pipe", addr.Network())
	assert.Equal(t, "pipe", addr.String())
}

func TestDialAccept(t *testing.T) {
	l := NewPipeListener()
	assert.NotNil(t, l)
	defer l.Close()

	clientCh := make(chan net.Conn, 1)
	go func() {
		defer close(clientCh)
		client, err := l.DialContext(context.Background(), "foo", "bar")
		if assert.NoError(t, err) {
			clientCh <- client
		}
	}()

	server, err := l.Accept()
	assert.NoError(t, err)
	client := <-clientCh
	defer server.Close()
	defer client.Close()

	hello := []byte("hello!")
	go client.Write(hello)
	buf := make([]byte, len(hello))
	_, err = server.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, string(hello), string(buf))
}

func TestAcceptClosed(t *testing.T) {
	l := NewPipeListener()
	assert.NotNil(t, l)
	defer l.Close()

	err := l.Close()
	assert.NoError(t, err)
	_, err = l.Accept()
	assert.Error(t, errListenerClosed, err)
}

func TestDialClosed(t *testing.T) {
	l := NewPipeListener()
	assert.NotNil(t, l)
	defer l.Close()

	err := l.Close()
	assert.NoError(t, err)
	_, err = l.DialContext(context.Background(), "foo", "bar")
	assert.Error(t, errListenerClosed, err)
}

func TestDialContextCanceled(t *testing.T) {
	l := NewPipeListener()
	assert.NotNil(t, l)
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := l.DialContext(ctx, "foo", "bar")
	assert.Error(t, context.Canceled, err)
}
