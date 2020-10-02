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

package sip

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestParseURI(t *testing.T) {
	scheme, username, host, port := parseURI(common.NetString("sip:test@10.0.2.15:5060"))
	assert.Equal(t, common.NetString("sip"), scheme)
	assert.Equal(t, common.NetString("test"), username)
	assert.Equal(t, common.NetString("10.0.2.15"), host)
	assert.Equal(t, 5060, port)

	scheme, username, host, port = parseURI(common.NetString("sips:test@10.0.2.15:5061 ; ignored"))
	assert.Equal(t, common.NetString("sips"), scheme)
	assert.Equal(t, common.NetString("test"), username)
	assert.Equal(t, common.NetString("10.0.2.15"), host)
	assert.Equal(t, 5061, port)
}

func TestParseFromTo(t *testing.T) {
	// To
	displayInfo, uri, tag := parseFromTo(common.NetString("test <sip:test@10.0.2.15:5060>;tag=QvN921"))
	assert.Equal(t, common.NetString("test"), displayInfo)
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString("QvN921"), tag)
	displayInfo, uri, tag = parseFromTo(common.NetString("test <sip:test@10.0.2.15:5060>"))
	assert.Equal(t, common.NetString("test"), displayInfo)
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString(nil), tag)

	// From
	displayInfo, uri, tag = parseFromTo(common.NetString("\"PCMU/8000\" <sip:sipp@10.0.2.15:5060>;tag=1"))
	assert.Equal(t, common.NetString("PCMU/8000"), displayInfo)
	assert.Equal(t, common.NetString("sip:sipp@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString("1"), tag)
	displayInfo, uri, tag = parseFromTo(common.NetString("\"Matthew Hodgson\" <sip:matthew@mxtelecom.com>;tag=5c7cdb68"))
	assert.Equal(t, common.NetString("Matthew Hodgson"), displayInfo)
	assert.Equal(t, common.NetString("sip:matthew@mxtelecom.com"), uri)
	assert.Equal(t, common.NetString("5c7cdb68"), tag)
	displayInfo, uri, tag = parseFromTo(common.NetString("<sip:matthew@mxtelecom.com>;tag=5c7cdb68"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:matthew@mxtelecom.com"), uri)
	assert.Equal(t, common.NetString("5c7cdb68"), tag)
	displayInfo, uri, tag = parseFromTo(common.NetString("<sip:matthew@mxtelecom.com>"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:matthew@mxtelecom.com"), uri)
	assert.Equal(t, common.NetString(nil), tag)
}
