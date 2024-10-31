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

package es

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestStore(t *testing.T) {
	// This just a convenience test for store development
	// REMOVE: before the opening PR
	t.Skip()

	ctx, cn := context.WithCancel(context.Background())
	defer cn()

	notifier := NewNotifier()

	store, err := openStore(ctx, logp.NewLogger("tester"), "filebeat", notifier)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"api_key": "xxxxxxxxxx:xxxxxxxc-U6VH4DK8g",
		"hosts": []string{
			"https://6598f1d41f9d4e81a78117dddbb2b03e.us-central1.gcp.cloud.es.io:443",
		},
		"preset": "balanced",
		"type":   "elasticsearch",
	})

	if err != nil {
		t.Fatal(err)
	}

	notifier.NotifyConfigUpdate(config)

	var m map[string]any
	store.SetID("httpjson-okta.system-028ecf4b-babe-44c6-939e-9e3096af6959")
	err = store.Get("foo", &m)
	if err != nil && !errors.Is(err, ErrKeyUnknown) {
		t.Fatal(err)
	}

	err = store.Each(func(s string, vd backend.ValueDecoder) (bool, error) {
		var v any
		err := vd.Decode(&v)
		if err != nil {
			return false, err
		}
		return true, nil
	})

	v := map[string]interface{}{
		"updated": []interface{}{
			float64(280444598839616),
			float64(1729277837),
		},
		"cursor": map[string]interface{}{
			"published": "2024-10-17T18:33:58.960Z",
		},
		"ttl": float64(1800000000000),
	}

	err = store.Set("foo", v)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Get("foo", &m)
	if err != nil && !errors.Is(err, ErrKeyUnknown) {
		t.Fatal(err)
	}

	diff := cmp.Diff(v, m)
	if diff != "" {
		t.Fatal(diff)
	}

	var s1 = "dfsdf"
	err = store.Set("foo1", s1)
	if err != nil {
		t.Fatal(err)
	}

	var s2 string
	err = store.Get("foo1", &s2)
	if err != nil {
		t.Fatal(err)
	}

	diff = cmp.Diff(s1, s2)
	if diff != "" {
		t.Fatal(diff)
	}

	var n1 = 12345
	err = store.Set("foon", n1)
	if err != nil {
		t.Fatal(err)
	}

	var n2 int
	err = store.Get("foon", &n2)
	if err != nil {
		t.Fatal(err)
	}

	diff = cmp.Diff(n1, n2)
	if diff != "" {
		t.Fatal(diff)
	}

	if err != nil {
		t.Fatal(err)
	}

	err = store.Remove("foon")
	if err != nil {
		t.Fatal(err)
	}

	err = store.Each(func(s string, vd backend.ValueDecoder) (bool, error) {
		var v any
		err := vd.Decode(&v)
		if err != nil {
			return false, err
		}
		return true, nil
	})

	if err != nil {
		t.Fatal(err)
	}

}
