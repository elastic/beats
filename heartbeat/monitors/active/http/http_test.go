package http

import (
	"github.com/stretchr/testify/assert"
	"time"
	"net"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/heartbeat/monitors"
)

type ServerHandler struct {
	Invoked bool
}

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Invoked = true
}

type ProxyHandler struct {
	Invoked bool
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Invoked = true

	// proxy implementation from https://medium.com/@mlowicki/6a51c2f2c38c
	if r.Method == http.MethodConnect {
		handleTunneling(w, r)
	} else {
		handleHTTP(w, r)
	}
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
    destConnection, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
    hijacker, ok := w.(http.Hijacker)
    if !ok {
        http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
        return
    }
    clientConnection, _, err := hijacker.Hijack()
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
    }
    go transfer(destConnection, clientConnection)
    go transfer(clientConnection, destConnection)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
    resp, err := http.DefaultTransport.RoundTrip(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    defer resp.Body.Close()
    copyHeader(w.Header(), resp.Header)
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
    defer destination.Close()
    defer source.Close()
    io.Copy(destination, source)
}

func copyHeader(dst, src http.Header) {
    for k, vv := range src {
        for _, v := range vv {
            dst.Add(k, v)
        }
    }
}

func runAndAssert(t *testing.T, rawCfg map[string]interface{}) {
	// create an http monitor job, run it and make sure that the beat event indicates that the host is accessible
	info := monitors.Info{}
	cfg, _ := common.NewConfigFrom(rawCfg)
	jobs, err := create(info, cfg)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(jobs))

	event, _, err := jobs[0].Run()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Event %v", event)

	monitorStatus, err := event.GetValue("monitor.status")
	if err != nil {
		t.Fatal(err)
	}

	httpStatus, err := event.GetValue("http.response.status")
	if err != nil {
		t.Fatal(err)
	}

	hasError, err := event.HasKey("error")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "up", monitorStatus)
	assert.Equal(t, 200, httpStatus)
	assert.False(t, hasError)
}

func TestSSLVerificationModeToNone(t *testing.T) {
	serverHandler := &ServerHandler{}
	server := httptest.NewTLSServer(serverHandler)
	defer server.Close()

	rawCfg := map[string]interface{}{
		"urls": []string{
			server.URL,
		},
		"ssl": map[string]interface{}{
			"verification_mode": "none",
		},
	}

	runAndAssert(t, rawCfg)
	assert.True(t, serverHandler.Invoked)
}
func TestHTTPProxyForHTTPSEndpoint(t *testing.T) {
	serverHandler := &ServerHandler{}
	server := httptest.NewTLSServer(serverHandler)
	defer server.Close()

	proxyHandler := &ProxyHandler{}
	proxyServer := httptest.NewServer(proxyHandler)
	defer proxyServer.Close()

	rawCfg := map[string]interface{}{
		"urls": []string{
			server.URL,
		},
		"proxy_url": proxyServer.URL,
		"ssl": map[string]interface{}{
			"verification_mode": "none",
		},
	}

	runAndAssert(t, rawCfg)
	assert.True(t, proxyHandler.Invoked)
}

func TestHTTPSProxyForHTTPSEndpoint(t *testing.T) {
	serverHandler := &ServerHandler{}
	server := httptest.NewTLSServer(serverHandler)
	defer server.Close()

	proxyHandler := &ProxyHandler{}
	proxyServer := httptest.NewTLSServer(proxyHandler)
	defer proxyServer.Close()

	rawCfg := map[string]interface{}{
		"urls": []string{
			server.URL,
		},
		"proxy_url": proxyServer.URL,
		"ssl": map[string]interface{}{
			"verification_mode": "none",
		},
	}

	runAndAssert(t, rawCfg)
	assert.True(t, proxyHandler.Invoked)
}

func TestHTTPProxyConfiguredViaEnv(t *testing.T) {
	t.Skip("This test cannot be executed until https://github.com/golang/go/issues/22079 is merged, as the proxy settings are being cached and there is no way to refresh them.")

	serverHandler := &ServerHandler{}
	server := httptest.NewServer(serverHandler)
	defer server.Close()

	proxyHandler := &ProxyHandler{}
	proxyServer := httptest.NewServer(proxyHandler)
	defer proxyServer.Close()

	os.Setenv("HTTP_PROXY", proxyServer.URL)

	rawCfg := map[string]interface{}{
		"urls": []string{
			// cannot use the local server, as http.ProxyFromEnvironment wont use a proxy for localhost
			//server.URL,
			"http://example.com/",
		},
	}

	runAndAssert(t, rawCfg)
	assert.True(t, proxyHandler.Invoked)
}
