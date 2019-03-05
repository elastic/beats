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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/elastic/beats/metricbeat/helper/server"

	"github.com/stretchr/testify/assert"
)

func GetHttpServer(host string, port int) (server.Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &HttpServer{
		done:       make(chan struct{}),
		eventQueue: make(chan server.Event, 1),
		ctx:        ctx,
		stop:       cancel,
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: http.HandlerFunc(h.handleFunc),
	}
	h.server = httpServer

	return h, nil
}

func TestHttpServer(t *testing.T) {
	host := "127.0.0.1"
	port := 40050
	svc, err := GetHttpServer(host, port)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	svc.Start()
	defer svc.Stop()
	// make sure server is up before writing data into it.
	time.Sleep(2 * time.Second)
	writeToServer(t, "test1", host, port)
	msg := <-svc.GetEvents()

	assert.True(t, msg.GetEvent() != nil)
	ok, _ := msg.GetEvent().HasKey("data")
	assert.True(t, ok)
	bytes, _ := msg.GetEvent()["data"].([]byte)
	assert.True(t, string(bytes) == "test1")

}

func writeToServer(t *testing.T, message, host string, port int) {
	url := fmt.Sprintf("http://%s:%d/", host, port)
	var str = []byte(message)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(str))
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer resp.Body.Close()

}
