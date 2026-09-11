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

//go:build !requirefips

package translate_ldap_attribute

import (
	"io"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// newFakeLDAPServer starts a TCP listener that answers the initial LDAP bind
// request on every connection with a canned success response. Together with a
// configured base DN this is enough for newLDAPClient to succeed without a
// real LDAP server.
func newFakeLDAPServer(t *testing.T) net.Listener {
	t.Helper()

	var lc net.ListenConfig
	l, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	// BER-encoded LDAP BindResponse: messageID=1, resultCode=0 (success).
	bindSuccess := []byte{0x30, 0x0c, 0x02, 0x01, 0x01, 0x61, 0x07, 0x0a, 0x01, 0x00, 0x04, 0x00, 0x04, 0x00}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				if _, err := c.Read(buf); err != nil {
					return
				}
				if _, err := c.Write(bindSuccess); err != nil {
					return
				}
				// Drain until the client closes the connection.
				_, _ = io.Copy(io.Discard, c)
			}(conn)
		}
	}()

	return l
}

// TestConcurrentClientInitAndString exercises the race between String and the
// lazy client initialization: ensureClient stores the discovered address and
// base DN, while String reads the processor description. String is registered
// on the processor logger via logp.Stringer, so it may be evaluated from any
// goroutine while another initializes the client, because identically
// configured processors are shared between inputs. Only meaningful under -race.
func TestConcurrentClientInitAndString(t *testing.T) {
	l := newFakeLDAPServer(t)

	c := defaultConfig()
	c.Field = "guid"
	c.LDAPAddress = "ldap://" + l.Addr().String()
	c.LDAPBaseDN = "dc=example,dc=com"
	c.LDAPBindUser = "cn=admin,dc=example,dc=com"
	c.LDAPBindPassword = "password"

	p, err := newFromConfig(c, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	defer p.Close()

	done := make(chan struct{})
	var readers sync.WaitGroup
	for i := 0; i < 2; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = p.String()
				}
			}
		}()
	}

	var writers sync.WaitGroup
	for i := 0; i < 4; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for j := 0; j < 25; j++ {
				_, _ = p.ensureClient()
				if j%5 == 4 {
					// Drop the client so the next ensureClient
					// reinitializes it and writes the discovered values again.
					_ = p.Close()
				}
			}
		}()
	}

	writers.Wait()
	close(done)
	readers.Wait()
}
