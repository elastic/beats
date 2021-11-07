package test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/bundle"
	"github.com/open-policy-agent/opa/compile"
)

// MockBundle sets a bundle named file on the test server containing the given
// policies.
func MockBundle(file string, policies map[string]string) func(*Server) error {
	return func(s *Server) error {
		s.bundles[file] = policies
		return nil
	}
}

// Ready provides a channel that the server will use to gate readiness. The
// caller can provide this channel to prevent the server from becoming ready.
// The server will response with HTTP 500 responses until ready. The caller
// should close the channel to indicate readiness.
func Ready(ch chan struct{}) func(*Server) error {
	return func(s *Server) error {
		s.ready = ch
		return nil
	}
}

// Server provides a mock HTTP server for testing the SDK and integrations.
type Server struct {
	server  *httptest.Server
	ready   chan struct{}
	bundles map[string]map[string]string
}

// MustNewServer returns a new Server for test purposes or panics if an error occurs.
func MustNewServer(opts ...func(*Server) error) *Server {
	s, err := NewServer(opts...)
	if err != nil {
		panic(err)
	}
	return s
}

// NewServer returns a new Server for test purposes.
func NewServer(opts ...func(*Server) error) (*Server, error) {
	s := &Server{
		bundles: map[string]map[string]string{},
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if s.ready == nil {
		s.ready = make(chan struct{})
		close(s.ready)
	}
	s.server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s, nil
}

// WithTestBundle adds a bundle to the server at the specified endpoint.
func (s *Server) WithTestBundle(endpoint string, policies map[string]string) *Server {
	s.bundles[endpoint] = policies
	return s
}

// Stop stops the test server.
func (s *Server) Stop() {
	s.server.Close()
}

// URL returns the base URL of the server.
func (s *Server) URL() string {
	return s.server.URL
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {

	select {
	case <-s.ready:
	default:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/bundles") {
		s.handleBundles(w, r)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Server) handleBundles(w http.ResponseWriter, r *http.Request) {

	// Return 404 if bundle path does not exist.
	b, ok := s.bundles[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Prepare the modules to include in the bundle. Sort them so bundles are deterministic.
	var modules []bundle.ModuleFile
	for url, str := range b {
		module, err := ast.ParseModule(url, str)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		modules = append(modules, bundle.ModuleFile{
			URL:    url,
			Parsed: module,
		})
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].URL < modules[j].URL
	})

	// Compile the bundle out into a buffer
	buf := bytes.NewBuffer(nil)
	err := compile.New().WithOutput(buf).WithBundle(&bundle.Bundle{
		Data:    map[string]interface{}{},
		Modules: modules,
	}).Build(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	// Write out the bundle
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, buf)
}
