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

package api

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/api/npipe"
	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestNamedPipe(t *testing.T) {
	p := "npipe:///hello"

	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"host": p,
	})

	s, err := New(nil, simpleMux(), cfg)
	require.NoError(t, err)
	go s.Start()
	defer s.Stop()

	c := http.Client{
		Transport: &http.Transport{
			DialContext: npipe.DialContext(npipe.TransformString(p)),
		},
	}

	r, err := c.Get("http://npipe/echo-hello")
	require.NoError(t, err)
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)

	assert.Equal(t, "ehlo!", string(body))
}
