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
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain/tlsmeta"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type requestFactory func() (*http.Request, error)

func newHTTPMonitorHostJob(
	addr string,
	config *Config,
	transport http.RoundTripper,
	enc contentEncoder,
	body []byte,
	validator multiValidator,
) (jobs.Job, error) {

	var reqFactory requestFactory = func() (*http.Request, error) { return buildRequest(addr, config, enc) }

	return jobs.MakeSimpleJob(func(event *beat.Event) error {
		var redirects []string
		client := &http.Client{
			// Trace visited URLs when redirects occur
			CheckRedirect: makeCheckRedirect(config.MaxRedirects, &redirects),
			Transport:     transport,
			Timeout:       config.Transport.Timeout,
		}

		req, err := reqFactory()
		if err != nil {
			return fmt.Errorf("could not make http request: %w", err)
		}

		_, _, err = execPing(event, client, req, body, config.Transport.Timeout, validator, config.Response)
		if len(redirects) > 0 {
			event.PutValue("http.response.redirects", redirects)
		}
		return err
	}), nil
}

func newHTTPMonitorIPsJob(
	config *Config,
	addr string,
	tls *tlscommon.TLSConfig,
	enc contentEncoder,
	body []byte,
	validator multiValidator,
) (jobs.Job, error) {

	var reqFactory requestFactory = func() (*http.Request, error) { return buildRequest(addr, config, enc) }

	hostname, port, err := splitHostnamePort(addr)
	if err != nil {
		return nil, err
	}

	pingFactory := createPingFactory(config, port, tls, reqFactory, body, validator)
	job, err := monitors.MakeByHostJob(hostname, config.Mode, monitors.NewStdResolver(), pingFactory)

	return job, err
}

func createPingFactory(
	config *Config,
	port uint16,
	tls *tlscommon.TLSConfig,
	reqFactory requestFactory,
	body []byte,
	validator multiValidator,
) func(*net.IPAddr) jobs.Job {
	timeout := config.Transport.Timeout

	return monitors.MakePingIPFactory(func(event *beat.Event, ip *net.IPAddr) error {
		req, err := reqFactory()
		if err != nil {
			return fmt.Errorf("could not create http request: %w", err)
		}
		isTLS := req.URL.Scheme == "https"

		addr := net.JoinHostPort(ip.String(), strconv.Itoa(int(port)))
		d := &dialchain.DialerChain{
			Net: dialchain.MakeConstAddrDialer(addr, dialchain.TCPDialer(timeout)),
		}

		// TODO: add socks5 proxy?

		if isTLS {
			d.AddLayer(dialchain.TLSLayer(tls, timeout))
		}

		dialer, err := d.Build(event)
		if err != nil {
			return err
		}

		var (
			writeStart, readStart, writeEnd time.Time
		)
		// Ensure memory consistency for these callbacks.
		// It seems they can be invoked still sometime after the request is done
		cbMutex := sync.Mutex{}

		// We don't support redirects for IP jobs, so this effectively just
		// prevents following redirects in this case, we know that
		// config.MaxRedirects must be zero to even be here
		checkRedirect := makeCheckRedirect(0, nil)
		transport := &SimpleTransport{
			Dialer: dialer,
			OnStartWrite: func() {
				cbMutex.Lock()
				writeStart = time.Now()
				cbMutex.Unlock()
			},
			OnEndWrite: func() {
				cbMutex.Lock()
				writeEnd = time.Now()
				cbMutex.Unlock()
			},
			OnStartRead: func() {
				cbMutex.Lock()
				readStart = time.Now()
				cbMutex.Unlock()
			},
		}
		client := &http.Client{
			CheckRedirect: checkRedirect,
			Timeout:       timeout,
			Transport:     httpcommon.HeaderRoundTripper(transport, map[string]string{"User-Agent": userAgent}),
		}

		_, end, err := execPing(event, client, req, body, timeout, validator, config.Response)
		cbMutex.Lock()
		defer cbMutex.Unlock()

		if !readStart.IsZero() {
			eventext.MergeEventFields(event, common.MapStr{
				"http": common.MapStr{
					"rtt": common.MapStr{
						"write_request":   look.RTT(writeEnd.Sub(writeStart)),
						"response_header": look.RTT(readStart.Sub(writeStart)),
					},
				},
			})
		}
		if !writeStart.IsZero() {
			event.PutValue("http.rtt.validate", look.RTT(end.Sub(writeStart)))
			event.PutValue("http.rtt.content", look.RTT(end.Sub(readStart)))
		}

		return err
	})
}

func buildRequest(addr string, config *Config, enc contentEncoder) (*http.Request, error) {
	method := strings.ToUpper(config.Check.Request.Method)
	request, err := http.NewRequest(method, addr, nil)
	if err != nil {
		return nil, err
	}
	request.Close = true

	if config.Username != "" {
		request.SetBasicAuth(config.Username, config.Password)
	}
	for k, v := range config.Check.Request.SendHeaders {
		// defining the Host header isn't enough. See https://github.com/golang/go/issues/7682
		if k == "Host" {
			request.Host = v
		}

		request.Header.Add(k, v)
	}

	if enc != nil {
		enc.AddHeaders(&request.Header)
	}

	return request, nil
}

func execPing(
	event *beat.Event,
	client *http.Client,
	req *http.Request,
	reqBody []byte,
	timeout time.Duration,
	validator multiValidator,
	responseConfig responseConfig,
) (start, end time.Time, err reason.Reason) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req = attachRequestBody(&ctx, req, reqBody)

	// Send the HTTP request. We don't immediately return on error since
	// we may want to add additional fields to contextualize the error.
	start, resp, errReason := execRequest(client, req)
	// If we have no response object or an error was set there probably was an IO error, we can skip the rest of the logic
	// since that logic is for adding metadata relating to completed HTTP transactions that have errored
	// in other ways
	if resp == nil || errReason != nil {
		if urlErr, ok := errReason.Unwrap().(*url.Error); ok {
			if certErr, ok := urlErr.Err.(x509.CertificateInvalidError); ok {
				tlsmeta.AddCertMetadata(event.Fields, []*x509.Certificate{certErr.Cert})
			}
		}

		return start, time.Now(), errReason
	}

	bodyFields, mimeType, errReason := processBody(resp, responseConfig, validator)

	responseFields := common.MapStr{
		"status_code": resp.StatusCode,
		"body":        bodyFields,
	}

	if mimeType != "" {
		responseFields["mime_type"] = mimeType
	}

	if responseConfig.IncludeHeaders {
		headerFields := common.MapStr{}
		for canonicalHeaderKey, vals := range resp.Header {
			if len(vals) > 1 {
				headerFields[canonicalHeaderKey] = vals
			} else {
				headerFields[canonicalHeaderKey] = vals[0]
			}
		}
		responseFields["headers"] = headerFields
	}

	httpFields := common.MapStr{"response": responseFields}

	eventext.MergeEventFields(event, common.MapStr{"http": httpFields})

	// Mark the end time as now, since we've finished downloading
	end = time.Now()

	// Enrich event with TLS information when available. This is useful when connecting to an HTTPS server through
	// a proxy.
	if resp.TLS != nil {
		tlsFields := common.MapStr{}
		tlsmeta.AddTLSMetadata(tlsFields, *resp.TLS, tlsmeta.UnknownTLSHandshakeDuration)
		eventext.MergeEventFields(event, tlsFields)
	}

	// Add total HTTP RTT
	eventext.MergeEventFields(event, common.MapStr{"http": common.MapStr{
		"rtt": common.MapStr{
			"total": look.RTT(end.Sub(start)),
		},
	}})

	return start, end, errReason
}

func attachRequestBody(ctx *context.Context, req *http.Request, body []byte) *http.Request {
	req = req.WithContext(*ctx)
	if len(body) > 0 {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		req.ContentLength = int64(len(body))
	}

	return req
}

// execute the request. Note that this does not close the resp body, which should be done by caller
func execRequest(client *http.Client, req *http.Request) (start time.Time, resp *http.Response, errReason reason.Reason) {
	start = time.Now()
	resp, err := client.Do(req)

	if err != nil {
		return start, nil, reason.IOFailed(err)
	}

	return start, resp, nil
}

func splitHostnamePort(addr string) (string, uint16, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", 0, err
	}
	host := u.Host
	// Try to add a default port if needed
	if strings.LastIndex(host, ":") == -1 {
		switch u.Scheme {
		case urlSchemaHTTP:
			host += ":80"
		case urlSchemaHTTPS:
			host += ":443"
		}
	}
	host, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", 0, err
	}
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("'%v' is no valid port number in '%v'", port, u.Host)
	}
	return host, uint16(p), nil
}

// makeCheckRedirect checks if max redirects are exceeded, also append to the redirects list if we're tracking those.
// It's kind of ugly to return a result via a pointer argument, but it's the interface the
// golang HTTP client gives us.
func makeCheckRedirect(max int, redirects *[]string) func(*http.Request, []*http.Request) error {
	if max == 0 {
		return func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return func(r *http.Request, via []*http.Request) error {
		if redirects != nil {
			*redirects = append(*redirects, r.URL.String())
		}

		if max == len(via) {
			return http.ErrUseLastResponse
		}
		return nil
	}
}
