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

package hbtest

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"math/bits"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v8/heartbeat/monitors/active/dialchain/tlsmeta"
	"github.com/elastic/beats/v8/libbeat/common"

	"github.com/elastic/beats/v8/heartbeat/hbtestllext"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v8/libbeat/common/x509util"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/validator"
)

// HelloWorldBody is the body of the HelloWorldHandler.
const HelloWorldBody = "hello, world!"

// HelloWorldHandler is a handler for an http server that returns
// HelloWorldBody and a 200 OK status.
func HelloWorldHandler(status int) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if status >= 301 && status <= 303 {
				w.Header().Set("Location", "/somewhere")
			}
			w.WriteHeader(status)
			io.WriteString(w, HelloWorldBody)
		},
	)
}

// SizedResponseHandler responds with 200 to any request with a body
// exactly the size of the `bytes` argument, where each byte is the
// character 'x'
func SizedResponseHandler(bytes int) http.HandlerFunc {
	var body strings.Builder
	for i := 0; i < bytes; i++ {
		body.WriteString("x")
	}

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			io.WriteString(w, body.String())
		},
	)
}

func CustomResponseHandler(body []byte, status int, extraHeaders map[string]string) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			for key, val := range extraHeaders {
				w.Header().Add(key, val)
			}
			w.WriteHeader(status)
			w.Write(body)
		},
	)
}

// RedirectHandler redirects the paths at the keys in the redirectingPaths map to the locations in their values.
// For paths not in the redirectingPaths map it returns a 200 response with the given body.
func RedirectHandler(redirectingPaths map[string]string, body string) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			url, _ := url.Parse(r.RequestURI)
			redirectTarget, isRedirect := redirectingPaths[url.Path]
			if isRedirect {
				w.Header().Add("Location", redirectTarget)
				w.WriteHeader(302)
			} else {
				w.WriteHeader(200)
				io.WriteString(w, body)
			}
		})
}

// ServerPort takes an httptest.Server and returns its port as a uint16.
func ServerPort(server *httptest.Server) (uint16, error) {
	u, err := url.Parse(server.URL)
	if err != nil {
		return 0, err
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		return 0, err
	}
	return uint16(p), nil
}

// TLSChecks validates the given x509 cert at the given position.
func TLSChecks(chainIndex, certIndex int, certificate *x509.Certificate) validator.Validator {
	expected := common.MapStr{}
	// This function is well tested independently, so we just test that things match up here.
	tlsmeta.AddTLSMetadata(expected, tls.ConnectionState{
		Version:           tls.VersionTLS13,
		HandshakeComplete: true,
		CipherSuite:       tls.TLS_AES_128_GCM_SHA256,
		ServerName:        certificate.Subject.CommonName,
		PeerCertificates:  []*x509.Certificate{certificate},
	}, time.Duration(1))

	expected.Put("tls.rtt.handshake.us", isdef.IsDuration)

	// Generally, the exact cipher will match, but on windows 7 32bit this is not true!
	// We don't actually care about the exact cipher matching, since we're not testing the TLS
	// implementation, we trust go there, just that most of the metadata is present
	if runtime.GOOS == "windows" && bits.UintSize == 32 {
		expected.Put("tls.cipher", isdef.IsString)
	}

	return lookslike.MustCompile(expected)
}

func TLSCertChecks(certificate *x509.Certificate) validator.Validator {
	expected := common.MapStr{}
	tlsmeta.AddCertMetadata(expected, []*x509.Certificate{certificate})
	return lookslike.MustCompile(expected)
}

// BaseChecks creates a skima.Validator that represents the "monitor" field present
// in all heartbeat events.
// If IP is set to "" this will check that the field is not present
func BaseChecks(ip string, status string, typ string) validator.Validator {
	var ipCheck isdef.IsDef
	if len(ip) > 0 {
		ipCheck = isdef.IsEqual(ip)
	} else {
		ipCheck = isdef.Optional(isdef.IsEqual(ip))
	}

	return lookslike.Compose(
		lookslike.MustCompile(map[string]interface{}{
			"monitor": map[string]interface{}{
				"ip":          ipCheck,
				"status":      status,
				"duration.us": isdef.IsDuration,
				"id":          isdef.IsNonEmptyString,
				"name":        isdef.IsString,
				"type":        typ,
				"check_group": isdef.IsString,
			},
		}),
		hbtestllext.MonitorTimespanValidator,
	)
}

// SummaryChecks validates the "summary" field and its subfields.
func SummaryChecks(up int, down int) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"summary": map[string]interface{}{
			"up":   uint16(up),
			"down": uint16(down),
		},
	})
}

// ResolveChecks returns a lookslike matcher for the 'resolve' fields.
func ResolveChecks(ip string) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"resolve": map[string]interface{}{
			"ip":     ip,
			"rtt.us": isdef.IsDuration,
		},
	})
}

// SimpleURLChecks returns a check for a simple URL
// with only a scheme, host, and port
func SimpleURLChecks(t *testing.T, scheme string, host string, port uint16) validator.Validator {
	hostPort := host
	if port != 0 {
		hostPort = fmt.Sprintf("%s:%d", host, port)
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s", scheme, hostPort))
	require.NoError(t, err)

	return URLChecks(t, u)
}

// URLChecks returns a validator for the given URL's fields
func URLChecks(t *testing.T, u *url.URL) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"url": wrappers.URLFields(u),
	})
}

// ErrorChecks checks the standard heartbeat error hierarchy, which should
// consist of a message (or a lookslike isdef that can match the message) and a type under the error key.
// The message is checked only as a substring since exact string matches can be fragile due to platform differences.
func ErrorChecks(msgSubstr string, errType string) validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{
		"error": map[string]interface{}{
			"message": isdef.IsStringContaining(msgSubstr),
			"type":    errType,
		},
	})
}

func ExpiredCertChecks(cert *x509.Certificate) validator.Validator {
	msg := x509.CertificateInvalidError{Cert: cert, Reason: x509.Expired}.Error()
	return lookslike.Compose(
		ErrorChecks(msg, "io"),
		TLSCertChecks(cert),
	)
}

// RespondingTCPChecks creates a skima.Validator that represents the "tcp" field present
// in all heartbeat events that use a Tcp connection as part of their DialChain
func RespondingTCPChecks() validator.Validator {
	return lookslike.MustCompile(map[string]interface{}{"tcp.rtt.connect.us": isdef.IsDuration})
}

// CertToTempFile takes a certificate and returns an *os.File with a PEM encoded
// x.509 representation of that cert. Note that this takes tls.Certificate
// objects from a server like httptest. This doesn't take x509 certs.
// We never parse the x509 data in this case, we just transpose the bytes.
// This is a little confusing, but is actually less work and less code.
func CertToTempFile(t *testing.T, cert *x509.Certificate) *os.File {
	// Write the certificate to a tempFile. Heartbeat would normally read certs from
	// disk, not memory, so this little bit of extra work is worthwhile
	certFile, err := ioutil.TempFile("", "sslcert")
	require.NoError(t, err)
	certFile.WriteString(x509util.CertToPEMString(cert))
	return certFile
}

func StartHTTPSServer(t *testing.T, tlsCert tls.Certificate) (host string, port string, cert *x509.Certificate, doClose func() error) {
	cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	// No need to start a real server, since this is invalid, we just
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	})
	require.NoError(t, err)

	srv := &http.Server{Handler: HelloWorldHandler(200)}
	go func() {
		srv.Serve(l)
	}()

	host, port, err = net.SplitHostPort(l.Addr().String())
	require.NoError(t, err)
	return host, port, cert, srv.Close
}
