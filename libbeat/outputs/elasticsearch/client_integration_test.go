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

// +build integration

package elasticsearch

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go.elastic.co/apm/apmtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegtest"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
)

func TestClientPublishEvent(t *testing.T) {
	index := "beat-int-pub-single-event"
	cfg := map[string]interface{}{
		"index": index,
	}

	testPublishEvent(t, index, cfg)
}

func TestClientPublishEventKerberosAware(t *testing.T) {
	err := setupRoleMapping(t, eslegtest.GetEsKerberosHost())
	if err != nil {
		t.Fatal(err)
	}

	index := "beat-int-pub-single-event-behind-kerb"
	cfg := map[string]interface{}{
		"hosts":    eslegtest.GetEsKerberosHost(),
		"index":    index,
		"username": "",
		"password": "",
		"kerberos": map[string]interface{}{
			"auth_type":   "password",
			"config_path": "testdata/krb5.conf",
			"username":    eslegtest.GetUser(),
			"password":    eslegtest.GetPass(),
			"realm":       "ELASTIC",
		},
	}

	testPublishEvent(t, index, cfg)
}

func testPublishEvent(t *testing.T, index string, cfg map[string]interface{}) {
	output, client := connectTestEs(t, cfg)

	// drop old index preparing test
	client.conn.Delete(index, "", "", nil)

	batch := outest.NewBatch(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":    "libbeat",
			"message": "Test message from libbeat",
		},
	})

	err := output.Publish(context.Background(), batch)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = client.conn.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	_, resp, err := client.conn.CountSearchURI(index, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, resp.Count)
}

func TestClientPublishEventWithPipeline(t *testing.T) {
	type obj map[string]interface{}

	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := "beat-int-pub-single-with-pipeline"
	pipeline := "beat-int-pub-single-pipeline"

	output, client := connectTestEs(t, obj{
		"index":    index,
		"pipeline": "%{[pipeline]}",
	})
	client.conn.Delete(index, "", "", nil)

	// Check version
	if client.conn.GetVersion().Major < 5 {
		t.Skip("Skipping tests as pipeline not available in <5.x releases")
	}

	publish := func(event beat.Event) {
		err := output.Publish(context.Background(), outest.NewBatch(event))
		if err != nil {
			t.Fatal(err)
		}
	}

	getCount := func(query string) int {
		_, resp, err := client.conn.CountSearchURI(index, "", map[string]string{
			"q": query,
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Count
	}

	pipelineBody := obj{
		"description": "Test pipeline",
		"processors": []obj{
			{
				"set": obj{
					"field": "testfield",
					"value": 1,
				},
			},
		},
	}

	client.conn.DeletePipeline(pipeline, nil)
	_, resp, err := client.conn.CreatePipeline(pipeline, nil, pipelineBody)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Acknowledged {
		t.Fatalf("Test pipeline %v not created", pipeline)
	}

	publish(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":      "libbeat",
			"message":   "Test message 1",
			"pipeline":  pipeline,
			"testfield": 0,
		}})
	publish(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":      "libbeat",
			"message":   "Test message 2",
			"testfield": 0,
		}})

	_, _, err = client.conn.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, getCount("testfield:1")) // with pipeline 1
	assert.Equal(t, 1, getCount("testfield:0")) // no pipeline
}

func TestClientBulkPublishEventsWithPipeline(t *testing.T) {
	type obj map[string]interface{}

	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := "beat-int-pub-bulk-with-pipeline"
	pipeline := "beat-int-pub-bulk-pipeline"

	output, client := connectTestEs(t, obj{
		"index":    index,
		"pipeline": "%{[pipeline]}",
	})
	client.conn.Delete(index, "", "", nil)

	if client.conn.GetVersion().Major < 5 {
		t.Skip("Skipping tests as pipeline not available in <5.x releases")
	}

	publish := func(events ...beat.Event) {
		err := output.Publish(context.Background(), outest.NewBatch(events...))
		if err != nil {
			t.Fatal(err)
		}
	}

	getCount := func(query string) int {
		_, resp, err := client.conn.CountSearchURI(index, "", map[string]string{
			"q": query,
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Count
	}

	pipelineBody := obj{
		"description": "Test pipeline",
		"processors": []obj{
			{
				"set": obj{
					"field": "testfield",
					"value": 1,
				},
			},
		},
	}

	client.conn.DeletePipeline(pipeline, nil)
	_, resp, err := client.conn.CreatePipeline(pipeline, nil, pipelineBody)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Acknowledged {
		t.Fatalf("Test pipeline %v not created", pipeline)
	}

	publish(
		beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":      "libbeat",
				"message":   "Test message 1",
				"pipeline":  pipeline,
				"testfield": 0,
			}},
		beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":      "libbeat",
				"message":   "Test message 2",
				"testfield": 0,
			}},
	)

	_, _, err = client.conn.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, getCount("testfield:1")) // with pipeline 1
	assert.Equal(t, 1, getCount("testfield:0")) // no pipeline
}

func TestClientPublishTracer(t *testing.T) {
	index := "beat-apm-tracer-test"
	output, client := connectTestEs(t, map[string]interface{}{
		"index": index,
	})

	client.conn.Delete(index, "", "", nil)

	batch := outest.NewBatch(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": "Hello world",
		},
	})

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		err := output.Publish(ctx, batch)
		if err != nil {
			t.Fatal(err)
		}
	})
	require.Len(t, spans, 2)

	// get spans in reverse order
	firstSpan := spans[1]

	assert.Equal(t, "publishEvents", firstSpan.Name)
	assert.Equal(t, "output", firstSpan.Type)
	assert.Equal(t, [8]byte(firstSpan.TransactionID), [8]byte(tx.ID))
	assert.True(t, len(firstSpan.Context.Tags) > 0, "no tags found")

	secondSpan := spans[0]
	assert.Contains(t, secondSpan.Name, "POST")
	assert.Equal(t, "db", secondSpan.Type)
	assert.Equal(t, "elasticsearch", secondSpan.Subtype)
	assert.Equal(t, [8]byte(secondSpan.ParentID), [8]byte(firstSpan.ID))
	assert.Equal(t, [8]byte(secondSpan.TransactionID), [8]byte(tx.ID))
	assert.Equal(t, "/_bulk", secondSpan.Context.HTTP.URL.Path)
}

func connectTestEs(t *testing.T, cfg interface{}) (outputs.Client, *Client) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"hosts":            eslegtest.GetEsHost(),
		"username":         eslegtest.GetUser(),
		"password":         eslegtest.GetPass(),
		"template.enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	tmp, err := common.NewConfigFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = config.Merge(tmp)
	if err != nil {
		t.Fatal(err)
	}

	info := beat.Info{Beat: "libbeat"}
	im, _ := idxmgmt.DefaultSupport(nil, info, nil)
	output, err := makeES(im, info, outputs.NewNilObserver(), config)
	if err != nil {
		t.Fatal(err)
	}

	type clientWrap interface {
		outputs.NetworkClient
		Client() outputs.NetworkClient
	}
	client := randomClient(output).(clientWrap).Client().(*Client)

	// Load version number
	client.Connect()

	return client, client
}

// setupRoleMapping sets up role mapping for the Kerberos user beats@ELASTIC
func setupRoleMapping(t *testing.T, host string) error {
	_, client := connectTestEs(t, map[string]interface{}{
		"hosts":    host,
		"username": "elastic",
		"password": "changeme",
	})

	roleMappingURL := client.conn.URL + "/_security/role_mapping/kerbrolemapping"

	status, _, err := client.conn.RequestURL("POST", roleMappingURL, map[string]interface{}{
		"roles":   []string{"superuser"},
		"enabled": true,
		"rules": map[string]interface{}{
			"field": map[string]interface{}{
				"username": "beats@ELASTIC",
			},
		},
	})

	if status >= 300 {
		return fmt.Errorf("non-2xx return code: %d", status)
	}

	return err
}

func randomClient(grp outputs.Group) outputs.NetworkClient {
	L := len(grp.Clients)
	if L == 0 {
		panic("no elasticsearch client")
	}

	client := grp.Clients[rand.Intn(L)]
	return client.(outputs.NetworkClient)
}

// startTestProxy starts a proxy that redirects all connections to the specified URL
func startTestProxy(t *testing.T, redirectURL string) *httptest.Server {
	t.Helper()

	realURL, err := url.Parse(redirectURL)
	require.NoError(t, err)

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := r.Clone(context.Background())
		req.RequestURI = ""
		req.URL.Scheme = realURL.Scheme
		req.URL.Host = realURL.Host

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		for _, header := range []string{"Content-Encoding", "Content-Type"} {
			w.Header().Set(header, resp.Header.Get(header))
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	}))
	return proxy
}
