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

// +build !integration

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/helper/server"
)

func TestHTTPServers(t *testing.T) {

	var cases = []struct {
		testName, inputMessage, connectionType, connectionMethod, expectedOutput string
		expectedHTTPCode                                                         int
	}{
		{"HTTP GET", `"@timestamp":"2016-05-23T08:05:34.853Z"`, "HTTP", "GET", "HTTP Server accepts data via POST", 200},
		{"HTTPS GET", `"@timestamp":"2016-05-23T08:05:34.853Z"`, "HTTPS", "GET", "HTTPS Server accepts data via POST", 200},
		{"HTTP POST", `"@timestamp":"2016-05-23T08:05:34.853Z"`, "HTTP", "POST", `"@timestamp":"2016-05-23T08:05:34.853Z"`, 202},
		{"HTTPS POST", `"@timestamp":"2016-05-23T08:05:34.853Z"`, "HTTPS", "POST", `"@timestamp":"2016-05-23T08:05:34.853Z"`, 202},
	}

	for _, test := range cases {
		t.Run(test.testName, func(t *testing.T) {
			host := "127.0.0.1"
			port := 40050
			svc, err := getHTTPServer(t, host, port, test.connectionType)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			svc.Start()
			defer svc.Stop()
			// make sure server is up before writing data into it.
			err = checkServerReady(host, port)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			httpCode, response := writeToServer(t, test.inputMessage, host, port, test.connectionMethod, test.connectionType)

			assert.True(t, httpCode == test.expectedHTTPCode)

			if test.connectionMethod == "POST" {
				msg := <-svc.GetEvents()

				assert.True(t, msg.GetEvent() != nil)
				ok, _ := msg.GetEvent().HasKey("data")
				assert.True(t, ok)
				bytes, _ := msg.GetEvent()["data"].([]byte)
				httpOutput := string(bytes)
				assert.True(t, httpOutput == test.expectedOutput)
			} else {
				assert.True(t, response == test.expectedOutput)
			}

		})
	}
}

func checkServerReady(host string, port int) error {

	const (
		checkServerReadyTimeout = 5 * time.Second
		checkServerReadyTick    = 100 * time.Millisecond
	)
	var conn net.Conn
	var err error

	ctx, cancel := context.WithTimeout(context.TODO(), checkServerReadyTimeout)
	defer cancel()
	ticker := time.NewTicker(checkServerReadyTick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn, err = net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
			if conn != nil {
				_ = conn.Close()
				return nil
			}
			if err != nil {
				return err
			}

		case <-ctx.Done():
			return fmt.Errorf("HTTP server at %s:%d never responded: %+v", host, port, err)
		}
	}

}

func getHTTPServer(t *testing.T, host string, port int, connectionType string) (server.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &HttpServer{
		done:       make(chan struct{}),
		eventQueue: make(chan server.Event, 1),
		ctx:        ctx,
		stop:       cancel,
	}
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(host, strconv.Itoa(int(port))),
		Handler: http.HandlerFunc(h.handleFunc),
	}
	if connectionType == "HTTPS" {
		cfg := prepareTLSConfig(t, host)
		httpServer.TLSConfig = cfg
	}
	h.server = httpServer
	return h, nil
}

func prepareTLSConfig(t *testing.T, host string) *tls.Config {
	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIDwTCCAqmgAwIBAgIJAONBEV813hm6MA0GCSqGSIb3DQEBCwUAMHcxCzAJBgNV
BAYTAkJSMQswCQYDVQQIDAJTUDEPMA0GA1UEBwwGRlJBTkNBMRAwDgYDVQQKDAdF
TEFTVElDMQswCQYDVQQLDAJPVTERMA8GA1UEAwwIaG9tZS5jb20xGDAWBgkqhkiG
9w0BCQEWCWV1QGV1LmNvbTAeFw0xOTAzMjYxOTMxMjhaFw0yOTAzMjMxOTMxMjha
MHcxCzAJBgNVBAYTAkJSMQswCQYDVQQIDAJTUDEPMA0GA1UEBwwGRlJBTkNBMRAw
DgYDVQQKDAdFTEFTVElDMQswCQYDVQQLDAJPVTERMA8GA1UEAwwIaG9tZS5jb20x
GDAWBgkqhkiG9w0BCQEWCWV1QGV1LmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBALOJ2dxpBsQtRvs2hSuUhDsf4w6G3swFqtIXLedPvz1rNuofm75G
dA9pqXiI3hDw2ZuIJZItXE3FfVXxoE/ugsFw6cVLKrnpQ8exIv8K0JNuR22faFcR
LmDx/YLw0wmOnM2maBSaetrM5F4CwoVqDmOwZHs9fbADqthAHrbCAzNTkqnx2B4/
RWaYPbRWlSQ7CrWQE9cNJ/WMdUjznd5H0IiV7k/cHKIbXi3+JNinCWHAACWWS3ig
DjjCZd9lHkDH6qSpNGsQU5y0eiFAiiBVPqDIdVfPRe4pC81z3Dp6Wqs0uHXHYHqB
o3YWkXngTLlMLZtIMF+pWlCJZkscgLjL/N8CAwEAAaNQME4wHQYDVR0OBBYEFBpI
Tu/9mmRqithdHZZMu5jRLHebMB8GA1UdIwQYMBaAFBpITu/9mmRqithdHZZMu5jR
LHebMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAGTS+cvN/vGjbkDF
wZRG8xMeHPHzlCWKNEGwZXTTBADrjfnppW5I2f5cDZzg71+UzQSJmBmHKZd+adrW
2GA888CAT+birIE6EAwIyq7ZGe77ymRspugyb7AK46QOKApED3izxId36Tk5/a0P
QY3WOTC0Y4yvz++gbx/uviYDMoHuJl0nIEXqtT9OZ2V2GqCToJu300RV/MIRtk6s
0U1d9CRDkjNolGVbYo2VnDJbZ8LQtJHS5iDeiEztay5Cky4NvVZsbCxrgNrr3h/v
upHEJ28Q7QzMnRC7d/THI6fRW1mG6BuFT3WPW5K7EAfgQDlyyspTDrACrYTuWC+y
013uTlI=
-----END CERTIFICATE-----`)

	keyPem := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAs4nZ3GkGxC1G+zaFK5SEOx/jDobezAWq0hct50+/PWs26h+b
vkZ0D2mpeIjeEPDZm4glki1cTcV9VfGgT+6CwXDpxUsquelDx7Ei/wrQk25HbZ9o
VxEuYPH9gvDTCY6czaZoFJp62szkXgLChWoOY7Bkez19sAOq2EAetsIDM1OSqfHY
Hj9FZpg9tFaVJDsKtZAT1w0n9Yx1SPOd3kfQiJXuT9wcohteLf4k2KcJYcAAJZZL
eKAOOMJl32UeQMfqpKk0axBTnLR6IUCKIFU+oMh1V89F7ikLzXPcOnpaqzS4dcdg
eoGjdhaReeBMuUwtm0gwX6laUIlmSxyAuMv83wIDAQABAoIBAD1kY/T0jPXELcN1
LzBpxpWZH8E16TWGspTIjE/Oeyx7XvnL+SulV8Z1cRfgZV8RnLeMZJyJmkiVwXgD
+bebbWbMP4PRYjjURPMh5T+k6RGg4hfgLIOpQlywIuoFg4R/GatQvcJd2Ki861Ii
S3XngCgihxmFO1dWybLMqjQAP6vq01sbctUXYddFd5STInzrceoXwkLjp3gTR1et
FG+Anmzbxp8e2ETXvwuf7eZhVwCJ2DxBt7tx1j5Csuj1LjaVTe5qR7B1oM7/vo0b
LlY9IixAAi62Rrv4YSvMAtMI6mQt+AM/4uBVqoG/ipgkuoQVuQ+M4lGdmEXwEEkz
Ol7SlMECgYEA11tV+ZekVsujBmasTU7TfWtcYtRHh+FSC040bVLiE6XZbuVJ4sSA
TvuUDs+3XM8blnkfVo826WY4+bKkj1PdCFsmG5pm+wnSTPFKWsCtsSyA3ts85t3O
IvcCxXA/1xL9O/UdWfrl2+IJ3yLDEjEU5QTYP34+KDBZM3u6tJzjWe8CgYEA1WwA
8d75h9UQyFXWEOiwJmR6yX7PGkpYE3J7m2p2giEbLm+9no5CEmE9T74k3m0eLZug
g/F1MA/evhXEYho6f+lS9Q0ZdtyU2EFrdvuLlUw6FJIWnaOLlVR/aC6BvAlxLDRb
RUGqDKDjl1Die0s8F1aDHGvNvGaZRN4Z23BRPBECgYBE8pMGA8yzlSKui/SiE5iW
UOcVJQ15rWPNBs62KZED5VdFr9cF6Q+DOfxe+ZWk+xHEDSdBWTylYPrgxpb05E6h
vDzpHXfW64AO7jl18LYrQSpJLzvCVkUG4LpcZ+GohAXbSlCJXFB3I1kxvTli+5/K
6tApE8vmpgQI/ZX6+Te4tQKBgBcQ3C1H5voaOf0c4czkCR2tIGQkk2eI/2nipp9O
a053G4PySbEYOOXZopG6wCtV6bwOJNP9xaeTH4S1v4rGwOnQIsofR1BEWMXilCXA
2/4fxesxOsaAxXY3Mqnk1NqovpWDdxXOGf3RaaeR81hV8kGndPYeZJbnE8uQoYTI
586xAoGBAI2SR17xbgfiQBZxgGqamslz4NqBkZUBs4DIAGMAXS21rW/2bbbRaSii
mGmkdaXx+l077AuO0peX2uBvJAx6PvAVW0qroeOLcCo6EuUGTNVhBej6L9hMwhIO
r0tZLlMt75zcnJBicMbIrrzIGVYMHjT+m1QTGbrGb/tcEIGtmXwO
-----END RSA PRIVATE KEY-----`)

	cfg := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}
	cfg.Certificates = make([]tls.Certificate, 1)
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		t.Error(err)
	}
	cfg.Certificates = []tls.Certificate{cert}
	return cfg
}

func writeToServer(t *testing.T, message, host string, port int, connectionMethod string, connectionType string) (int, string) {
	url := fmt.Sprintf("%s://%s:%d/", strings.ToLower(connectionType), host, port)
	var str = []byte(message)
	req, err := http.NewRequest(connectionMethod, url, bytes.NewBuffer(str))
	req.Header.Set("Content-Type", "text/plain")
	client := &http.Client{}
	if connectionType == "HTTPS" {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // test server certificate is not trusted.
			}}
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	defer resp.Body.Close()

	if connectionMethod == "GET" {
		if resp.StatusCode == http.StatusOK {
			bodyBytes, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				t.Error(err)
				t.FailNow()
			}
			bodyString := string(bodyBytes)
			return resp.StatusCode, bodyString
		}
	}
	return resp.StatusCode, ""
}
