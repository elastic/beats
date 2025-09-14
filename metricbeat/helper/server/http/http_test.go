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

//go:build !integration

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/helper/server"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
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
		logger:     logptest.NewTestingLogger(t, ""),
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
MIIC9TCCAd2gAwIBAgIUa4hI3ZErW13j7zCXg1Ory+FhITYwDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI0MDUxNjIwNDIwMloYDzMwMjMw
OTE3MjA0MjAyWjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDJcUM8vV6vGTycqImCwu06NSsuIHdKukHQTuvHbRGP
kXwlXNDMYEdoUX1mPArqGFunrQ9/myWoqQA7b9MTIZl4GheHvABuw0kuRos0/t4Y
zCFRRV27ATswAYp/WVBvHRZEedLJj25x8DoMeljV9dq/JKtaNNGKgztMcqWTSFPy
c+pDSSgRiP/sDebUhRaLXUhRVMsud9Wlwf6bmn62Ocj7EgrLj75u0IAb2alQ9bL9
cLAPAi0/KFx4nl8tCMQUXYM0PyNCkSM8wdwHcLiYNEKOtEx0Y4otiYLH98wlWJcl
AtMzHk5IexcTfCGzOk1fau3gNxbM9fH3+C8WBprm5lT5AgMBAAGjPTA7MBoGA1Ud
EQQTMBGHBH8AAAGCCWxvY2FsaG9zdDAdBgNVHQ4EFgQUjuHPOPincRSGgEC4DnOs
RGR8MW4wDQYJKoZIhvcNAQELBQADggEBAIFdEIGhjWrQMDx5bjif21XOaBr61uKU
3YnKMlX4bJrqjSy164SN0qBaurYUspam8YyC31IU3FSvulRoUVr3Y/VCpnfuDuEw
c5C2XJWvslRUTqZ4TAopj1vvt7wcFOJixfH3PMMdA8sKArWxlV4LtPN8h5Det0qG
F5D03fWQehviLetk7l/fdAElSoigGhJrb3HddfRcepvrWVpcUJEX3rdgwKh5RszN
1WTX/kA6w5o7JAylybV5JNKvzbpfQOH4MQD8306FB+xFPSZHgXUWJ9bJE/CbR5vd
onX6v9itbKD/hxMOZQ6HIn6F1fKK3JMJ77t35cJonwVHwV+/K2HJmNA=
-----END CERTIFICATE-----`)

	keyPem := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDJcUM8vV6vGTyc
qImCwu06NSsuIHdKukHQTuvHbRGPkXwlXNDMYEdoUX1mPArqGFunrQ9/myWoqQA7
b9MTIZl4GheHvABuw0kuRos0/t4YzCFRRV27ATswAYp/WVBvHRZEedLJj25x8DoM
eljV9dq/JKtaNNGKgztMcqWTSFPyc+pDSSgRiP/sDebUhRaLXUhRVMsud9Wlwf6b
mn62Ocj7EgrLj75u0IAb2alQ9bL9cLAPAi0/KFx4nl8tCMQUXYM0PyNCkSM8wdwH
cLiYNEKOtEx0Y4otiYLH98wlWJclAtMzHk5IexcTfCGzOk1fau3gNxbM9fH3+C8W
Bprm5lT5AgMBAAECggEAEYpJsv/AP1ngs7lfI+IqOt/HT0BncrvOID/G+vntxgUC
fNRcn/cgMJ6r3xuKTcDqNir1BwTw3gM9MG+3vto1nUYUV27Q0NQzSpK861Pn7dvU
aNmz5CUizLbNovIZdVtghXzgFEnncYdb3ptGofbC4dLlErk3p6punuT6stzg5mL2
y/2yHBrfQEnuDRI8pQ5Vcuo24GioZqWiS35qVGLbonvor0DKv4lkNjMix6ulwwb+
3rvEAhTOhgYKe7h6RjKnc4SbIsnSpGzhC9M7hLF+F57GIw61uaJnISfkuw/FGhaR
XkeyV8TB8MDTgP30+7xam6pvB2rKcRsrVgPmLC7WgQKBgQDRHgRHDTgpBSx9F+N6
6KU01g5cemxKVBHMm5L2n99YpR9BoiWViKkFWAWALmRlq/nFk22hq4t2+niH/6a+
0ioAhIOnZZTXK/n5DsBCdqg1d1ZO4ih4Iw1/TR1iIR0M8ptkIBGVWKslV8OKQNd4
zNUCmDzb8pmuzVKjwVs7ca9HmQKBgQD2msK7eh81A2dxXPl1chcudFB33zMwA1Y0
3ZEPsGAinvU5ILwwMlg1w7N1NKwcDYiBkJG1SCoujoTsYoXMKjnlgf5uoklfJJBI
U3QKYMGDRdlqE02V31KBVcv/EdNR8olfjy1xbgCKu04rYnCPGLSLNc6MgcSMYnLr
y9rZlq5UYQKBgQCi0K4f6+j39zFGTF0vCwfl9WvFEQRTctVQ6ygnoR4yVI3bejWt
EXQX1wqhXH2Ks7WK4ViQcZHqluVVbfUTyWoucP5YTTzvsyuzgIqstNoOltW6IVfF
AfW2UgI4rvOBazsVX+qQzzKhpo12jTm2sjR/Cq0HywFhGjfni9pOlBsWsQKBgQDz
3IbFLja+Dee1SuPFKFWUMqGAaNANor8U+CYDBb+LfPWy0JRIdQCV6jkEplmsRBXB
Sl1Mj1hnQbhgqez1wKwQMUSR0xoLY/TqENynhpbWYbRmGUCX/IdyLo3UZqQ6XUVL
oiKmEMmoZyEd9fKpDx06rLLcb1cWHCTY2HZKxZ8PAQKBgF3ftzNurXMCBH9W2RkI
hHhpHArwSLCsDVeGpS6vYDz+EX+RP1t1jJZbTRyOkk/X5RNVA3Yup6Lw8ANWqpPJ
MMbn7YyWGaClkcuHqavOU7kfaqF5S6vECOAtSWd+NPOHUALTDnmBUnLTE4KmzarO
8hd7Y6EEu0Lwkc3GnoQUwzRh
-----END PRIVATE KEY-----`)

	cfg := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
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

	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIC9TCCAd2gAwIBAgIUa4hI3ZErW13j7zCXg1Ory+FhITYwDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI0MDUxNjIwNDIwMloYDzMwMjMw
OTE3MjA0MjAyWjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDJcUM8vV6vGTycqImCwu06NSsuIHdKukHQTuvHbRGP
kXwlXNDMYEdoUX1mPArqGFunrQ9/myWoqQA7b9MTIZl4GheHvABuw0kuRos0/t4Y
zCFRRV27ATswAYp/WVBvHRZEedLJj25x8DoMeljV9dq/JKtaNNGKgztMcqWTSFPy
c+pDSSgRiP/sDebUhRaLXUhRVMsud9Wlwf6bmn62Ocj7EgrLj75u0IAb2alQ9bL9
cLAPAi0/KFx4nl8tCMQUXYM0PyNCkSM8wdwHcLiYNEKOtEx0Y4otiYLH98wlWJcl
AtMzHk5IexcTfCGzOk1fau3gNxbM9fH3+C8WBprm5lT5AgMBAAGjPTA7MBoGA1Ud
EQQTMBGHBH8AAAGCCWxvY2FsaG9zdDAdBgNVHQ4EFgQUjuHPOPincRSGgEC4DnOs
RGR8MW4wDQYJKoZIhvcNAQELBQADggEBAIFdEIGhjWrQMDx5bjif21XOaBr61uKU
3YnKMlX4bJrqjSy164SN0qBaurYUspam8YyC31IU3FSvulRoUVr3Y/VCpnfuDuEw
c5C2XJWvslRUTqZ4TAopj1vvt7wcFOJixfH3PMMdA8sKArWxlV4LtPN8h5Det0qG
F5D03fWQehviLetk7l/fdAElSoigGhJrb3HddfRcepvrWVpcUJEX3rdgwKh5RszN
1WTX/kA6w5o7JAylybV5JNKvzbpfQOH4MQD8306FB+xFPSZHgXUWJ9bJE/CbR5vd
onX6v9itbKD/hxMOZQ6HIn6F1fKK3JMJ77t35cJonwVHwV+/K2HJmNA=
-----END CERTIFICATE-----`)

	keyPem := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDJcUM8vV6vGTyc
qImCwu06NSsuIHdKukHQTuvHbRGPkXwlXNDMYEdoUX1mPArqGFunrQ9/myWoqQA7
b9MTIZl4GheHvABuw0kuRos0/t4YzCFRRV27ATswAYp/WVBvHRZEedLJj25x8DoM
eljV9dq/JKtaNNGKgztMcqWTSFPyc+pDSSgRiP/sDebUhRaLXUhRVMsud9Wlwf6b
mn62Ocj7EgrLj75u0IAb2alQ9bL9cLAPAi0/KFx4nl8tCMQUXYM0PyNCkSM8wdwH
cLiYNEKOtEx0Y4otiYLH98wlWJclAtMzHk5IexcTfCGzOk1fau3gNxbM9fH3+C8W
Bprm5lT5AgMBAAECggEAEYpJsv/AP1ngs7lfI+IqOt/HT0BncrvOID/G+vntxgUC
fNRcn/cgMJ6r3xuKTcDqNir1BwTw3gM9MG+3vto1nUYUV27Q0NQzSpK861Pn7dvU
aNmz5CUizLbNovIZdVtghXzgFEnncYdb3ptGofbC4dLlErk3p6punuT6stzg5mL2
y/2yHBrfQEnuDRI8pQ5Vcuo24GioZqWiS35qVGLbonvor0DKv4lkNjMix6ulwwb+
3rvEAhTOhgYKe7h6RjKnc4SbIsnSpGzhC9M7hLF+F57GIw61uaJnISfkuw/FGhaR
XkeyV8TB8MDTgP30+7xam6pvB2rKcRsrVgPmLC7WgQKBgQDRHgRHDTgpBSx9F+N6
6KU01g5cemxKVBHMm5L2n99YpR9BoiWViKkFWAWALmRlq/nFk22hq4t2+niH/6a+
0ioAhIOnZZTXK/n5DsBCdqg1d1ZO4ih4Iw1/TR1iIR0M8ptkIBGVWKslV8OKQNd4
zNUCmDzb8pmuzVKjwVs7ca9HmQKBgQD2msK7eh81A2dxXPl1chcudFB33zMwA1Y0
3ZEPsGAinvU5ILwwMlg1w7N1NKwcDYiBkJG1SCoujoTsYoXMKjnlgf5uoklfJJBI
U3QKYMGDRdlqE02V31KBVcv/EdNR8olfjy1xbgCKu04rYnCPGLSLNc6MgcSMYnLr
y9rZlq5UYQKBgQCi0K4f6+j39zFGTF0vCwfl9WvFEQRTctVQ6ygnoR4yVI3bejWt
EXQX1wqhXH2Ks7WK4ViQcZHqluVVbfUTyWoucP5YTTzvsyuzgIqstNoOltW6IVfF
AfW2UgI4rvOBazsVX+qQzzKhpo12jTm2sjR/Cq0HywFhGjfni9pOlBsWsQKBgQDz
3IbFLja+Dee1SuPFKFWUMqGAaNANor8U+CYDBb+LfPWy0JRIdQCV6jkEplmsRBXB
Sl1Mj1hnQbhgqez1wKwQMUSR0xoLY/TqENynhpbWYbRmGUCX/IdyLo3UZqQ6XUVL
oiKmEMmoZyEd9fKpDx06rLLcb1cWHCTY2HZKxZ8PAQKBgF3ftzNurXMCBH9W2RkI
hHhpHArwSLCsDVeGpS6vYDz+EX+RP1t1jJZbTRyOkk/X5RNVA3Yup6Lw8ANWqpPJ
MMbn7YyWGaClkcuHqavOU7kfaqF5S6vECOAtSWd+NPOHUALTDnmBUnLTE4KmzarO
8hd7Y6EEu0Lwkc3GnoQUwzRh
-----END PRIVATE KEY-----`)

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(certPem); !ok {
		t.Fatal("failed to append server certificate to the pool")
	}

	cfg := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
		RootCAs:    certPool,
	}
	cfg.Certificates = make([]tls.Certificate, 1)
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		t.Error(err)
	}
	cfg.Certificates = []tls.Certificate{cert}

	if connectionType == "HTTPS" {
		client.Transport = &http.Transport{
			TLSClientConfig: cfg,
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	defer resp.Body.Close()

	if connectionMethod == "GET" {
		if resp.StatusCode == http.StatusOK {
			bodyBytes, err2 := io.ReadAll(resp.Body)
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
