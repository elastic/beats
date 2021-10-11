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
// +build !integration

package tls

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func getParser() *parser {
	logp.TestingSetup(logp.WithSelectors("tls", "tlsdetailed"))
	return &parser{}
}

func mkBuf(t *testing.T, s string, length int) *bufferView {
	bytes, err := hex.DecodeString(s)
	assert.NoError(t, err)
	return newBufferView(streambuf.New(bytes), 0, length)
}

func TestParse(t *testing.T) {
	parser := getParser()
	err := parser.parseAlert(mkBuf(t, "0102", 2))
	assert.NoError(t, err)
	assert.Len(t, parser.alerts, 1)
	assert.Equal(t, alertSeverity(1), parser.alerts[0].severity)
	assert.Equal(t, alertCode(2), parser.alerts[0].code)
}

func TestShortBuffer(t *testing.T) {
	parser := getParser()
	err := parser.parseAlert(mkBuf(t, "", 2))
	assert.Error(t, err)
	assert.Empty(t, parser.alerts)

	err = parser.parseAlert(mkBuf(t, "01", 2))
	assert.Error(t, err)
	assert.Empty(t, parser.alerts)
}

func TestEncrypted(t *testing.T) {
	parser := getParser()
	err := parser.parseAlert(mkBuf(t, "010200000000", 6))
	assert.NoError(t, err)
	assert.Empty(t, parser.alerts)
}
