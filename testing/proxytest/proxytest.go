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
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid/v5"
)

type Proxy struct {
	*httptest.Server

	// Port is the port Server is listening on.
	Port string

	// LocalhostURL is the server URL as "http(s)://localhost:PORT".
	// Deprecated. Use Proxy.URL instead.
	LocalhostURL string

	// proxiedRequests is a "request log" for every request the proxy receives.
	proxiedRequests   []string
	proxiedRequestsMu sync.Mutex
	requestsWG        *sync.WaitGroup

	opts options
	log  *slog.Logger

	ca     ca
	client *http.Client
}

type Option func(o *options)

type options struct {
	addr        string
	rewriteHost func(string) string
	rewriteURL  func(u *url.URL)
	// logFn if set will be used to log every request.
	logFn           func(format string, a ...any)
	verbose         bool
	serverTLSConfig *tls.Config
	capriv          crypto.PrivateKey
	cacert          *x509.Certificate
	client          *http.Client
}

type ca struct {
	capriv crypto.PrivateKey
	cacert *x509.Certificate
}

// WithAddress will set the address the server will listen on. The format is as
// defined by net.Listen for a tcp connection.
func WithAddress(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

// WithHTTPClient sets http.Client used to proxy requests to the target host.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) {
		o.client = c
	}
}

// WithMITMCA sets the CA used for MITM (men in the middle) when proxying HTTPS
// requests. It's used to generate TLS certificates matching the target host.
// Ideally the CA is the same as the one issuing the TLS certificate for the
// proxy set by WithServerTLSConfig.
func WithMITMCA(priv crypto.PrivateKey, cert *x509.Certificate) func(o *options) {
	return func(o *options) {
		o.capriv = priv
		o.cacert = cert
	}
}

// WithRequestLog sets the proxy to log every request using logFn. It uses name
// as a prefix to the log.
func WithRequestLog(name string, logFn func(format string, a ...any)) Option {
	return func(o *options) {
		o.logFn = func(format string, a ...any) {
			logFn("[proxy-"+name+"] "+format, a...)
		}
	}
}

// WithRewrite will replace old by new on the request URL host when forwarding it.
func WithRewrite(old, new string) Option {
	return func(o *options) {
		o.rewriteHost = func(s string) string {
			return strings.Replace(s, old, new, 1)
		}
	}
}

// WithRewriteFn calls f on the request *url.URL before forwarding it.
// It takes precedence over WithRewrite. Use if more control over the rewrite
// is needed.
func WithRewriteFn(f func(u *url.URL)) Option {
	return func(o *options) {
		o.rewriteURL = f
	}
}

// WithServerTLSConfig sets the TLS config for the server.
func WithServerTLSConfig(tc *tls.Config) Option {
	return func(o *options) {
		o.serverTLSConfig = tc
	}
}

// WithVerboseLog sets the proxy to log every request verbosely and enables
// debug level logging. WithRequestLog must be used as well, otherwise
// WithVerboseLog will not take effect.
func WithVerboseLog() Option {
	return func(o *options) {
		o.verbose = true
	}
}

// New returns a new Proxy ready for use. Use:
//   - WithAddress to set the proxy's address,
//   - WithRewrite or WithRewriteFn to rewrite the URL before forwarding the request.
//
// Check the other With* functions for more options.
func New(t *testing.T, optns ...Option) *Proxy {
	t.Helper()

	opts := options{addr: "127.0.0.1:0", client: &http.Client{}}
	for _, o := range optns {
		o(&opts)
	}

	if opts.logFn == nil {
		opts.logFn = func(format string, a ...any) {}
	}

	l, err := net.Listen("tcp", opts.addr) //nolint:gosec,nolintlint // it's a test
	if err != nil {
		t.Fatalf("NewServer failed to create a net.Listener: %v", err)
	}

	// Create a text handler that writes to standard output
	lv := slog.LevelInfo
	if opts.verbose {
		lv = slog.LevelDebug
	}
	p := Proxy{
		requestsWG: &sync.WaitGroup{},
		opts:       opts,
		client:     opts.client,
		log: slog.New(slog.NewTextHandler(logfWriter(opts.logFn), &slog.HandlerOptions{
			Level: lv,
		})),
	}
	if opts.capriv != nil && opts.cacert != nil {
		p.ca = ca{capriv: opts.capriv, cacert: opts.cacert}
	}

	p.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(ww http.ResponseWriter, r *http.Request) {
			// Sometimes, on CI obviously, the last log happens after the test
			// finishes. See https://github.com/elastic/elastic-agent/issues/5869.
			// Therefore, let's add an extra layer to try to avoid that.
			p.requestsWG.Add(1)
			defer p.requestsWG.Done()

			w := &proxyResponseWriter{w: ww}

			requestID := uuid.Must(uuid.NewV4()).String()
			p.log.Info(fmt.Sprintf("STARTING - %s '%s' %s %s",
				r.Method, r.URL, r.Proto, r.RemoteAddr))

			rr := addIDToReqCtx(r, requestID)
			rrr := addLoggerReqCtx(rr, p.log.With("req_id", requestID))

			p.ServeHTTP(w, rrr)

			p.log.Info(fmt.Sprintf("[%s] DONE %d - %s %s %s %s\n",
				requestID, w.statusCode, r.Method, r.URL, r.Proto, r.RemoteAddr))
		}),
	)
	p.Server.Listener = l

	if opts.serverTLSConfig != nil {
		p.Server.TLS = opts.serverTLSConfig
	}

	u, err := url.Parse(p.URL)
	if err != nil {
		panic(fmt.Sprintf("could parse fleet-server URL: %v", err))
	}

	p.Port = u.Port()
	p.LocalhostURL = "http://localhost:" + p.Port

	return &p
}

func (p *Proxy) Start() error {
	p.Server.Start()
	u, err := url.Parse(p.URL)
	if err != nil {
		return fmt.Errorf("could not parse fleet-server URL: %w", err)
	}

	p.Port = u.Port()
	p.LocalhostURL = "http://localhost:" + p.Port

	p.log.Info(fmt.Sprintf("running on %s -> %s", p.URL, p.LocalhostURL))
	return nil
}

func (p *Proxy) StartTLS() error {
	p.Server.StartTLS()
	u, err := url.Parse(p.URL)
	if err != nil {
		return fmt.Errorf("could not parse fleet-server URL: %w", err)
	}

	p.Port = u.Port()
	p.LocalhostURL = "https://localhost:" + p.Port

	p.log.Info(fmt.Sprintf("running on %s -> %s", p.URL, p.LocalhostURL))
	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.serveHTTPS(w, r)
		return
	}

	p.serveHTTP(w, r)
}

func (p *Proxy) Close() {
	// Sometimes, on CI obviously, the last log happens after the test
	// finishes. See https://github.com/elastic/elastic-agent/issues/5869.
	// So, manually wait all ongoing requests to finish.
	p.requestsWG.Wait()

	p.Server.Close()
}

func (p *Proxy) serveHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := p.processRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("could not make request: %#v", err.Error())
		log.Print(msg)
		_, _ = fmt.Fprint(w, msg)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	w.WriteHeader(resp.StatusCode)

	if _, err = io.Copy(w, resp.Body); err != nil {
		p.opts.logFn("[ERROR] could not write response body: %v", err)
	}
}

// processRequest executes the configured request manipulation and perform the
// request.
func (p *Proxy) processRequest(r *http.Request) (*http.Response, error) {
	origURL := r.URL.String()

	switch {
	case p.opts.rewriteURL != nil:
		p.opts.rewriteURL(r.URL)
	case p.opts.rewriteHost != nil:
		r.URL.Host = p.opts.rewriteHost(r.URL.Host)
	}

	// It should not be required, however if not set, enroll will fail with
	// "Unknown resource"
	r.Host = r.URL.Host

	p.log.Debug(fmt.Sprintf("original URL: %s, new URL: %s",
		origURL, r.URL.String()))

	p.proxiedRequestsMu.Lock()
	p.proxiedRequests = append(p.proxiedRequests,
		fmt.Sprintf("%s - %s %s %s",
			r.Method, r.URL.Scheme, r.URL.Host, r.URL.String()))
	p.proxiedRequestsMu.Unlock()

	// when modifying the request, RequestURI isn't updated, and it isn't
	// needed anyway, so remove it.
	r.RequestURI = ""

	return p.client.Do(r)
}

// ProxiedRequests returns a slice with the "request log" with every request the
// proxy received.
func (p *Proxy) ProxiedRequests() []string {
	p.proxiedRequestsMu.Lock()
	defer p.proxiedRequestsMu.Unlock()

	var rs []string
	rs = append(rs, p.proxiedRequests...)
	return rs
}

var _ http.Hijacker = &proxyResponseWriter{}

// proxyResponseWriter wraps a http.ResponseWriter to expose the status code
// through proxyResponseWriter.statusCode
type proxyResponseWriter struct {
	w          http.ResponseWriter
	statusCode int
}

func (s *proxyResponseWriter) Header() http.Header {
	return s.w.Header()
}

func (s *proxyResponseWriter) Write(bs []byte) (int, error) {
	return s.w.Write(bs)
}

func (s *proxyResponseWriter) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.w.WriteHeader(statusCode)
}

func (s *proxyResponseWriter) StatusCode() int {
	return s.statusCode
}

func (s *proxyResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := s.w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T does not support hijacking", s.w)
	}

	return hijacker.Hijack()
}

type ctxKeyRecID struct{}
type ctxKeyLogger struct{}

func addIDToReqCtx(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxKeyRecID{}, id))
}

func idFromReqCtx(r *http.Request) string { //nolint:unused // kept for completeness
	return r.Context().Value(ctxKeyRecID{}).(string)
}

func addLoggerReqCtx(r *http.Request, log *slog.Logger) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxKeyLogger{}, log))
}

func loggerFromReqCtx(r *http.Request) *slog.Logger {
	l, ok := r.Context().Value(ctxKeyLogger{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return l
}

type logfWriter func(format string, a ...any)

func (w logfWriter) Write(p []byte) (n int, err error) {
	w(string(p))
	return len(p), nil
}
