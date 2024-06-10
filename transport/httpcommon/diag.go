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

package httpcommon

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// DiagRequest returns a diagnostics hook callback that will send the passed requests using a roundtripper generated from the settings and log httptrace events in the returned bytes.
func (settings *HTTPTransportSettings) DiagRequests(reqs []*http.Request, opts ...TransportOption) func() []byte {
	if settings == nil {
		return func() []byte {
			return []byte(`error: nil httpcommon.HTTPTransportSettings`)
		}
	}
	if len(reqs) == 0 {
		return func() []byte {
			return []byte(`error: 0 requests`)
		}
	}
	return func() []byte {
		var b bytes.Buffer
		rt, err := settings.RoundTripper(opts...)
		if err != nil {
			b.WriteString("unable to get roundtripper: " + err.Error())
			return b.Bytes()
		}
		logger := log.New(&b, "", log.LstdFlags|log.Lmicroseconds|log.LUTC)
		if settings.TLS == nil {
			logger.Print("No TLS settings")
		} else {
			logger.Print("TLS settings detected")
		}
		logger.Printf("Proxy disable=%v url=%s", settings.Proxy.Disable, settings.Proxy.URL)

		ct := &httptrace.ClientTrace{
			GetConn: func(hostPort string) {
				logger.Printf("GetConn called for %q", hostPort)
			},
			GotConn: func(connInfo httptrace.GotConnInfo) {
				logger.Printf("GotConn for %q", connInfo.Conn.RemoteAddr())
			},
			GotFirstResponseByte: func() {
				logger.Print("Response started")
			},
			Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
				logger.Printf("Got info response status=%d, headers=%v", code, header)
				return nil
			},
			DNSStart: func(info httptrace.DNSStartInfo) {
				logger.Printf("Starting DNS lookup for %q", info.Host)
			},
			DNSDone: func(info httptrace.DNSDoneInfo) {
				logger.Printf("Done DNS lookup: %+v", info)
			},
			ConnectStart: func(network, addr string) {
				logger.Printf("Connection started to %s:%s", network, addr)
			},
			ConnectDone: func(network, addr string, err error) {
				logger.Printf("Connection to %s:%s done, err: %v", network, addr, err)
			},
			TLSHandshakeStart: func() {
				logger.Print("TLS handshake starting")
			},
			TLSHandshakeDone: func(state tls.ConnectionState, err error) {
				logger.Printf("TLS handshake done. state=%+v err=%v", state, err)
				logger.Printf("Peer certificate count %d", len(state.PeerCertificates))
				for i, crt := range state.PeerCertificates {
					logger.Printf("- Peer Certificate %d\n\t%s", i, tlscommon.CertDiagString(crt))
				}

				logger.Printf("Verified chains count: %d", len(state.VerifiedChains))
				for i, chain := range state.VerifiedChains {
					for j, crt := range chain {
						logger.Printf("- Chain %d certificate %d\n\t%s", i, j, tlscommon.CertDiagString(crt))
					}
				}
			},
			WroteHeaders: func() {
				logger.Printf("Wrote request headers")
			},
			Wait100Continue: func() {
				logger.Printf("Waiting for continue")
			},
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				logger.Printf("Wrote request err=%v", info.Err)
			},
		}
		for i, req := range reqs {
			logger.Printf("Request %d to %s starting", i, req.URL.String())
			req = req.WithContext(httptrace.WithClientTrace(req.Context(), ct))
			if resp, err := rt.RoundTrip(req); err != nil {
				logger.Printf("request %d error: %s", i, diagError(err))
			} else if isGoHTTPResp(resp) {
				resp.Body.Close()
				logger.Printf("request %d error: HTTP request sent to HTTPS server.", i)
			} else {
				resp.Body.Close()
				logger.Printf("request %d successful. status=%d", i, resp.StatusCode)
			}
		}
		return b.Bytes()
	}
}

// isGoHTTPResp detects if the response is one that a go http.Server sends if an HTTP request is made to an HTTPS server.
// non Go servers may return a net.OpError instead.
func isGoHTTPResp(r *http.Response) bool {
	if r.StatusCode != http.StatusBadRequest {
		return false
	}
	p, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	return string(p) == "Client sent an HTTP request to an HTTPS server.\n"
}

// diagError tries to diagnose the error and return a cause/possible cause in a human readable format.
// If no matching errors are found err.Error is returned.
func diagError(err error) string {
	// client does not support server algorithm
	if errors.Is(err, x509.ErrUnsupportedAlgorithm) {
		return fmt.Sprintf("%v: caused by client does not support server's signature algorithm.", err)
	}

	// a *net.OpError could indicate an HTTP request made to an HTTPS server
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Err.Error() == "read: connection reset by peer" {
			return fmt.Sprintf("%v: possible cause: HTTP schema used for HTTPS server.", netErr)
		}
	}

	// Client does not have CA that matches server cert
	var unknownAuthErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthErr) {
		return fmt.Sprintf("%v: caused by no trusted client CA.", err)
	}

	// CA is ok but the server's cert is not.
	var certValidErr x509.CertificateInvalidError
	if errors.As(err, &certValidErr) {
		return fmt.Sprintf("%v: caused by invalid server certificate.", certValidErr)
	}

	// cert validation error can indicate that a custom CA needs to be used
	var tlsErr *tls.CertificateVerificationError
	if errors.As(err, &tlsErr) {
		return fmt.Sprintf("%v: possible cause: client TLS settings incorrect.", tlsErr)
	}

	// keep unwrapping to url.Error as the last error as other failures can be embedded in a url.Error
	// Can detect if an HTTPS request is made to an HTTP server
	var uErr *url.Error
	if errors.As(err, &uErr) {
		switch uErr.Err.Error() {
		case "http: server gave HTTP response to HTTPS client":
			return fmt.Sprintf("%v: caused by using HTTPS schema on HTTP server.", uErr)
		case "remote error: tls: certificate required":
			return fmt.Sprintf("%v: caused by missing mTLS client cert.", uErr)
		case "remote error: tls: expired certificate":
			return fmt.Sprintf("%v: caused by expired mTLS client cert.", uErr)
		case "remote error: tls: bad certificate":
			return fmt.Sprintf("%v: caused by invalid mTLS client cert, does the server trust the CA used for the client cert?.", uErr)
		}
	}

	return err.Error()
}
