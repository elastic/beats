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

package tcp

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
)

// Client is a remote client.
type client struct {
	conn           net.Conn
	log            *logp.Logger
	callback       inputsource.NetworkFunc
	done           chan struct{}
	metadata       inputsource.NetworkMetadata
	splitFunc      bufio.SplitFunc
	maxMessageSize uint64
	timeout        time.Duration
}

func newClient(
	conn net.Conn,
	log *logp.Logger,
	callback inputsource.NetworkFunc,
	splitFunc bufio.SplitFunc,
	maxReadMessage uint64,
	timeout time.Duration,
) *client {
	client := &client{
		conn:           conn,
		log:            log.With("remote_address", conn.RemoteAddr()),
		callback:       callback,
		done:           make(chan struct{}),
		splitFunc:      splitFunc,
		maxMessageSize: maxReadMessage,
		timeout:        timeout,
		metadata: inputsource.NetworkMetadata{
			RemoteAddr: conn.RemoteAddr(),
			TLS:        extractSSLInformation(conn),
		},
	}
	extractSSLInformation(conn)
	return client
}

func (c *client) handle() error {
	r := NewResetableLimitedReader(NewDeadlineReader(c.conn, c.timeout), c.maxMessageSize)
	buf := bufio.NewReader(r)
	scanner := bufio.NewScanner(buf)
	scanner.Split(c.splitFunc)

	for scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			// we are forcing a close on the socket, lets ignore any error that could happen.
			select {
			case <-c.done:
				break
			default:
			}
			// This is a user defined limit and we should notify the user.
			if IsMaxReadBufferErr(err) {
				c.log.Errorw("client error", "error", err)
			}
			return errors.Wrap(err, "tcp client error")
		}
		r.Reset()
		c.callback(scanner.Bytes(), c.metadata)
	}

	// We are out of the scanner, either we reached EOF or another fatal error occurred.
	// like we failed to complete the TLS handshake or we are missing the client certificate when
	// mutual auth is on, which is the default.
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (c *client) close() {
	close(c.done)
	c.conn.Close()
}

func extractSSLInformation(c net.Conn) *inputsource.TLSMetadata {
	if tls, ok := c.(*tls.Conn); ok {
		state := tls.ConnectionState()
		return &inputsource.TLSMetadata{
			TLSVersion:       tlscommon.ResolveTLSVersion(state.Version),
			CipherSuite:      tlscommon.ResolveCipherSuite(state.CipherSuite),
			ServerName:       state.ServerName,
			PeerCertificates: extractCertificate(state.PeerCertificates),
		}
	}
	return nil
}

func extractCertificate(certificates []*x509.Certificate) []string {
	strCertificate := make([]string, len(certificates))
	for idx, c := range certificates {
		// Ignore errors here, problematics cert have failed
		//the handshake at this point.
		b, _ := x509.MarshalPKIXPublicKey(c.PublicKey)
		strCertificate[idx] = string(b)
	}
	return strCertificate
}
