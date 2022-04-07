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

package http

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common/transport"
)

const (
	gzipEncoding   = "gzip"
	urlSchemaHTTP  = "http"
	urlSchemaHTTPS = "https"
)

// SimpleTransport contains the dialer and read/write callbacks
type SimpleTransport struct {
	Dialer transport.Dialer

	OnStartWrite func()
	OnEndWrite   func()
	OnStartRead  func()
}

func (t *SimpleTransport) checkRequest(req *http.Request) error {
	if req.URL == nil {
		return errors.New("http: missing request URL")
	}

	if req.Header == nil {
		return errors.New("http: missing request headers")
	}

	scheme := req.URL.Scheme
	isHTTP := scheme == urlSchemaHTTP || scheme == urlSchemaHTTPS
	if !isHTTP {
		return fmt.Errorf("http: unsupported scheme %v", scheme)
	}
	if req.URL.Host == "" {
		return errors.New("http: no host in URL")
	}

	return nil
}

// RoundTrip sets up goroutines to write the request and read the responses
func (t *SimpleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	type readReturn struct {
		resp *http.Response
		err  error
	}

	defer reqCloseBody(req)

	if err := t.checkRequest(req); err != nil {
		return nil, err
	}

	conn, err := t.Dialer.Dial("tcp", canonicalAddr(req.URL))
	if err != nil {
		return nil, err
	}

	done := req.Context().Done()
	readerDone := make(chan readReturn, 1)
	writerDone := make(chan error, 1)

	// write request
	go func() {
		writerDone <- t.writeRequest(conn, req)
	}()

	// read response
	go func() {
		resp, err := t.readResponse(conn, req)
		readerDone <- readReturn{resp, err}
	}()

	select {
	case err := <-writerDone:
		if err != nil {
			return nil, err
		}
	case <-done:
		return nil, errors.New("http: request timed out before writing finished")
	}
	close(writerDone)

	var ret readReturn
	select {
	case ret = <-readerDone:
		break
	case <-done:
		// We need to free resources from the main reader
		// We start by closing the conn, which will most likely cause an error
		// in the read goroutine (unless we are right on the boundary between timeout and success)
		// and will free up both the connection and cause that go routine to terminate.
		conn.Close()
		// Now we block waiting for that goroutine to finish. We do this synchronously
		// because with a closed connection it should return immediately.
		// We can ignore the ret.err value because the error is most likely due to us
		// prematurely closing the conn
		ret := <-readerDone
		// If the body has been allocated we need to close it
		if ret.resp != nil {
			ret.resp.Body.Close()
		}
		// finally, return the real error. No need to return a response here
		return nil, errors.New("http: request timed out while waiting for response")
	}
	close(readerDone)

	return ret.resp, ret.err
}

func (t *SimpleTransport) writeRequest(conn net.Conn, req *http.Request) error {
	writer := bufio.NewWriter(conn)

	t.sigStartWrite()
	err := req.Write(writer)
	if err == nil {
		err = writer.Flush()
	}
	t.sigEndWrite()
	return err
}

// comboConnReadCloser wraps a ReadCloser that is backed by
// on a net.Conn. It will close the net.Conn when the ReadCloser is closed.
type comboConnReadCloser struct {
	conn net.Conn
	rc   io.ReadCloser
}

func (c comboConnReadCloser) Read(p []byte) (n int, err error) {
	return c.rc.Read(p)
}

func (c comboConnReadCloser) Close() error {
	defer c.conn.Close()
	return c.rc.Close()
}

func (t *SimpleTransport) readResponse(
	conn net.Conn,
	req *http.Request,
) (*http.Response, error) {
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, err
	}
	resp.Body = comboConnReadCloser{conn, resp.Body}

	t.sigStartRead()

	if resp.Header.Get("Content-Encoding") == gzipEncoding {
		unzipper, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		resp.Body = struct {
			io.Reader
			io.Closer
		}{unzipper, resp.Body}
	}

	return resp, nil
}

func (t *SimpleTransport) sigStartRead()  { call(t.OnStartRead) }
func (t *SimpleTransport) sigStartWrite() { call(t.OnStartWrite) }
func (t *SimpleTransport) sigEndWrite()   { call(t.OnEndWrite) }

func call(f func()) {
	if f != nil {
		f()
	}
}

func reqCloseBody(req *http.Request) {
	if req.Body != nil {
		req.Body.Close()
	}
}

func canonicalAddr(url *url.URL) string {
	scheme, addr := url.Scheme, url.Host
	if !hasPort(addr) {
		portmap := map[string]string{
			"http":  "80",
			"https": "443",
		}
		addr = net.JoinHostPort(addr, portmap[scheme])
	}
	return addr
}

func hasPort(s string) bool {
	return strings.LastIndex(s, ":") > strings.LastIndex(s, "]")
}
