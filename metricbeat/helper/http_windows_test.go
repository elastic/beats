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

//go:build windows
// +build windows

package helper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/api/npipe"
	"github.com/elastic/beats/v8/metricbeat/helper/dialer"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

func TestOverNamedpipe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("npipe is only supported under Windows")
		return
	}

	t.Run("at root", func(t *testing.T) {
		p := `\\.\pipe\hellofromnpipe`
		sd, err := npipe.DefaultSD("")
		require.NoError(t, err)
		l, err := npipe.NewListener(p, sd)
		require.NoError(t, err)
		defer l.Close()

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ehlo!")
		})

		go http.Serve(l, mux)

		cfg := defaultConfig()
		hostData := mb.HostData{
			Transport:    dialer.NewNpipeDialerBuilder(p),
			URI:          "http://npipe/",
			SanitizedURI: "http://npipe/",
		}

		h, err := NewHTTPFromConfig(cfg, hostData)
		require.NoError(t, err)

		r, err := h.FetchResponse()
		require.NoError(t, err)
		defer r.Body.Close()
		content, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte("ehlo!"), content)
	})

	t.Run("at specific path", func(t *testing.T) {
		p := `\\.\pipe\apath`
		sd, err := npipe.DefaultSD("")
		require.NoError(t, err)
		l, err := npipe.NewListener(p, sd)
		require.NoError(t, err)
		defer l.Close()

		mux := http.NewServeMux()
		mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "ehlo!")
		})

		go http.Serve(l, mux)

		cfg := defaultConfig()
		hostData := mb.HostData{
			Transport:    dialer.NewNpipeDialerBuilder(p),
			URI:          "http://npipe/ok",
			SanitizedURI: "http://npipe/ok",
		}

		h, err := NewHTTPFromConfig(cfg, hostData)
		require.NoError(t, err)

		r, err := h.FetchResponse()
		require.NoError(t, err)
		defer r.Body.Close()
		content, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte("ehlo!"), content)
	})

}
