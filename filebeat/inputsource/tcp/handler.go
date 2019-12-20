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

// splitHandler is a TCP client that has splitting capabilities.
type splitHandler struct {
	callback       inputsource.NetworkFunc
	done           chan struct{}
	metadata       inputsource.NetworkMetadata
	splitFunc      bufio.SplitFunc
	maxMessageSize uint64
	timeout        time.Duration
}

// HandlerFactory returns a ConnectionHandler func
type HandlerFactory func(config Config) ConnectionHandler

// ConnectionHandler interface provides mechanisms for handling of incoming TCP connections
type ConnectionHandler interface {
	Handle(CloseRef, net.Conn) error
}

// SplitHandlerFactory allows creation of a ConnectionHandler that can do splitting of messages received on a TCP connection.
func SplitHandlerFactory(callback inputsource.NetworkFunc, splitFunc bufio.SplitFunc) HandlerFactory {
	return func(config Config) ConnectionHandler {
		return newSplitHandler(
			callback,
			splitFunc,
			uint64(config.MaxMessageSize),
			config.Timeout,
		)
	}
}

// newSplitHandler allows creation of a TCP client that has splitting capabilities.
func newSplitHandler(
	callback inputsource.NetworkFunc,
	splitFunc bufio.SplitFunc,
	maxReadMessage uint64,
	timeout time.Duration,
) ConnectionHandler {
	client := &splitHandler{
		callback:       callback,
		done:           make(chan struct{}),
		splitFunc:      splitFunc,
		maxMessageSize: maxReadMessage,
		timeout:        timeout,
	}
	return client
}

// Handle takes a connection as input and processes data received on it.
func (c *splitHandler) Handle(closer CloseRef, conn net.Conn) error {
	c.metadata = inputsource.NetworkMetadata{
		RemoteAddr: conn.RemoteAddr(),
		TLS:        extractSSLInformation(conn),
	}

	log := logp.NewLogger("split_client").With("remote_addr", conn.RemoteAddr().String())

	r := NewResetableLimitedReader(NewDeadlineReader(conn, c.timeout), c.maxMessageSize)
	buf := bufio.NewReader(r)
	scanner := bufio.NewScanner(buf)
	scanner.Split(c.splitFunc)
	//16 is ratio of MaxScanTokenSize/startBufSize
	buffer := make([]byte, c.maxMessageSize/16)
	scanner.Buffer(buffer, int(c.maxMessageSize))
	for {
		select {
		case <-closer.Done():
			break
		default:
		}

		// Ensure that if the Conn is already closed then dont attempt to scan again
		if closer.Err() == ErrClosed {
			break
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
			return errors.Wrap(err, "tcp split_client error")
		}
		r.Reset()
		c.callback(scanner.Bytes(), c.metadata)
	}

	// We are out of the scanner, either we reached EOF or another fatal error occurred.
	// like we failed to complete the TLS handshake or we are missing the splitHandler certificate when
	// mutual auth is on, which is the default.
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
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
