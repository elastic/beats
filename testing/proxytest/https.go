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

package proxytest

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

func (p *Proxy) serveHTTPS(w http.ResponseWriter, r *http.Request) {
	log := loggerFromReqCtx(r)

	clientCon, err := hijack(w)
	if err != nil {
		p.http500Error(clientCon, "cannot handle request", err, log)
		return
	}
	defer clientCon.Close()

	// Hijack successful, w is now useless, let's make sure it isn't used by
	// mistake ;)
	w = nil //nolint:ineffassign,wastedassign // w is now useless, let's make sure it isn't used by mistake ;)

	// ==================== CONNECT accepted, let the client know
	_, err = clientCon.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if err != nil {
		p.http500Error(clientCon, "failed to send 200-OK after CONNECT", err, log)
		return
	}

	// ==================== TLS handshake
	// client will proceed to perform the TLS handshake with the "target",
	// which we're impersonating.

	// generate a TLS certificate matching the target's host
	cert, err := p.newTLSCert(r.URL)
	if err != nil {
		p.http500Error(clientCon, "failed generating certificate", err, log)
		return
	}

	tlscfg := p.TLS.Clone()
	tlscfg.Certificates = []tls.Certificate{*cert}
	clientTLSConn := tls.Server(clientCon, tlscfg)
	defer clientTLSConn.Close()
	err = clientTLSConn.Handshake()
	if err != nil {
		p.http500Error(clientCon, "failed TLS handshake with client", err, log)
		return
	}

	clientTLSReader := bufio.NewReader(clientTLSConn)

	notEOF := func(r *bufio.Reader) bool {
		_, err = r.Peek(1)
		return !errors.Is(err, io.EOF)
	}
	// ==================== Handle the actual request
	for notEOF(clientTLSReader) {
		// read request from the client sent after the 1s CONNECT request
		req, err := http.ReadRequest(clientTLSReader)
		if err != nil {
			p.http500Error(clientTLSConn, "failed reading client request", err, log)
			return
		}

		// carry over the original remote addr
		req.RemoteAddr = r.RemoteAddr

		// the read request is relative to the host from the original CONNECT
		// request and without scheme. Therefore, set them in the new request.
		req.URL, err = url.Parse("https://" + r.Host + req.URL.String())
		if err != nil {
			p.http500Error(clientTLSConn, "failed reading request URL from client", err, log)
			return
		}
		cleanUpHeaders(req.Header)

		// now the request is ready, it can be altered and sent just as it's
		// done for an HTTP request.
		resp, err := p.processRequest(req)
		if err != nil {
			p.httpError(clientTLSConn,
				http.StatusBadGateway,
				"failed performing request to target", err, log)
			return
		}

		clientResp := http.Response{
			ProtoMajor:       1,
			ProtoMinor:       1,
			StatusCode:       resp.StatusCode,
			TransferEncoding: append([]string{}, resp.TransferEncoding...),
			Trailer:          resp.Trailer.Clone(),
			Body:             resp.Body,
			ContentLength:    resp.ContentLength,
			Header:           resp.Header.Clone(),
		}

		err = clientResp.Write(clientTLSConn)
		if err != nil {
			p.http500Error(clientTLSConn, "failed writing response body", err, log)
			return
		}

		_ = resp.Body.Close()
	}
}

func (p *Proxy) newTLSCert(u *url.URL) (*tls.Certificate, error) {
	// generate the certificate key - it needs to be RSA because Elastic Defend
	// do not support EC :/
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not create RSA private key: %w", err)
	}
	host := u.Hostname()

	var name string
	var ips []net.IP
	ip := net.ParseIP(host)
	if ip == nil { // host isn't an IP, therefore it must be an DNS
		name = host
	} else {
		ips = append(ips, ip)
	}

	cert, _, err := certutil.GenerateGenericChildCert(
		name,
		ips,
		priv,
		&priv.PublicKey,
		p.ca.capriv,
		p.ca.cacert)
	if err != nil {
		return nil, fmt.Errorf("could not generate TLS certificate for %s: %w",
			host, err)
	}

	return cert, nil
}

func (p *Proxy) http500Error(clientCon net.Conn, msg string, err error, log *slog.Logger) {
	p.httpError(clientCon, http.StatusInternalServerError, msg, err, log)
}

func (p *Proxy) httpError(clientCon net.Conn, status int, msg string, err error, log *slog.Logger) {
	log.Error(msg, "err", err)

	resp := http.Response{
		StatusCode: status,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(strings.NewReader(msg)),
		Header:     http.Header{},
	}
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")

	err = resp.Write(clientCon)
	if err != nil {
		log.Error("failed writing response", "err", err)
	}
}

func hijack(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "cannot handle request")
		return nil, errors.New("http.ResponseWriter does not support hijacking")
	}

	clientCon, _, err := hijacker.Hijack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = fmt.Fprint(w, "cannot handle request")

		return nil, fmt.Errorf("could not Hijack HTTPS CONNECT request: %w", err)
	}

	return clientCon, err
}

func cleanUpHeaders(h http.Header) {
	h.Del("Proxy-Connection")
	h.Del("Proxy-Authenticate")
	h.Del("Proxy-Authorization")
	h.Del("Connection")
}
