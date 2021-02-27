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

// +build windows

package npipe

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPOverNamedPipe(t *testing.T) {
	sd, err := DefaultSD("")
	require.NoError(t, err)
	npipe := TransformString("npipe:///hello-world")
	l, err := NewListener(npipe, sd)
	require.NoError(t, err)
	defer l.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/echo-hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ehlo!")
	})

	go http.Serve(l, mux)

	c := http.Client{
		Transport: &http.Transport{
			DialContext: DialContext(npipe),
		},
	}

	r, err := c.Get("http://npipe/echo-hello")
	require.NoError(t, err)
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	assert.Equal(t, "ehlo!", string(body))
}

func TestTransformString(t *testing.T) {
	t.Run("with npipe:// scheme", func(t *testing.T) {
		assert.Equal(t, `\\.\pipe\hello`, TransformString("npipe:///hello"))
	})

	t.Run("with windows pipe syntax", func(t *testing.T) {
		assert.Equal(t, `\\.\pipe\hello`, TransformString(`\\.\pipe\hello`))
	})

	t.Run("everything else", func(t *testing.T) {
		assert.Equal(t, "hello", TransformString("hello"))
	})
}
