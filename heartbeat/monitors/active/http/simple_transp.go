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

	"github.com/elastic/beats/libbeat/outputs/transport"
)

type SimpleTransport struct {
	Dialer             transport.Dialer
	DisableCompression bool

	OnStartWrite func()
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
	isHTTP := scheme == "http" || scheme == "https"
	if !isHTTP {
		return fmt.Errorf("http: unsupported scheme %v", scheme)
	}
	if req.URL.Host == "" {
		return errors.New("http: no host in URL")
	}

	return nil
}

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
	defer conn.Close()

	requestedGzip := false
	if t.DisableCompression &&
		req.Header.Get("Accept-Encoding") == "" &&
		req.Header.Get("Range") == "" &&
		req.Method != "HEAD" {

		requestedGzip = true
		req.Header.Add("Accept-Encoding", "gzip")
		defer req.Header.Del("Accept-Encoding")
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
		resp, err := t.readResponse(conn, req, requestedGzip)
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
	return err
}

func (t *SimpleTransport) readResponse(
	conn net.Conn,
	req *http.Request,
	requestedGzip bool,
) (*http.Response, error) {
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, err
	}
	t.sigStartRead()

	if requestedGzip && resp.Header.Get("Content-Encoding") == "gzip" {
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.ContentLength = -1

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

func (t *SimpleTransport) sigStartRead() {
	if f := t.OnStartRead; f != nil {
		f()
	}
}

func (t *SimpleTransport) sigStartWrite() {
	if f := t.OnStartWrite; f != nil {
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
