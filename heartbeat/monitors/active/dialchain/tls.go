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

package dialchain

import (
	cryptoTLS "crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// TLSLayer configures the TLS layer in a DialerChain.
//
// The layer will update the active event with:
//
//  {
//    "tls": {
//        "rtt": { "handshake": { "us": ... }}
//    }
//  }
func TLSLayer(cfg *transport.TLSConfig, to time.Duration) Layer {
	return func(event *beat.Event, next transport.Dialer) (transport.Dialer, error) {
		var timer timer

		// Wrap next dialer so to start the timer when 'next' returns.
		// This gets us the timestamp for when the TLS layer will start the handshake.
		next = startTimerAfterDial(&timer, next)

		dialer, err := transport.TLSDialer(next, cfg, to)
		if err != nil {
			return nil, err
		}

		return afterDial(dialer, func(conn net.Conn) (net.Conn, error) {
			tlsConn, ok := conn.(*cryptoTLS.Conn)
			if !ok {
				panic(fmt.Sprintf("TLS afterDial received a non-tls connection %t. This should never happen", conn))
			}

			// TODO: extract TLS connection parameters from connection object.
			timer.stop()
			event.PutValue("tls.rtt.handshake", look.RTT(timer.duration()))

			addCertMetdata(event.Fields, tlsConn.ConnectionState().PeerCertificates)

			return conn, nil
		}), nil
	}
}

func addCertMetdata(fields common.MapStr, certs []*x509.Certificate) {
	// The behavior here might seem strange. We *always* set a notBefore, but only optionally set a notAfter.
	// Why might we do this?
	// The root cause is that the x509.Certificate type uses time.Time for these fields instead of *time.Time
	// so we have no way to know if the user actually set these fields. The x509 RFC says that only one of the
	// two fields must be set. Most tools (including openssl and go's certgen) always set both. BECAUSE WHY NOT
	//
	// In the wild, however, there are certs missing one of these two fields.
	// So, what's the correct behavior here? We cannot know if a field was omitted due to the lack of nullability.
	// So, in this case, we try to do what people will want 99.99999999999999999% of the time.
	// People might set notBefore to go's zero date intentionally when creating certs. So, we always set that
	// field, even if we find a zero value.
	// However, it would be weird to set notAfter to the zero value. That could invalidate a cert that was intended
	// to be valid forever. So, in that case, we treat the zero value as non-existent.
	// This is why notBefore is a time.Time and notAfter is a *time.Time
	var chainNotValidBefore time.Time
	var chainNotValidAfter *time.Time

	// We need the zero date later
	var zeroTime time.Time

	// Here we compute the minimal bounds during which this certificate chain is valid
	// To do this correctly, we take the maximum NotBefore and the minimum NotAfter.
	// This *should* always wind up being the terminal cert in the chain, but we should
	// compute this correctly.
	for _, cert := range certs {
		if chainNotValidBefore.Before(cert.NotBefore) {
			chainNotValidBefore = cert.NotBefore
		}

		if cert.NotAfter != zeroTime && (chainNotValidAfter == nil || chainNotValidAfter.After(cert.NotAfter)) {
			chainNotValidAfter = &cert.NotAfter
		}
	}

	fields.Put("tls.certificate_not_valid_before", chainNotValidBefore)

	if chainNotValidAfter != nil {
		fields.Put("tls.certificate_not_valid_after", *chainNotValidAfter)
	}
}
