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

//This file was contributed to by generative AI

package udp

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/testpipeline"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestInput(t *testing.T) {
	serverAddr := "127.0.0.1:9042"
	wg := sync.WaitGroup{}
	inp, err := configure(conf.MustNewConfigFrom(map[string]any{
		"host": serverAddr,
	}))
	if err != nil {
		t.Fatalf("cannot create input: %s", err)
	}

	pipeline := testpipeline.NewPipelineConnector()
	client, _ := pipeline.Connect()

	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		ID:          t.Name(),
		Cancelation: ctx,
		Logger:      logp.NewNopLogger(),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := inp.Run(v2Ctx, client); err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Errorf("input exited with error: %s", err)
			}
		}
	}()

	// Give the input to start running. Because it's UDP we cannot know from
	// the client side if the server is up and running.
	time.Sleep(time.Second)

	runUDPClient(t, serverAddr, []string{"foo", "bar"})
	inputtest.RequireNetMetricsCount(
		t,
		3*time.Second,
		2,
		2,
	)

	// Stop the input, this removes all metrics
	cancel()

	// Ensure the input Run method returns
	wg.Wait()
}

func runUDPClient(t *testing.T, address string, dataToSend []string) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		t.Fatalf("cannot create connection: %s", err)
	}
	defer conn.Close()

	// Send data to the server
	for _, data := range dataToSend {
		_, err = conn.Write([]byte(data + "\n"))
		if err != nil {
			t.Logf("Error sending data: %s, retrying in 100ms", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
