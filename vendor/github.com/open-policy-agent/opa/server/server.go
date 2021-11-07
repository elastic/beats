// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/plugins"
	bundlePlugin "github.com/open-policy-agent/opa/plugins/bundle"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/server/authorizer"
	"github.com/open-policy-agent/opa/server/identifier"
	"github.com/open-policy-agent/opa/server/types"
	"github.com/open-policy-agent/opa/server/writer"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/topdown"
	iCache "github.com/open-policy-agent/opa/topdown/cache"
	"github.com/open-policy-agent/opa/topdown/lineage"
	"github.com/open-policy-agent/opa/util"
	"github.com/open-policy-agent/opa/version"
)

// AuthenticationScheme enumerates the supported authentication schemes. The
// authentication scheme determines how client identities are established.
type AuthenticationScheme int

// Set of supported authentication schemes.
const (
	AuthenticationOff AuthenticationScheme = iota
	AuthenticationToken
	AuthenticationTLS
)

var supportedTLSVersions = []uint16{tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13}

// AuthorizationScheme enumerates the supported authorization schemes. The authorization
// scheme determines how access to OPA is controlled.
type AuthorizationScheme int

// Set of supported authorization schemes.
const (
	AuthorizationOff AuthorizationScheme = iota
	AuthorizationBasic
)

const defaultMinTLSVersion = tls.VersionTLS12

// Set of handlers for use in the "handler" dimension of the duration metric.
const (
	PromHandlerV0Data     = "v0/data"
	PromHandlerV1Data     = "v1/data"
	PromHandlerV1Query    = "v1/query"
	PromHandlerV1Policies = "v1/policies"
	PromHandlerV1Compile  = "v1/compile"
	PromHandlerV1Config   = "v1/config"
	PromHandlerIndex      = "index"
	PromHandlerCatch      = "catchall"
	PromHandlerHealth     = "health"
)

const pqMaxCacheSize = 100

// map of unsafe builtins
var unsafeBuiltinsMap = map[string]struct{}{ast.HTTPSend.Name: {}}

// Server represents an instance of OPA running in server mode.
type Server struct {
	Handler           http.Handler
	DiagnosticHandler http.Handler

	router                 *mux.Router
	addrs                  []string
	diagAddrs              []string
	h2cEnabled             bool
	authentication         AuthenticationScheme
	authorization          AuthorizationScheme
	cert                   *tls.Certificate
	certPool               *x509.CertPool
	minTLSVersion          uint16
	mtx                    sync.RWMutex
	partials               map[string]rego.PartialResult
	preparedEvalQueries    *cache
	store                  storage.Store
	manager                *plugins.Manager
	decisionIDFactory      func() string
	buffer                 Buffer
	logger                 func(context.Context, *Info) error
	errLimit               int
	pprofEnabled           bool
	runtime                *ast.Term
	httpListeners          []httpListener
	metrics                Metrics
	defaultDecisionPath    string
	interQueryBuiltinCache iCache.InterQueryCache
	allPluginsOkOnce       bool
}

// Metrics defines the interface that the server requires for recording HTTP
// handler metrics.
type Metrics interface {
	RegisterEndpoints(registrar func(path, method string, handler http.Handler))
	InstrumentHandler(handler http.Handler, label string) http.Handler
}

// Loop will contain all the calls from the server that we'll be listening on.
type Loop func() error

// New returns a new Server.
func New() *Server {
	s := Server{}
	return &s
}

// Init initializes the server. This function MUST be called before starting any loops
// from s.Listeners().
func (s *Server) Init(ctx context.Context) (*Server, error) {
	s.initRouters()
	s.Handler = s.initHandlerAuth(s.Handler)
	s.DiagnosticHandler = s.initHandlerAuth(s.DiagnosticHandler)

	txn, err := s.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		return nil, err
	}

	// Register triggers so that if runtime reloads the policies, the
	// server sees the change.
	config := storage.TriggerConfig{
		OnCommit: s.reload,
	}
	if _, err := s.store.Register(ctx, txn, config); err != nil {
		s.store.Abort(ctx, txn)
		return nil, err
	}

	s.partials = map[string]rego.PartialResult{}
	s.preparedEvalQueries = newCache(pqMaxCacheSize)
	s.defaultDecisionPath = s.generateDefaultDecisionPath()
	s.interQueryBuiltinCache = iCache.NewInterQueryCache(s.manager.InterQueryBuiltinCacheConfig())
	s.manager.RegisterCacheTrigger(s.updateCacheConfig)

	return s, s.store.Commit(ctx, txn)
}

// Shutdown will attempt to gracefully shutdown each of the http servers
// currently in use by the OPA Server. If any exceed the deadline specified
// by the context an error will be returned.
func (s *Server) Shutdown(ctx context.Context) error {
	errChan := make(chan error)
	for _, srvr := range s.httpListeners {
		go func(s httpListener) {
			errChan <- s.Shutdown(ctx)
		}(srvr)
	}
	// wait until each server has finished shutting down
	var errorList []error
	for i := 0; i < len(s.httpListeners); i++ {
		err := <-errChan
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	if len(errorList) > 0 {
		errMsg := "error while shutting down: "
		for i, err := range errorList {
			errMsg += fmt.Sprintf("(%d) %s. ", i, err.Error())
		}
		return errors.New(errMsg)
	}
	return nil
}

// WithAddresses sets the listening addresses that the server will bind to.
func (s *Server) WithAddresses(addrs []string) *Server {
	s.addrs = addrs
	return s
}

// WithDiagnosticAddresses sets the listening addresses that the server will
// bind to and *only* serve read-only diagnostic API's.
func (s *Server) WithDiagnosticAddresses(addrs []string) *Server {
	s.diagAddrs = addrs
	return s
}

// WithAuthentication sets authentication scheme to use on the server.
func (s *Server) WithAuthentication(scheme AuthenticationScheme) *Server {
	s.authentication = scheme
	return s
}

// WithAuthorization sets authorization scheme to use on the server.
func (s *Server) WithAuthorization(scheme AuthorizationScheme) *Server {
	s.authorization = scheme
	return s
}

// WithCertificate sets the server-side certificate that the server will use.
func (s *Server) WithCertificate(cert *tls.Certificate) *Server {
	s.cert = cert
	return s
}

// WithCertPool sets the server-side cert pool that the server will use.
func (s *Server) WithCertPool(pool *x509.CertPool) *Server {
	s.certPool = pool
	return s
}

// WithStore sets the storage used by the server.
func (s *Server) WithStore(store storage.Store) *Server {
	s.store = store
	return s
}

// WithMetrics sets the metrics provider used by the server.
func (s *Server) WithMetrics(m Metrics) *Server {
	s.metrics = m
	return s
}

// WithManager sets the plugins manager used by the server.
func (s *Server) WithManager(manager *plugins.Manager) *Server {
	s.manager = manager
	return s
}

// WithCompilerErrorLimit sets the limit on the number of compiler errors the server will
// allow.
func (s *Server) WithCompilerErrorLimit(limit int) *Server {
	s.errLimit = limit
	return s
}

// WithPprofEnabled sets whether pprof endpoints are enabled
func (s *Server) WithPprofEnabled(pprofEnabled bool) *Server {
	s.pprofEnabled = pprofEnabled
	return s
}

// WithH2CEnabled sets whether h2c ("HTTP/2 cleartext") is enabled for the http listener
func (s *Server) WithH2CEnabled(enabled bool) *Server {
	s.h2cEnabled = enabled
	return s
}

// WithDecisionLogger sets the decision logger used by the
// server. DEPRECATED. Use WithDecisionLoggerWithErr instead.
func (s *Server) WithDecisionLogger(logger func(context.Context, *Info)) *Server {
	s.logger = func(ctx context.Context, info *Info) error {
		logger(ctx, info)
		return nil
	}
	return s
}

// WithDecisionLoggerWithErr sets the decision logger used by the server.
func (s *Server) WithDecisionLoggerWithErr(logger func(context.Context, *Info) error) *Server {
	s.logger = logger
	return s
}

// WithDecisionIDFactory sets a function on the server to generate decision IDs.
func (s *Server) WithDecisionIDFactory(f func() string) *Server {
	s.decisionIDFactory = f
	return s
}

// WithRuntime sets the runtime data to provide to the evaluation engine.
func (s *Server) WithRuntime(term *ast.Term) *Server {
	s.runtime = term
	return s
}

// WithRouter sets the mux.Router to attach OPA's HTTP API routes onto. If a
// router is not supplied, the server will create it's own.
func (s *Server) WithRouter(router *mux.Router) *Server {
	s.router = router
	return s
}

func (s *Server) WithMinTLSVersion(minTLSVersion uint16) *Server {
	if isMinTLSVersionSupported(minTLSVersion) {
		s.minTLSVersion = minTLSVersion
	} else {
		s.minTLSVersion = defaultMinTLSVersion
	}
	return s
}

// Listeners returns functions that listen and serve connections.
func (s *Server) Listeners() ([]Loop, error) {
	loops := []Loop{}

	handlerBindings := map[httpListenerType]struct {
		addrs   []string
		handler http.Handler
	}{
		defaultListenerType:    {s.addrs, s.Handler},
		diagnosticListenerType: {s.diagAddrs, s.DiagnosticHandler},
	}

	for t, binding := range handlerBindings {
		for _, addr := range binding.addrs {
			loop, listener, err := s.getListener(addr, binding.handler, t)
			if err != nil {
				return nil, err
			}
			s.httpListeners = append(s.httpListeners, listener)
			loops = append(loops, loop)
		}
	}

	return loops, nil
}

// Addrs returns a list of addresses that the server is listening on.
// If the server hasn't been started it will not return an address.
func (s *Server) Addrs() []string {
	return s.addrsForType(defaultListenerType)
}

// DiagnosticAddrs returns a list of addresses that the server is listening on
// for the read-only diagnostic API's (eg /health, /metrics, etc)
// If the server hasn't been started it will not return an address.
func (s *Server) DiagnosticAddrs() []string {
	return s.addrsForType(diagnosticListenerType)
}

func (s *Server) addrsForType(t httpListenerType) []string {
	var addrs []string
	for _, l := range s.httpListeners {
		a := l.Addr()
		if a != "" && l.Type() == t {
			addrs = append(addrs, a)
		}
	}
	return addrs
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlivePeriod(3 * time.Minute)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

type httpListenerType int

const (
	defaultListenerType httpListenerType = iota
	diagnosticListenerType
)

type httpListener interface {
	Addr() string
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(ctx context.Context) error
	Type() httpListenerType
}

// baseHTTPListener is just a wrapper around http.Server
type baseHTTPListener struct {
	s       *http.Server
	l       net.Listener
	t       httpListenerType
	addr    string
	addrMtx sync.RWMutex
}

var _ httpListener = (*baseHTTPListener)(nil)

func newHTTPListener(srvr *http.Server, t httpListenerType) httpListener {
	return &baseHTTPListener{s: srvr, t: t}
}

func newHTTPUnixSocketListener(srvr *http.Server, l net.Listener, t httpListenerType) httpListener {
	return &baseHTTPListener{s: srvr, l: l, t: t}
}

func (b *baseHTTPListener) ListenAndServe() error {
	addr := b.s.Addr
	if addr == "" {
		addr = ":http"
	}
	var err error
	b.l, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	b.initAddr()

	return b.s.Serve(tcpKeepAliveListener{b.l.(*net.TCPListener)})
}

func (b *baseHTTPListener) initAddr() {
	b.addrMtx.Lock()
	if addr := b.l.(*net.TCPListener).Addr(); addr != nil {
		b.addr = addr.String()
	}
	b.addrMtx.Unlock()
}

func (b *baseHTTPListener) Addr() string {
	b.addrMtx.Lock()
	defer b.addrMtx.Unlock()
	return b.addr
}

func (b *baseHTTPListener) ListenAndServeTLS(certFile, keyFile string) error {
	addr := b.s.Addr
	if addr == "" {
		addr = ":https"
	}

	var err error
	b.l, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	b.initAddr()

	defer b.l.Close()

	return b.s.ServeTLS(tcpKeepAliveListener{b.l.(*net.TCPListener)}, certFile, keyFile)
}

func (b *baseHTTPListener) Shutdown(ctx context.Context) error {
	return b.s.Shutdown(ctx)
}

func (b *baseHTTPListener) Type() httpListenerType {
	return b.t
}

func isMinTLSVersionSupported(TLSVersion uint16) bool {
	for _, version := range supportedTLSVersions {
		if TLSVersion == version {
			return true
		}
	}
	return false
}

func (s *Server) getListener(addr string, h http.Handler, t httpListenerType) (Loop, httpListener, error) {
	parsedURL, err := parseURL(addr, s.cert != nil)
	if err != nil {
		return nil, nil, err
	}

	var loop Loop
	var listener httpListener
	switch parsedURL.Scheme {
	case "unix":
		loop, listener, err = s.getListenerForUNIXSocket(parsedURL, h, t)
	case "http":
		loop, listener, err = s.getListenerForHTTPServer(parsedURL, h, t)
	case "https":
		loop, listener, err = s.getListenerForHTTPSServer(parsedURL, h, t)
	default:
		err = fmt.Errorf("invalid url scheme %q", parsedURL.Scheme)
	}

	return loop, listener, err
}

func (s *Server) getListenerForHTTPServer(u *url.URL, h http.Handler, t httpListenerType) (Loop, httpListener, error) {
	if s.h2cEnabled {
		h2s := &http2.Server{}
		h = h2c.NewHandler(h, h2s)
	}
	h1s := http.Server{
		Addr:    u.Host,
		Handler: h,
	}

	l := newHTTPListener(&h1s, t)
	return l.ListenAndServe, l, nil
}

func (s *Server) getListenerForHTTPSServer(u *url.URL, h http.Handler, t httpListenerType) (Loop, httpListener, error) {

	if s.cert == nil {
		return nil, nil, fmt.Errorf("TLS certificate required but not supplied")
	}

	httpsServer := http.Server{
		Addr:    u.Host,
		Handler: h,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{*s.cert},
			ClientCAs:    s.certPool,
		},
	}
	if s.authentication == AuthenticationTLS {
		httpsServer.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if s.minTLSVersion != 0 {
		httpsServer.TLSConfig.MinVersion = s.minTLSVersion
	} else {
		httpsServer.TLSConfig.MinVersion = defaultMinTLSVersion
	}

	l := newHTTPListener(&httpsServer, t)

	httpsLoop := func() error { return l.ListenAndServeTLS("", "") }

	return httpsLoop, l, nil
}

func (s *Server) getListenerForUNIXSocket(u *url.URL, h http.Handler, t httpListenerType) (Loop, httpListener, error) {
	socketPath := u.Host + u.Path

	// Recover @ prefix for abstract Unix sockets.
	if strings.HasPrefix(u.String(), u.Scheme+"://@") {
		socketPath = "@" + socketPath
	} else {
		// Remove domain socket file in case it already exists.
		os.Remove(socketPath)
	}

	domainSocketServer := http.Server{Handler: h}
	unixListener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, err
	}

	l := newHTTPUnixSocketListener(&domainSocketServer, unixListener, t)

	domainSocketLoop := func() error { return domainSocketServer.Serve(unixListener) }
	return domainSocketLoop, l, nil
}

func (s *Server) initHandlerAuth(handler http.Handler) http.Handler {
	// Add authorization handler. This must come BEFORE authentication handler
	// so that the latter can run first.
	switch s.authorization {
	case AuthorizationBasic:
		handler = authorizer.NewBasic(
			handler,
			s.getCompiler,
			s.store,
			authorizer.Runtime(s.runtime),
			authorizer.Decision(s.manager.Config.DefaultAuthorizationDecisionRef))
	}

	switch s.authentication {
	case AuthenticationToken:
		handler = identifier.NewTokenBased(handler)
	case AuthenticationTLS:
		handler = identifier.NewTLSBased(handler)
	}

	return handler
}

func (s *Server) initRouters() {
	mainRouter := s.router
	if mainRouter == nil {
		mainRouter = mux.NewRouter()
	}

	diagRouter := mux.NewRouter()

	// All routers get the same base configuration *and* diagnostic API's
	for _, router := range []*mux.Router{mainRouter, diagRouter} {
		router.StrictSlash(true)
		router.UseEncodedPath()
		router.StrictSlash(true)

		if s.metrics != nil {
			s.metrics.RegisterEndpoints(func(path, method string, handler http.Handler) {
				router.Handle(path, handler).Methods(method)
			})
		}

		router.Handle("/health", s.instrumentHandler(s.unversionedGetHealth, PromHandlerHealth)).Methods(http.MethodGet)
		// Use this route to evaluate health policy defined at system.health
		// By convention, policy is typically defined at system.health.live and system.health.ready, and is
		// evaluated by calling /health/live and /health/ready respectively.
		router.Handle("/health/{path:.+}", s.instrumentHandler(s.unversionedGetHealthWithPolicy, PromHandlerHealth)).Methods(http.MethodGet)
	}

	if s.pprofEnabled {
		mainRouter.HandleFunc("/debug/pprof/", pprof.Index)
		mainRouter.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
		mainRouter.Handle("/debug/pprof/block", pprof.Handler("block"))
		mainRouter.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mainRouter.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mainRouter.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mainRouter.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mainRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mainRouter.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	// Only the main mainRouter gets the OPA API's (data, policies, query, etc)
	s.registerHandler(mainRouter, 0, "/data/{path:.+}", http.MethodPost, s.instrumentHandler(s.v0DataPost, PromHandlerV0Data))
	s.registerHandler(mainRouter, 0, "/data", http.MethodPost, s.instrumentHandler(s.v0DataPost, PromHandlerV0Data))
	s.registerHandler(mainRouter, 1, "/data/{path:.+}", http.MethodDelete, s.instrumentHandler(s.v1DataDelete, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data/{path:.+}", http.MethodPut, s.instrumentHandler(s.v1DataPut, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data", http.MethodPut, s.instrumentHandler(s.v1DataPut, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data/{path:.+}", http.MethodGet, s.instrumentHandler(s.v1DataGet, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data", http.MethodGet, s.instrumentHandler(s.v1DataGet, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data/{path:.+}", http.MethodPatch, s.instrumentHandler(s.v1DataPatch, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data", http.MethodPatch, s.instrumentHandler(s.v1DataPatch, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data/{path:.+}", http.MethodPost, s.instrumentHandler(s.v1DataPost, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/data", http.MethodPost, s.instrumentHandler(s.v1DataPost, PromHandlerV1Data))
	s.registerHandler(mainRouter, 1, "/policies", http.MethodGet, s.instrumentHandler(s.v1PoliciesList, PromHandlerV1Policies))
	s.registerHandler(mainRouter, 1, "/policies/{path:.+}", http.MethodDelete, s.instrumentHandler(s.v1PoliciesDelete, PromHandlerV1Policies))
	s.registerHandler(mainRouter, 1, "/policies/{path:.+}", http.MethodGet, s.instrumentHandler(s.v1PoliciesGet, PromHandlerV1Policies))
	s.registerHandler(mainRouter, 1, "/policies/{path:.+}", http.MethodPut, s.instrumentHandler(s.v1PoliciesPut, PromHandlerV1Policies))
	s.registerHandler(mainRouter, 1, "/query", http.MethodGet, s.instrumentHandler(s.v1QueryGet, PromHandlerV1Query))
	s.registerHandler(mainRouter, 1, "/query", http.MethodPost, s.instrumentHandler(s.v1QueryPost, PromHandlerV1Query))
	s.registerHandler(mainRouter, 1, "/compile", http.MethodPost, s.instrumentHandler(s.v1CompilePost, PromHandlerV1Compile))
	s.registerHandler(mainRouter, 1, "/config", http.MethodGet, s.instrumentHandler(s.v1ConfigGet, PromHandlerV1Config))
	mainRouter.Handle("/", s.instrumentHandler(s.unversionedPost, PromHandlerIndex)).Methods(http.MethodPost)
	mainRouter.Handle("/", s.instrumentHandler(s.indexGet, PromHandlerIndex)).Methods(http.MethodGet)

	// These are catch all handlers that respond 405 for resources that exist but the method is not allowed
	mainRouter.Handle("/v0/data/{path:.*}", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodGet, http.MethodHead,
		http.MethodConnect, http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodPut, http.MethodTrace)
	mainRouter.Handle("/v0/data", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodGet, http.MethodHead,
		http.MethodConnect, http.MethodDelete, http.MethodOptions, http.MethodPatch, http.MethodPut,
		http.MethodTrace)
	// v1 Data catch all
	mainRouter.Handle("/v1/data/{path:.*}", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodHead,
		http.MethodConnect, http.MethodOptions, http.MethodTrace)
	mainRouter.Handle("/v1/data", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodHead,
		http.MethodConnect, http.MethodDelete, http.MethodOptions, http.MethodTrace)
	// Policies catch all
	mainRouter.Handle("/v1/policies", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodHead,
		http.MethodConnect, http.MethodDelete, http.MethodOptions, http.MethodTrace, http.MethodPost, http.MethodPut,
		http.MethodPatch)
	// Policies (/policies/{path.+} catch all
	mainRouter.Handle("/v1/policies/{path:.*}", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodHead,
		http.MethodConnect, http.MethodOptions, http.MethodTrace, http.MethodPost)
	// Query catch all
	mainRouter.Handle("/v1/query/{path:.*}", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodHead,
		http.MethodConnect, http.MethodDelete, http.MethodOptions, http.MethodTrace, http.MethodPost, http.MethodPut, http.MethodPatch)
	mainRouter.Handle("/v1/query", s.instrumentHandler(writer.HTTPStatus(405), PromHandlerCatch)).Methods(http.MethodHead,
		http.MethodConnect, http.MethodDelete, http.MethodOptions, http.MethodTrace, http.MethodPut, http.MethodPatch)

	s.Handler = mainRouter
	s.DiagnosticHandler = diagRouter
}

func (s *Server) instrumentHandler(handler func(http.ResponseWriter, *http.Request), label string) http.Handler {
	if s.metrics != nil {
		return s.metrics.InstrumentHandler(http.HandlerFunc(handler), label)
	}
	return http.HandlerFunc(handler)
}

func (s *Server) execQuery(ctx context.Context, r *http.Request, br bundleRevisions, txn storage.Transaction, decisionID string, parsedQuery ast.Body, input ast.Value, m metrics.Metrics, explainMode types.ExplainModeV1, includeMetrics, includeInstrumentation, pretty bool) (results types.QueryResponseV1, err error) {

	logger := s.getDecisionLogger(br)

	var buf *topdown.BufferTracer
	if explainMode != types.ExplainOffV1 {
		buf = topdown.NewBufferTracer()
	}

	var rawInput *interface{}
	if input != nil {
		x, err := ast.JSON(input)
		if err != nil {
			return results, err
		}
		rawInput = &x
	}

	opts := []func(*rego.Rego){
		rego.Store(s.store),
		rego.Transaction(txn),
		rego.Compiler(s.getCompiler()),
		rego.ParsedQuery(parsedQuery),
		rego.ParsedInput(input),
		rego.Metrics(m),
		rego.Instrument(includeInstrumentation),
		rego.QueryTracer(buf),
		rego.Runtime(s.runtime),
		rego.UnsafeBuiltins(unsafeBuiltinsMap),
		rego.InterQueryBuiltinCache(s.interQueryBuiltinCache),
		rego.PrintHook(s.manager.PrintHook()),
		rego.EnablePrintStatements(s.manager.EnablePrintStatements()),
	}

	for _, r := range s.manager.GetWasmResolvers() {
		for _, entrypoint := range r.Entrypoints() {
			opts = append(opts, rego.Resolver(entrypoint, r))
		}
	}

	rego := rego.New(opts...)

	output, err := rego.Eval(ctx)
	if err != nil {
		_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, "", parsedQuery.String(), rawInput, input, nil, err, m)
		return results, err
	}

	for _, result := range output {
		results.Result = append(results.Result, result.Bindings.WithoutWildcards())
	}

	if includeMetrics || includeInstrumentation {
		results.Metrics = m.All()
	}

	if explainMode != types.ExplainOffV1 {
		results.Explanation = s.getExplainResponse(explainMode, *buf, pretty)
	}

	var x interface{} = results.Result
	err = logger.Log(ctx, txn, decisionID, r.RemoteAddr, "", parsedQuery.String(), rawInput, input, &x, nil, m)
	return results, err
}

func (s *Server) indexGet(w http.ResponseWriter, r *http.Request) {
	_ = indexHTML.Execute(w, struct {
		Version        string
		BuildCommit    string
		BuildTimestamp string
		BuildHostname  string
	}{
		Version:        version.Version,
		BuildCommit:    version.Vcs,
		BuildTimestamp: version.Timestamp,
		BuildHostname:  version.Hostname,
	})
}

func (s *Server) registerHandler(router *mux.Router, version int, path string, method string, h http.Handler) {
	prefix := fmt.Sprintf("/v%d", version)
	router.Handle(prefix+path, h).Methods(method)
}

type bundleRevisions struct {
	LegacyRevision string
	Revisions      map[string]string
}

func getRevisions(ctx context.Context, store storage.Store, txn storage.Transaction) (bundleRevisions, error) {

	var err error
	var br bundleRevisions
	br.Revisions = map[string]string{}

	// Check if we still have a legacy bundle manifest in the store
	br.LegacyRevision, err = bundle.LegacyReadRevisionFromStore(ctx, store, txn)
	if err != nil && !storage.IsNotFound(err) {
		return br, err
	}

	// read all bundle revisions from storage (if any exist)
	names, err := bundle.ReadBundleNamesFromStore(ctx, store, txn)
	if err != nil && !storage.IsNotFound(err) {
		return br, err
	}

	for _, name := range names {
		r, err := bundle.ReadBundleRevisionFromStore(ctx, store, txn, name)
		if err != nil && !storage.IsNotFound(err) {
			return br, err
		}
		br.Revisions[name] = r
	}

	return br, nil
}

func (s *Server) reload(ctx context.Context, txn storage.Transaction, event storage.TriggerEvent) {

	// NOTE(tsandall): We currently rely on the storage txn to provide
	// critical sections in the server.
	//
	// If you modify this function to change any other state on the server, you must
	// review the other places in the server where that state is accessed to avoid data
	// races--the state must be accessed _after_ a txn has been opened.

	// reset some cached info
	s.partials = map[string]rego.PartialResult{}
	s.preparedEvalQueries = newCache(pqMaxCacheSize)
	s.defaultDecisionPath = s.generateDefaultDecisionPath()
}

func (s *Server) unversionedPost(w http.ResponseWriter, r *http.Request) {
	s.v0QueryPath(w, r, "", true)
}

func (s *Server) v0DataPost(w http.ResponseWriter, r *http.Request) {
	s.v0QueryPath(w, r, mux.Vars(r)["path"], false)
}

func (s *Server) v0QueryPath(w http.ResponseWriter, r *http.Request, urlPath string, useDefaultDecisionPath bool) {
	m := metrics.New()
	m.Timer(metrics.ServerHandler).Start()

	decisionID := s.generateDecisionID()

	ctx := r.Context()
	input, err := readInputV0(r)
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, errors.Wrapf(err, "unexpected parse error for input"))
		return
	}

	var goInput *interface{}
	if input != nil {
		x, err := ast.JSON(input)
		if err != nil {
			writer.ErrorString(w, http.StatusInternalServerError, types.CodeInvalidParameter, errors.Wrapf(err, "could not marshal input"))
			return
		}
		goInput = &x
	}

	// Prepare for query.
	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	br, err := getRevisions(ctx, s.store, txn)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	if useDefaultDecisionPath {
		urlPath = s.defaultDecisionPath
	}

	logger := s.getDecisionLogger(br)

	pqID := "v0QueryPath::" + urlPath
	preparedQuery, ok := s.getCachedPreparedEvalQuery(pqID, m)
	if !ok {
		path := stringPathToDataRef(urlPath)

		opts := []func(*rego.Rego){
			rego.Compiler(s.getCompiler()),
			rego.Store(s.store),
			rego.Transaction(txn),
			rego.Query(path.String()),
			rego.Metrics(m),
			rego.Runtime(s.runtime),
			rego.UnsafeBuiltins(unsafeBuiltinsMap),
			rego.PrintHook(s.manager.PrintHook()),
		}

		// Set resolvers on the base Rego object to avoid having them get
		// re-initialized, and to propagate them to the prepared query.
		for _, r := range s.manager.GetWasmResolvers() {
			for _, entrypoint := range r.Entrypoints() {
				opts = append(opts, rego.Resolver(entrypoint, r))
			}
		}

		pq, err := rego.New(opts...).PrepareForEval(ctx)
		if err != nil {
			_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
			writer.ErrorAuto(w, err)
			return
		}
		preparedQuery = &pq
		s.preparedEvalQueries.Insert(pqID, preparedQuery)
	}

	evalOpts := []rego.EvalOption{
		rego.EvalTransaction(txn),
		rego.EvalParsedInput(input),
		rego.EvalMetrics(m),
		rego.EvalInterQueryBuiltinCache(s.interQueryBuiltinCache),
	}

	rs, err := preparedQuery.Eval(
		ctx,
		evalOpts...,
	)

	m.Timer(metrics.ServerHandler).Stop()

	// Handle results.
	if err != nil {
		_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
		writer.ErrorAuto(w, err)
		return
	}

	if len(rs) == 0 {
		err := types.NewErrorV1(types.CodeUndefinedDocument, fmt.Sprintf("%v: %v", types.MsgUndefinedError, stringPathToDataRef(urlPath)))
		if logErr := logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m); logErr != nil {
			writer.ErrorAuto(w, logErr)
			return
		}

		writer.Error(w, 404, err)
		return
	}
	err = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, &rs[0].Expressions[0].Value, nil, m)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	writer.JSON(w, 200, rs[0].Expressions[0].Value, pretty)
}

func (s *Server) getCachedPreparedEvalQuery(key string, m metrics.Metrics) (*rego.PreparedEvalQuery, bool) {
	pq, ok := s.preparedEvalQueries.Get(key)
	m.Counter(metrics.ServerQueryCacheHit) // Creates the counter on the metrics if it doesn't exist, starts at 0
	if ok {
		m.Counter(metrics.ServerQueryCacheHit).Incr() // Increment counter on hit
		return pq.(*rego.PreparedEvalQuery), true
	}
	return nil, false
}

func (s *Server) canEval(ctx context.Context) bool {
	// Create very simple query that binds a single variable.
	opts := []func(*rego.Rego){
		rego.Compiler(s.getCompiler()),
		rego.Store(s.store),
		rego.Query("x = 1"),
	}

	for _, r := range s.manager.GetWasmResolvers() {
		for _, ep := range r.Entrypoints() {
			opts = append(opts, rego.Resolver(ep, r))
		}
	}

	eval := rego.New(opts...)
	// Run evaluation.
	rs, err := eval.Eval(ctx)
	if err != nil {
		return false
	}

	v, ok := rs[0].Bindings["x"]
	if ok {
		jsonNumber, ok := v.(json.Number)
		if ok && jsonNumber.String() == "1" {
			return true
		}
	}
	return false
}

func (s *Server) bundlesReady(pluginStatuses map[string]*plugins.Status) bool {

	// Look for a discovery plugin first, if it exists and isn't ready
	// then don't bother with the others.
	// Note: use "discovery" instead of `discovery.Name` to avoid import
	// cycle problems..
	dpStatus, ok := pluginStatuses["discovery"]
	if ok && dpStatus != nil && (dpStatus.State != plugins.StateOK) {
		return false
	}

	// The bundle plugin won't return "OK" until the first activation
	// of each configured bundle.
	bpStatus, ok := pluginStatuses[bundlePlugin.Name]
	if ok && bpStatus != nil && (bpStatus.State != plugins.StateOK) {
		return false
	}

	return true
}

func (s *Server) unversionedGetHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	includeBundleStatus := getBoolParam(r.URL, types.ParamBundleActivationV1, true) ||
		getBoolParam(r.URL, types.ParamBundlesActivationV1, true)
	includePluginStatus := getBoolParam(r.URL, types.ParamPluginsV1, true)
	excludePlugin := getStringSliceParam(r.URL, types.ParamExcludePluginV1)
	excludePluginMap := map[string]struct{}{}
	for _, name := range excludePlugin {
		excludePluginMap[name] = struct{}{}
	}

	// Ensure the server can evaluate a simple query
	if !s.canEval(ctx) {
		writeHealthResponse(w, errors.New("unable to perform evaluation"))
		return
	}

	pluginStatuses := s.manager.PluginStatus()

	// Ensure that bundles (if configured, and requested to be included in the result)
	// have been activated successfully. This will include discovery bundles as well as
	// normal bundles that are configured.
	if includeBundleStatus && !s.bundlesReady(pluginStatuses) {
		// For backwards compatibility we don't return a payload with statuses for the bundle endpoint
		writeHealthResponse(w, errors.New("one or more bundles are not activated"))
		return
	}

	if includePluginStatus {
		// Ensure that all plugins (if requested to be included in the result) have an OK status.
		hasErr := false
		for name, status := range pluginStatuses {
			if _, exclude := excludePluginMap[name]; exclude {
				continue
			}
			if status != nil && status.State != plugins.StateOK {
				hasErr = true
				break
			}
		}
		if hasErr {
			writeHealthResponse(w, errors.New("one or more plugins are not up"))
			return
		}
	}
	writeHealthResponse(w, nil)
}

func (s *Server) unversionedGetHealthWithPolicy(w http.ResponseWriter, r *http.Request) {
	pluginStatus := s.manager.PluginStatus()
	pluginState := map[string]string{}

	// optimistically assume all plugins are ok
	allPluginsOk := true

	// build input document for health check query
	input := func() map[string]interface{} {
		s.mtx.Lock()
		defer s.mtx.Unlock()

		// iterate over plugin status to extract state
		for name, status := range pluginStatus {
			if status != nil {
				pluginState[name] = string(status.State)
				// if all plugins have not been in OK state yet, then check to see if plugin state is OKx
				if !s.allPluginsOkOnce && status.State != plugins.StateOK {
					allPluginsOk = false
				}
			}
		}
		// once all plugins are OK, set the allPluginsOkOnce flag to true, indicating that all
		// plugins have achieved a "ready" state at least once on the server.
		if allPluginsOk {
			s.allPluginsOkOnce = true
		}

		return map[string]interface{}{
			"plugin_state":  pluginState,
			"plugins_ready": s.allPluginsOkOnce,
		}
	}()

	vars := mux.Vars(r)
	urlPath := vars["path"]
	healthDataPath := fmt.Sprintf("/system/health/%s", urlPath)
	healthDataPath = stringPathToDataRef(healthDataPath).String()

	rego := rego.New(
		rego.Query(healthDataPath),
		rego.Compiler(s.getCompiler()),
		rego.Store(s.store),
		rego.Input(input),
		rego.Runtime(s.runtime),
		rego.PrintHook(s.manager.PrintHook()),
	)

	rs, err := rego.Eval(r.Context())

	if err != nil {
		writeHealthResponse(w, err)
		return
	}

	if len(rs) == 0 {
		writeHealthResponse(w, fmt.Errorf("health check (%v) was undefined", healthDataPath))
		return
	}

	result, ok := rs[0].Expressions[0].Value.(bool)

	if ok && result {
		writeHealthResponse(w, nil)
		return
	}

	writeHealthResponse(w, fmt.Errorf("health check (%v) returned unexpected value", healthDataPath))
}

func writeHealthResponse(w http.ResponseWriter, err error) {
	status := http.StatusOK
	var response types.HealthResponseV1

	if err != nil {
		status = http.StatusInternalServerError
		response.Error = err.Error()
	}

	writer.JSON(w, status, response, false)
}

func (s *Server) v1CompilePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	explainMode := getExplain(r.URL.Query()[types.ParamExplainV1], types.ExplainOffV1)
	includeMetrics := getBoolParam(r.URL, types.ParamMetricsV1, true)
	includeInstrumentation := getBoolParam(r.URL, types.ParamInstrumentV1, true)

	m := metrics.New()
	m.Timer(metrics.ServerHandler).Start()
	m.Timer(metrics.RegoQueryParse).Start()

	request, reqErr := readInputCompilePostV1(r.Body)
	if reqErr != nil {
		writer.Error(w, http.StatusBadRequest, reqErr)
		return
	}

	m.Timer(metrics.RegoQueryParse).Stop()

	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	var buf *topdown.BufferTracer
	if explainMode != types.ExplainOffV1 {
		buf = topdown.NewBufferTracer()
	}

	eval := rego.New(
		rego.Compiler(s.getCompiler()),
		rego.Store(s.store),
		rego.Transaction(txn),
		rego.ParsedQuery(request.Query),
		rego.ParsedInput(request.Input),
		rego.ParsedUnknowns(request.Unknowns),
		rego.QueryTracer(buf),
		rego.Instrument(includeInstrumentation),
		rego.Metrics(m),
		rego.Runtime(s.runtime),
		rego.UnsafeBuiltins(unsafeBuiltinsMap),
		rego.InterQueryBuiltinCache(s.interQueryBuiltinCache),
		rego.PrintHook(s.manager.PrintHook()),
	)

	pq, err := eval.Partial(ctx)
	if err != nil {
		switch err := err.(type) {
		case ast.Errors:
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgCompileModuleError).WithASTErrors(err))
		default:
			writer.ErrorAuto(w, err)
		}
		return
	}

	m.Timer(metrics.ServerHandler).Stop()

	result := types.CompileResponseV1{}

	if includeMetrics || includeInstrumentation {
		result.Metrics = m.All()
	}

	if explainMode != types.ExplainOffV1 {
		result.Explanation = s.getExplainResponse(explainMode, *buf, pretty)
	}

	var i interface{} = types.PartialEvaluationResultV1{
		Queries: pq.Queries,
		Support: pq.Support,
	}

	result.Result = &i

	writer.JSON(w, 200, result, pretty)
}

func (s *Server) v1DataGet(w http.ResponseWriter, r *http.Request) {
	m := metrics.New()

	m.Timer(metrics.ServerHandler).Start()

	decisionID := s.generateDecisionID()

	ctx := r.Context()
	vars := mux.Vars(r)
	urlPath := vars["path"]
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	explainMode := getExplain(r.URL.Query()["explain"], types.ExplainOffV1)
	includeMetrics := getBoolParam(r.URL, types.ParamMetricsV1, true)
	includeInstrumentation := getBoolParam(r.URL, types.ParamInstrumentV1, true)
	provenance := getBoolParam(r.URL, types.ParamProvenanceV1, true)
	strictBuiltinErrors := getBoolParam(r.URL, types.ParamStrictBuiltinErrors, true)

	m.Timer(metrics.RegoInputParse).Start()

	inputs := r.URL.Query()[types.ParamInputV1]

	var input ast.Value

	if len(inputs) > 0 {
		var err error
		input, err = readInputGetV1(inputs[len(inputs)-1])
		if err != nil {
			writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
			return
		}
	}

	var goInput *interface{}
	if input != nil {
		x, err := ast.JSON(input)
		if err != nil {
			writer.ErrorString(w, http.StatusInternalServerError, types.CodeInvalidParameter, errors.Wrapf(err, "could not marshal input"))
			return
		}
		goInput = &x
	}

	m.Timer(metrics.RegoInputParse).Stop()

	// Prepare for query.
	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}
	defer s.store.Abort(ctx, txn)

	br, err := getRevisions(ctx, s.store, txn)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	logger := s.getDecisionLogger(br)

	var buf *topdown.BufferTracer

	if explainMode != types.ExplainOffV1 {
		buf = topdown.NewBufferTracer()
	}

	pqID := "v1DataGet::"
	if strictBuiltinErrors {
		pqID += "strict-builtin-errors::"
	}
	pqID += urlPath
	preparedQuery, ok := s.getCachedPreparedEvalQuery(pqID, m)
	if !ok {
		opts := []func(*rego.Rego){
			rego.Compiler(s.getCompiler()),
			rego.Store(s.store),
			rego.Transaction(txn),
			rego.ParsedInput(input),
			rego.Query(stringPathToDataRef(urlPath).String()),
			rego.Metrics(m),
			rego.QueryTracer(buf),
			rego.Instrument(includeInstrumentation),
			rego.Runtime(s.runtime),
			rego.UnsafeBuiltins(unsafeBuiltinsMap),
			rego.StrictBuiltinErrors(strictBuiltinErrors),
			rego.PrintHook(s.manager.PrintHook()),
		}

		for _, r := range s.manager.GetWasmResolvers() {
			for _, entrypoint := range r.Entrypoints() {
				opts = append(opts, rego.Resolver(entrypoint, r))
			}
		}

		pq, err := rego.New(opts...).PrepareForEval(ctx)
		if err != nil {
			_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
			writer.ErrorAuto(w, err)
			return
		}
		preparedQuery = &pq
		s.preparedEvalQueries.Insert(pqID, preparedQuery)
	}

	evalOpts := []rego.EvalOption{
		rego.EvalTransaction(txn),
		rego.EvalParsedInput(input),
		rego.EvalMetrics(m),
		rego.EvalQueryTracer(buf),
		rego.EvalInterQueryBuiltinCache(s.interQueryBuiltinCache),
		rego.EvalInstrument(includeInstrumentation),
	}

	rs, err := preparedQuery.Eval(
		ctx,
		evalOpts...,
	)

	m.Timer(metrics.ServerHandler).Stop()

	// Handle results.
	if err != nil {
		_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
		writer.ErrorAuto(w, err)
		return
	}

	result := types.DataResponseV1{
		DecisionID: decisionID,
	}

	if includeMetrics || includeInstrumentation {
		result.Metrics = m.All()
	}

	if provenance {
		result.Provenance = s.getProvenance(br)
	}

	if len(rs) == 0 {
		if explainMode == types.ExplainFullV1 {
			result.Explanation, err = types.NewTraceV1(*buf, pretty)
			if err != nil {
				writer.ErrorAuto(w, err)
			}
		}
		err = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, nil, m)
		if err != nil {
			writer.ErrorAuto(w, err)
			return
		}
		writer.JSON(w, 200, result, pretty)
		return
	}

	result.Result = &rs[0].Expressions[0].Value

	if explainMode != types.ExplainOffV1 {
		result.Explanation = s.getExplainResponse(explainMode, *buf, pretty)
	}

	err = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, result.Result, nil, m)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}
	writer.JSON(w, 200, result, pretty)
}

func (s *Server) v1DataPatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	ops := []types.PatchV1{}

	if err := util.NewJSONDecoder(r.Body).Decode(&ops); err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	patches, err := s.prepareV1PatchSlice(vars["path"], ops)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	txn, err := s.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	for _, patch := range patches {
		if err := s.checkPathScope(ctx, txn, patch.path); err != nil {
			s.abortAuto(ctx, txn, w, err)
			return
		}

		if err := s.store.Write(ctx, txn, patch.op, patch.path, patch.value); err != nil {
			s.abortAuto(ctx, txn, w, err)
			return
		}
	}

	if err := ast.CheckPathConflicts(s.getCompiler(), storage.NonEmpty(ctx, s.store, txn)); len(err) > 0 {
		s.store.Abort(ctx, txn)
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	if err := s.store.Commit(ctx, txn); err != nil {
		writer.ErrorAuto(w, err)
	} else {
		writer.Bytes(w, 204, nil)
	}
}

func (s *Server) v1DataPost(w http.ResponseWriter, r *http.Request) {
	m := metrics.New()
	m.Timer(metrics.ServerHandler).Start()

	decisionID := s.generateDecisionID()

	ctx := r.Context()
	vars := mux.Vars(r)
	urlPath := vars["path"]
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	explainMode := getExplain(r.URL.Query()[types.ParamExplainV1], types.ExplainOffV1)
	includeMetrics := getBoolParam(r.URL, types.ParamMetricsV1, true)
	includeInstrumentation := getBoolParam(r.URL, types.ParamInstrumentV1, true)
	partial := getBoolParam(r.URL, types.ParamPartialV1, true)
	provenance := getBoolParam(r.URL, types.ParamProvenanceV1, true)
	strictBuiltinErrors := getBoolParam(r.URL, types.ParamStrictBuiltinErrors, true)

	m.Timer(metrics.RegoInputParse).Start()

	input, err := readInputPostV1(r)
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	var goInput *interface{}
	if input != nil {
		x, err := ast.JSON(input)
		if err != nil {
			writer.ErrorString(w, http.StatusInternalServerError, types.CodeInvalidParameter, errors.Wrapf(err, "could not marshal input"))
			return
		}
		goInput = &x
	}

	m.Timer(metrics.RegoInputParse).Stop()

	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	br, err := getRevisions(ctx, s.store, txn)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	logger := s.getDecisionLogger(br)

	var buf *topdown.BufferTracer

	if explainMode != types.ExplainOffV1 {
		buf = topdown.NewBufferTracer()
	}

	pqID := "v1DataPost::"
	if partial {
		pqID += "partial::"
	}
	if strictBuiltinErrors {
		pqID += "strict-builtin-errors::"
	}
	pqID += urlPath
	preparedQuery, ok := s.getCachedPreparedEvalQuery(pqID, m)
	if !ok {
		opts := []func(*rego.Rego){
			rego.Compiler(s.getCompiler()),
			rego.Store(s.store),
			rego.StrictBuiltinErrors(strictBuiltinErrors),
		}

		// Set resolvers on the base Rego object to avoid having them get
		// re-initialized, and to propagate them to the prepared query.
		for _, r := range s.manager.GetWasmResolvers() {
			for _, entrypoint := range r.Entrypoints() {
				opts = append(opts, rego.Resolver(entrypoint, r))
			}
		}

		rego, err := s.makeRego(ctx, partial, txn, input, urlPath, m, includeInstrumentation, buf, opts)

		if err != nil {
			_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
			writer.ErrorAuto(w, err)
			return
		}

		pq, err := rego.PrepareForEval(ctx)
		if err != nil {
			_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
			writer.ErrorAuto(w, err)
			return
		}
		preparedQuery = &pq
		s.preparedEvalQueries.Insert(pqID, preparedQuery)
	}

	evalOpts := []rego.EvalOption{
		rego.EvalTransaction(txn),
		rego.EvalParsedInput(input),
		rego.EvalMetrics(m),
		rego.EvalQueryTracer(buf),
		rego.EvalInterQueryBuiltinCache(s.interQueryBuiltinCache),
		rego.EvalInstrument(includeInstrumentation),
	}

	rs, err := preparedQuery.Eval(
		ctx,
		evalOpts...,
	)

	m.Timer(metrics.ServerHandler).Stop()

	// Handle results.
	if err != nil {
		_ = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, err, m)
		writer.ErrorAuto(w, err)
		return
	}

	result := types.DataResponseV1{
		DecisionID: decisionID,
	}

	if includeMetrics || includeInstrumentation {
		result.Metrics = m.All()
	}

	if provenance {
		result.Provenance = s.getProvenance(br)
	}

	if len(rs) == 0 {
		if explainMode == types.ExplainFullV1 {
			result.Explanation, err = types.NewTraceV1(*buf, pretty)
			if err != nil {
				writer.ErrorAuto(w, err)
				return
			}
		}
		err = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, nil, nil, m)
		if err != nil {
			writer.ErrorAuto(w, err)
			return
		}
		writer.JSON(w, 200, result, pretty)
		return
	}

	result.Result = &rs[0].Expressions[0].Value

	if explainMode != types.ExplainOffV1 {
		result.Explanation = s.getExplainResponse(explainMode, *buf, pretty)
	}

	err = logger.Log(ctx, txn, decisionID, r.RemoteAddr, urlPath, "", goInput, input, result.Result, nil, m)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}
	writer.JSON(w, 200, result, pretty)
}

func (s *Server) v1DataPut(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	var value interface{}
	if err := util.NewJSONDecoder(r.Body).Decode(&value); err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	path, ok := storage.ParsePathEscaped("/" + strings.Trim(vars["path"], "/"))
	if !ok {
		writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, "bad path: %v", vars["path"]))
		return
	}

	txn, err := s.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	if err := s.checkPathScope(ctx, txn, path); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	_, err = s.store.Read(ctx, txn, path)

	if err != nil {
		if !storage.IsNotFound(err) {
			s.abortAuto(ctx, txn, w, err)
			return
		}
		if err := storage.MakeDir(ctx, s.store, txn, path[:len(path)-1]); err != nil {
			s.abortAuto(ctx, txn, w, err)
			return
		}
	} else if r.Header.Get("If-None-Match") == "*" {
		s.store.Abort(ctx, txn)
		writer.Bytes(w, 304, nil)
		return
	}

	if err := s.store.Write(ctx, txn, storage.AddOp, path, value); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	if err := ast.CheckPathConflicts(s.getCompiler(), storage.NonEmpty(ctx, s.store, txn)); len(err) > 0 {
		s.store.Abort(ctx, txn)
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	if err := s.store.Commit(ctx, txn); err != nil {
		writer.ErrorAuto(w, err)
	} else {
		writer.Bytes(w, 204, nil)
	}
}

func (s *Server) v1DataDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	path, ok := storage.ParsePathEscaped("/" + strings.Trim(vars["path"], "/"))
	if !ok {
		writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, "bad path: %v", vars["path"]))
		return
	}

	txn, err := s.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	if err := s.checkPathScope(ctx, txn, path); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	_, err = s.store.Read(ctx, txn, path)
	if err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	if err := s.store.Write(ctx, txn, storage.RemoveOp, path, nil); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	if err := s.store.Commit(ctx, txn); err != nil {
		writer.ErrorAuto(w, err)
	} else {
		writer.Bytes(w, 204, nil)
	}
}

func (s *Server) v1PoliciesDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	includeMetrics := getBoolParam(r.URL, types.ParamPrettyV1, true)

	id, err := url.PathUnescape(vars["path"])
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	m := metrics.New()

	txn, err := s.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	if err := s.checkPolicyIDScope(ctx, txn, id); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	modules, err := s.loadModules(ctx, txn)

	if err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	delete(modules, id)

	c := ast.NewCompiler().SetErrorLimit(s.errLimit)

	m.Timer(metrics.RegoModuleCompile).Start()

	if c.Compile(modules); c.Failed() {
		s.abort(ctx, txn, func() {
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidOperation, types.MsgCompileModuleError).WithASTErrors(c.Errors))
		})
		return
	}

	m.Timer(metrics.RegoModuleCompile).Stop()

	if err := s.store.DeletePolicy(ctx, txn, id); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	if err := s.store.Commit(ctx, txn); err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	response := types.PolicyDeleteResponseV1{}
	if includeMetrics {
		response.Metrics = m.All()
	}

	writer.JSON(w, http.StatusOK, response, pretty)
}

func (s *Server) v1PoliciesGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	path, err := url.PathUnescape(vars["path"])
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)

	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	bs, err := s.store.GetPolicy(ctx, txn, path)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	c := s.getCompiler()

	response := types.PolicyGetResponseV1{
		Result: types.PolicyV1{
			ID:  path,
			Raw: string(bs),
			AST: c.Modules[path],
		},
	}

	writer.JSON(w, http.StatusOK, response, pretty)
}

func (s *Server) v1PoliciesList(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)

	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	policies := []types.PolicyV1{}
	c := s.getCompiler()

	// Only return policies from the store, the compiler
	// may contain additional partially compiled modules.
	ids, err := s.store.ListPolicies(ctx, txn)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}
	for _, id := range ids {
		bs, err := s.store.GetPolicy(ctx, txn, id)
		if err != nil {
			writer.ErrorAuto(w, err)
			return
		}
		policy := types.PolicyV1{
			ID:  id,
			Raw: string(bs),
			AST: c.Modules[id],
		}
		policies = append(policies, policy)
	}

	response := types.PolicyListResponseV1{
		Result: policies,
	}

	writer.JSON(w, http.StatusOK, response, pretty)
}

func (s *Server) v1PoliciesPut(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	path, err := url.PathUnescape(vars["path"])
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	includeMetrics := getBoolParam(r.URL, types.ParamMetricsV1, true)
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	m := metrics.New()

	m.Timer("server_read_bytes").Start()

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		return
	}

	m.Timer("server_read_bytes").Stop()

	txn, err := s.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	if bs, err := s.store.GetPolicy(ctx, txn, path); err != nil {
		if !storage.IsNotFound(err) {
			s.abortAuto(ctx, txn, w, err)
			return
		}
	} else if bytes.Equal(buf, bs) {
		s.store.Abort(ctx, txn)
		response := types.PolicyPutResponseV1{}
		if includeMetrics {
			response.Metrics = m.All()
		}
		writer.JSON(w, http.StatusOK, response, pretty)
		return
	}

	m.Timer(metrics.RegoModuleParse).Start()
	parsedMod, err := ast.ParseModule(path, string(buf))
	m.Timer(metrics.RegoModuleParse).Stop()

	if err != nil {
		s.store.Abort(ctx, txn)
		switch err := err.(type) {
		case ast.Errors:
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgCompileModuleError).WithASTErrors(err))
		default:
			writer.ErrorString(w, http.StatusBadRequest, types.CodeInvalidParameter, err)
		}
		return
	}

	if parsedMod == nil {
		s.store.Abort(ctx, txn)
		writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, "empty module"))
		return
	}

	if err := s.checkPolicyPackageScope(ctx, txn, parsedMod.Package); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	modules, err := s.loadModules(ctx, txn)
	if err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	modules[path] = parsedMod

	c := ast.NewCompiler().
		SetErrorLimit(s.errLimit).
		WithPathConflictsCheck(storage.NonEmpty(ctx, s.store, txn)).
		WithEnablePrintStatements(s.manager.EnablePrintStatements())

	m.Timer(metrics.RegoModuleCompile).Start()

	if c.Compile(modules); c.Failed() {
		s.abort(ctx, txn, func() {
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgCompileModuleError).WithASTErrors(c.Errors))
		})
		return
	}

	m.Timer(metrics.RegoModuleCompile).Stop()

	if err := s.store.UpsertPolicy(ctx, txn, path, buf); err != nil {
		s.abortAuto(ctx, txn, w, err)
		return
	}

	if err := s.store.Commit(ctx, txn); err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	response := types.PolicyPutResponseV1{}

	if includeMetrics {
		response.Metrics = m.All()
	}

	writer.JSON(w, http.StatusOK, response, pretty)
}

func (s *Server) v1QueryGet(w http.ResponseWriter, r *http.Request) {
	m := metrics.New()

	decisionID := s.generateDecisionID()

	ctx := r.Context()
	values := r.URL.Query()

	qStrs := values[types.ParamQueryV1]
	if len(qStrs) == 0 {
		writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, "missing parameter 'q'"))
		return
	}
	qStr := qStrs[len(qStrs)-1]

	parsedQuery, err := validateQuery(qStr)
	if err != nil {
		switch err := err.(type) {
		case ast.Errors:
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgParseQueryError).WithASTErrors(err))
			return
		default:
			writer.ErrorAuto(w, err)
			return
		}
	}

	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	explainMode := getExplain(r.URL.Query()["explain"], types.ExplainOffV1)
	includeMetrics := getBoolParam(r.URL, types.ParamMetricsV1, true)
	includeInstrumentation := getBoolParam(r.URL, types.ParamInstrumentV1, true)

	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	br, err := getRevisions(ctx, s.store, txn)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	results, err := s.execQuery(ctx, r, br, txn, decisionID, parsedQuery, nil, m, explainMode, includeMetrics, includeInstrumentation, pretty)
	if err != nil {
		switch err := err.(type) {
		case ast.Errors:
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgCompileQueryError).WithASTErrors(err))
		default:
			writer.ErrorAuto(w, err)
		}
		return
	}

	writer.JSON(w, 200, results, pretty)
}

func (s *Server) v1QueryPost(w http.ResponseWriter, r *http.Request) {
	m := metrics.New()

	decisionID := s.generateDecisionID()

	ctx := r.Context()

	var request types.QueryRequestV1
	err := util.NewJSONDecoder(r.Body).Decode(&request)
	if err != nil {
		writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, "error(s) occurred while decoding request: %v", err.Error()))
		return
	}
	qStr := request.Query
	parsedQuery, err := validateQuery(qStr)
	if err != nil {
		switch err := err.(type) {
		case ast.Errors:
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgParseQueryError).WithASTErrors(err))
			return
		default:
			writer.ErrorAuto(w, err)
			return
		}
	}

	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	explainMode := getExplain(r.URL.Query()["explain"], types.ExplainOffV1)
	includeMetrics := getBoolParam(r.URL, types.ParamMetricsV1, true)
	includeInstrumentation := getBoolParam(r.URL, types.ParamInstrumentV1, true)

	var input ast.Value

	if request.Input != nil {
		input, err = ast.InterfaceToValue(*request.Input)
		if err != nil {
			writer.ErrorAuto(w, err)
			return
		}
	}

	txn, err := s.store.NewTransaction(ctx)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	defer s.store.Abort(ctx, txn)

	br, err := getRevisions(ctx, s.store, txn)
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	results, err := s.execQuery(ctx, r, br, txn, decisionID, parsedQuery, input, m, explainMode, includeMetrics, includeInstrumentation, pretty)
	if err != nil {
		switch err := err.(type) {
		case ast.Errors:
			writer.Error(w, http.StatusBadRequest, types.NewErrorV1(types.CodeInvalidParameter, types.MsgCompileQueryError).WithASTErrors(err))
		default:
			writer.ErrorAuto(w, err)
		}
		return
	}

	writer.JSON(w, 200, results, pretty)
}

func (s *Server) v1ConfigGet(w http.ResponseWriter, r *http.Request) {
	pretty := getBoolParam(r.URL, types.ParamPrettyV1, true)
	result, err := s.manager.Config.ActiveConfig()
	if err != nil {
		writer.ErrorAuto(w, err)
		return
	}

	var resp types.ConfigResponseV1
	resp.Result = &result

	writer.JSON(w, http.StatusOK, resp, pretty)
}

func (s *Server) checkPolicyIDScope(ctx context.Context, txn storage.Transaction, id string) error {

	bs, err := s.store.GetPolicy(ctx, txn, id)
	if err != nil {
		return err
	}

	module, err := ast.ParseModule(id, string(bs))
	if err != nil {
		return err
	}

	return s.checkPolicyPackageScope(ctx, txn, module.Package)
}

func (s *Server) checkPolicyPackageScope(ctx context.Context, txn storage.Transaction, pkg *ast.Package) error {

	path, err := pkg.Path.Ptr()
	if err != nil {
		return err
	}

	spath, ok := storage.ParsePathEscaped("/" + path)
	if !ok {
		return types.BadRequestErr("invalid package path: cannot determine scope")
	}

	return s.checkPathScope(ctx, txn, spath)
}

func (s *Server) checkPathScope(ctx context.Context, txn storage.Transaction, path storage.Path) error {

	names, err := bundle.ReadBundleNamesFromStore(ctx, s.store, txn)
	if err != nil {
		if !storage.IsNotFound(err) {
			return err
		}
		return nil
	}

	bundleRoots := map[string][]string{}
	for _, name := range names {
		roots, err := bundle.ReadBundleRootsFromStore(ctx, s.store, txn, name)
		if err != nil && !storage.IsNotFound(err) {
			return err
		}
		bundleRoots[name] = roots
	}

	spath := strings.Trim(path.String(), "/")

	if spath == "" && len(bundleRoots) > 0 {
		return types.BadRequestErr("can't write to document root with bundle roots configured")
	}

	spathParts := strings.Split(spath, "/")

	for name, roots := range bundleRoots {
		if roots == nil {
			return types.BadRequestErr(fmt.Sprintf("all paths owned by bundle %q", name))
		}
		for _, root := range roots {
			if root == "" {
				return types.BadRequestErr(fmt.Sprintf("all paths owned by bundle %q", name))
			}
			if isPathOwned(spathParts, strings.Split(root, "/")) {
				return types.BadRequestErr(fmt.Sprintf("path %v is owned by bundle %q", spath, name))
			}
		}
	}

	return nil
}

func (s *Server) getDecisionLogger(br bundleRevisions) (logger decisionLogger) {
	// For backwards compatibility use `revision` as needed.
	if s.hasLegacyBundle(br) {
		logger.revision = br.LegacyRevision
	} else {
		logger.revisions = br.Revisions
	}
	logger.logger = s.logger
	logger.buffer = s.buffer
	return logger
}

func (s *Server) getExplainResponse(explainMode types.ExplainModeV1, trace []*topdown.Event, pretty bool) (explanation types.TraceV1) {
	switch explainMode {
	case types.ExplainNotesV1:
		var err error
		explanation, err = types.NewTraceV1(lineage.Notes(trace), pretty)
		if err != nil {
			break
		}
	case types.ExplainFailsV1:
		var err error
		explanation, err = types.NewTraceV1(lineage.Fails(trace), pretty)
		if err != nil {
			break
		}
	case types.ExplainFullV1:
		var err error
		explanation, err = types.NewTraceV1(trace, pretty)
		if err != nil {
			break
		}
	}
	return explanation
}

func (s *Server) abort(ctx context.Context, txn storage.Transaction, finish func()) {
	s.store.Abort(ctx, txn)
	finish()
}

func (s *Server) abortAuto(ctx context.Context, txn storage.Transaction, w http.ResponseWriter, err error) {
	s.abort(ctx, txn, func() { writer.ErrorAuto(w, err) })
}

func (s *Server) loadModules(ctx context.Context, txn storage.Transaction) (map[string]*ast.Module, error) {

	ids, err := s.store.ListPolicies(ctx, txn)
	if err != nil {
		return nil, err
	}

	modules := make(map[string]*ast.Module, len(ids))

	for _, id := range ids {
		bs, err := s.store.GetPolicy(ctx, txn, id)
		if err != nil {
			return nil, err
		}

		parsed, err := ast.ParseModule(id, string(bs))
		if err != nil {
			return nil, err
		}

		modules[id] = parsed
	}

	return modules, nil
}

func (s *Server) getCompiler() *ast.Compiler {
	return s.manager.GetCompiler()
}

func (s *Server) makeRego(ctx context.Context, partial bool, txn storage.Transaction, input ast.Value, urlPath string, m metrics.Metrics, instrument bool, tracer topdown.QueryTracer, opts []func(*rego.Rego)) (*rego.Rego, error) {
	queryPath := stringPathToDataRef(urlPath).String()

	opts = append(
		opts,
		rego.Transaction(txn),
		rego.Query(queryPath),
		rego.ParsedInput(input),
		rego.Metrics(m),
		rego.QueryTracer(tracer),
		rego.Instrument(instrument),
		rego.Runtime(s.runtime),
		rego.UnsafeBuiltins(unsafeBuiltinsMap),
		rego.PrintHook(s.manager.PrintHook()),
	)

	if partial {
		// pick a namespace for the query (path), doesn't really matter what it is
		// as long as it is unique for each path.
		namespace := fmt.Sprintf("partial[`%s`]", urlPath)
		s.mtx.Lock()
		defer s.mtx.Unlock()
		pr, ok := s.partials[queryPath]
		if !ok {
			peopts := append(opts, rego.PartialNamespace(namespace))
			r := rego.New(peopts...)
			var err error
			pr, err = r.PartialResult(ctx)
			if err != nil {
				if !rego.IsPartialEvaluationNotEffectiveErr(err) {
					return nil, err
				}
				return rego.New(opts...), nil
			}
			s.partials[queryPath] = pr
		}
		return pr.Rego(opts...), nil
	}

	return rego.New(opts...), nil
}

func (s *Server) prepareV1PatchSlice(root string, ops []types.PatchV1) (result []patchImpl, err error) {

	root = "/" + strings.Trim(root, "/")

	for _, op := range ops {

		impl := patchImpl{
			value: op.Value,
		}

		// Map patch operation.
		switch op.Op {
		case "add":
			impl.op = storage.AddOp
		case "remove":
			impl.op = storage.RemoveOp
		case "replace":
			impl.op = storage.ReplaceOp
		default:
			return nil, types.BadPatchOperationErr(op.Op)
		}

		// Construct patch path.
		path := strings.Trim(op.Path, "/")
		if len(path) > 0 {
			if root == "/" {
				path = root + path
			} else {
				path = root + "/" + path
			}
		} else {
			path = root
		}

		var ok bool
		impl.path, ok = parsePatchPathEscaped(path)
		if !ok {
			return nil, types.BadPatchPathErr(op.Path)
		}

		result = append(result, impl)
	}

	return result, nil
}

func (s *Server) generateDecisionID() string {
	if s.decisionIDFactory != nil {
		return s.decisionIDFactory()
	}
	return ""
}

func (s *Server) getProvenance(br bundleRevisions) *types.ProvenanceV1 {

	p := &types.ProvenanceV1{
		Version:   version.Version,
		Vcs:       version.Vcs,
		Timestamp: version.Timestamp,
		Hostname:  version.Hostname,
	}

	// For backwards compatibility, if the bundles are using the old
	// style config we need to fill in the older `Revision` field.
	// Otherwise use the newer `Bundles` keyword.
	if s.hasLegacyBundle(br) {
		p.Revision = br.LegacyRevision
	} else {
		p.Bundles = map[string]types.ProvenanceBundleV1{}
		for name, revision := range br.Revisions {
			p.Bundles[name] = types.ProvenanceBundleV1{Revision: revision}
		}
	}

	return p
}

func (s *Server) hasLegacyBundle(br bundleRevisions) bool {
	bp := bundlePlugin.Lookup(s.manager)
	return br.LegacyRevision != "" || (bp != nil && !bp.Config().IsMultiBundle())
}

func (s *Server) generateDefaultDecisionPath() string {
	// Assume the path is safe to transition back to a url
	p, _ := s.manager.Config.DefaultDecisionRef().Ptr()
	return p
}

func isPathOwned(path, root []string) bool {
	for i := 0; i < len(path) && i < len(root); i++ {
		if path[i] != root[i] {
			return false
		}
	}
	return true
}

func (s *Server) updateCacheConfig(cacheConfig *iCache.Config) {
	s.interQueryBuiltinCache.UpdateConfig(cacheConfig)
}

// parsePatchPathEscaped returns a new path for the given escaped str.
// This is based on storage.ParsePathEscaped so will do URL unescaping of
// the provided str for backwards compatibility, but also handles the
// specific escape strings defined in RFC 6901 (JSON Pointer) because
// that's what's mandated by RFC 6902 (JSON Patch).
func parsePatchPathEscaped(str string) (path storage.Path, ok bool) {
	path, ok = storage.ParsePathEscaped(str)
	if !ok {
		return
	}
	for i := range path {
		// RFC 6902 section 4: "[The "path" member's] value is a string containing
		// a JSON-Pointer value [RFC6901] that references a location within the
		// target document (the "target location") where the operation is performed."
		//
		// RFC 6901 section 3: "Because the characters '~' (%x7E) and '/' (%x2F)
		// have special meanings in JSON Pointer, '~' needs to be encoded as '~0'
		// and '/' needs to be encoded as '~1' when these characters appear in a
		// reference token."

		// RFC 6901 section 4: "Evaluation of each reference token begins by
		// decoding any escaped character sequence.  This is performed by first
		// transforming any occurrence of the sequence '~1' to '/', and then
		// transforming any occurrence of the sequence '~0' to '~'.  By performing
		// the substitutions in this order, an implementation avoids the error of
		// turning '~01' first into '~1' and then into '/', which would be
		// incorrect (the string '~01' correctly becomes '~1' after transformation)."
		path[i] = strings.Replace(path[i], "~1", "/", -1)
		path[i] = strings.Replace(path[i], "~0", "~", -1)
	}
	return
}

func stringPathToDataRef(s string) (r ast.Ref) {
	result := ast.Ref{ast.DefaultRootDocument}
	result = append(result, stringPathToRef(s)...)
	return result
}

func stringPathToRef(s string) (r ast.Ref) {
	if len(s) == 0 {
		return r
	}
	p := strings.Split(s, "/")
	for _, x := range p {
		if x == "" {
			continue
		}
		if y, err := url.PathUnescape(x); err == nil {
			x = y
		}
		i, err := strconv.Atoi(x)
		if err != nil {
			r = append(r, ast.StringTerm(x))
		} else {
			r = append(r, ast.IntNumberTerm(i))
		}
	}
	return r
}

func validateQuery(query string) (ast.Body, error) {

	var body ast.Body
	body, err := ast.ParseBody(query)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getBoolParam(url *url.URL, name string, ifEmpty bool) bool {

	p, ok := url.Query()[name]
	if !ok {
		return false
	}

	// Query params w/o values are represented as slice (of len 1) with an
	// empty string.
	if len(p) == 1 && p[0] == "" {
		return ifEmpty
	}

	for _, x := range p {
		if strings.ToLower(x) == "true" {
			return true
		}
	}

	return false
}

func getStringSliceParam(url *url.URL, name string) []string {

	p, ok := url.Query()[name]
	if !ok {
		return nil
	}

	// Query params w/o values are represented as slice (of len 1) with an
	// empty string.
	if len(p) == 1 && p[0] == "" {
		return nil
	}

	return p
}

func getExplain(p []string, zero types.ExplainModeV1) types.ExplainModeV1 {
	for _, x := range p {
		switch x {
		case string(types.ExplainNotesV1):
			return types.ExplainNotesV1
		case string(types.ExplainFullV1):
			return types.ExplainFullV1
		}
	}
	return zero
}

func readInputV0(r *http.Request) (ast.Value, error) {

	parsed, ok := authorizer.GetBodyOnContext(r.Context())
	if ok {
		return ast.InterfaceToValue(parsed)
	}

	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	bs = bytes.TrimSpace(bs)
	if len(bs) == 0 {
		return nil, nil
	}

	var x interface{}

	if strings.Contains(r.Header.Get("Content-Type"), "yaml") {
		if err := util.Unmarshal(bs, &x); err != nil {
			return nil, err
		}
	} else if err := util.UnmarshalJSON(bs, &x); err != nil {
		return nil, err
	}

	return ast.InterfaceToValue(x)
}

func readInputGetV1(str string) (ast.Value, error) {
	var input interface{}
	if err := util.UnmarshalJSON([]byte(str), &input); err != nil {
		return nil, errors.Wrapf(err, "parameter contains malformed input document")
	}
	return ast.InterfaceToValue(input)
}

func readInputPostV1(r *http.Request) (ast.Value, error) {

	parsed, ok := authorizer.GetBodyOnContext(r.Context())
	if ok {
		if obj, ok := parsed.(map[string]interface{}); ok {
			if input, ok := obj["input"]; ok {
				return ast.InterfaceToValue(input)
			}
		}
		return nil, nil
	}

	bs, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return nil, err
	}

	if len(bs) > 0 {

		ct := r.Header.Get("Content-Type")

		var request types.DataRequestV1

		// There is no standard for yaml mime-type so we just look for
		// anything related
		if strings.Contains(ct, "yaml") {
			if err := util.Unmarshal(bs, &request); err != nil {
				return nil, errors.Wrapf(err, "body contains malformed input document")
			}
		} else if err := util.UnmarshalJSON(bs, &request); err != nil {
			return nil, errors.Wrapf(err, "body contains malformed input document")
		}

		if request.Input == nil {
			return nil, nil
		}

		return ast.InterfaceToValue(*request.Input)
	}

	return nil, nil
}

type compileRequest struct {
	Query    ast.Body
	Input    ast.Value
	Unknowns []*ast.Term
}

func readInputCompilePostV1(r io.ReadCloser) (*compileRequest, *types.ErrorV1) {

	var request types.CompileRequestV1

	err := util.NewJSONDecoder(r).Decode(&request)
	if err != nil {
		return nil, types.NewErrorV1(types.CodeInvalidParameter, "error(s) occurred while decoding request: %v", err.Error())
	}

	query, err := ast.ParseBody(request.Query)
	if err != nil {
		switch err := err.(type) {
		case ast.Errors:
			return nil, types.NewErrorV1(types.CodeInvalidParameter, types.MsgParseQueryError).WithASTErrors(err)
		default:
			return nil, types.NewErrorV1(types.CodeInvalidParameter, "%v: %v", types.MsgParseQueryError, err)
		}
	} else if len(query) == 0 {
		return nil, types.NewErrorV1(types.CodeInvalidParameter, "missing required 'query' value")
	}

	var input ast.Value
	if request.Input != nil {
		input, err = ast.InterfaceToValue(*request.Input)
		if err != nil {
			return nil, types.NewErrorV1(types.CodeInvalidParameter, "error(s) occurred while converting input: %v", err)
		}
	}

	var unknowns []*ast.Term
	if request.Unknowns != nil {
		unknowns = make([]*ast.Term, len(*request.Unknowns))
		for i, s := range *request.Unknowns {
			unknowns[i], err = ast.ParseTerm(s)
			if err != nil {
				return nil, types.NewErrorV1(types.CodeInvalidParameter, "error(s) occurred while parsing unknowns: %v", err)
			}
		}
	}

	result := &compileRequest{
		Query:    query,
		Input:    input,
		Unknowns: unknowns,
	}

	return result, nil
}

var indexHTML, _ = template.New("index").Parse(`
<html>
<head>
<script type="text/javascript">
function query() {
	params = {
		'query': document.getElementById("query").value,
	}
	if (document.getElementById("input").value !== "") {
		try {
			params["input"] = JSON.parse(document.getElementById("input").value);
		} catch (e) {
			document.getElementById("result").innerHTML = e;
			return;
		}
	}
	body = JSON.stringify(params);
	opts = {
		'method': 'POST',
		'body': body,
	}
	fetch(new Request('/v1/query', opts))
		.then(resp => resp.json())
		.then(json => {
			str = JSON.stringify(json, null, 2);
			document.getElementById("result").innerHTML = str;
		});
}
</script>
</head>
</body>
<pre>
 ________      ________    ________
|\   __  \    |\   __  \  |\   __  \
\ \  \|\  \   \ \  \|\  \ \ \  \|\  \
 \ \  \\\  \   \ \   ____\ \ \   __  \
  \ \  \\\  \   \ \  \___|  \ \  \ \  \
   \ \_______\   \ \__\      \ \__\ \__\
    \|_______|    \|__|       \|__|\|__|
</pre>
Open Policy Agent - An open source project to policy-enable your service.<br>
<br>
Version: {{ .Version }}<br>
Build Commit: {{ .BuildCommit }}<br>
Build Timestamp: {{ .BuildTimestamp }}<br>
Build Hostname: {{ .BuildHostname }}<br>
<br>
Query:<br>
<textarea rows="10" cols="50" id="query"></textarea><br>
<br>Input Data (JSON):<br>
<textarea rows="10" cols="50" id="input"></textarea><br>
<br><button onclick="query()">Submit</button>
<pre><div id="result"></div></pre>
</body>
</html>
`)

type decisionLogger struct {
	revisions map[string]string
	revision  string // Deprecated: Use `revisions` instead.
	logger    func(context.Context, *Info) error
	buffer    Buffer
}

func (l decisionLogger) Log(ctx context.Context, txn storage.Transaction, decisionID, remoteAddr, path string, query string, goInput *interface{}, astInput ast.Value, goResults *interface{}, err error, m metrics.Metrics) error {

	bundles := map[string]BundleInfo{}
	for name, rev := range l.revisions {
		bundles[name] = BundleInfo{Revision: rev}
	}

	info := &Info{
		Txn:        txn,
		Revision:   l.revision,
		Bundles:    bundles,
		Timestamp:  time.Now().UTC(),
		DecisionID: decisionID,
		RemoteAddr: remoteAddr,
		Path:       path,
		Query:      query,
		Input:      goInput,
		InputAST:   astInput,
		Results:    goResults,
		Error:      err,
		Metrics:    m,
	}

	if l.logger != nil {
		if err := l.logger(ctx, info); err != nil {
			return errors.Wrap(err, "decision_logs")
		}
	}

	if l.buffer != nil {
		l.buffer.Push(info)
	}

	return nil
}

type patchImpl struct {
	path  storage.Path
	op    storage.PatchOp
	value interface{}
}

func parseURL(s string, useHTTPSByDefault bool) (*url.URL, error) {
	if !strings.Contains(s, "://") {
		scheme := "http://"
		if useHTTPSByDefault {
			scheme = "https://"
		}
		s = scheme + s
	}
	return url.Parse(s)
}
