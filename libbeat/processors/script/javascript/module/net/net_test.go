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

package net_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors/script/javascript"

	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/net"
	_ "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/require"
)

func TestNetIsIP(t *testing.T) {
	const script = `
var net = require('net');

function process(evt) {
    var ip = evt.Get("ip");
    var ipType = net.isIP(ip);
	switch (ipType) {
    case 4:
        evt.Put("network.type", "ipv4");
        break
    case 6:
        evt.Put("network.type", "ipv6");
        break
    }
}
`

	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	for ip, typ := range map[string]interface{}{
		"192.168.0.1":        "ipv4",
		"::ffff:192.168.0.1": "ipv4",
		"2001:0db8:0000:0000:0000:ff00:0042:8329": "ipv6",
		"2001:db8:0:0:0:ff00:42:8329":             "ipv6",
		"2001:db8::ff00:42:8329":                  "ipv6",
		"www.elastic.co":                          nil,
	} {
		evt, err := p.Run(&beat.Event{Fields: mapstr.M{"ip": ip}})
		if err != nil {
			t.Fatal(err)
		}

		fields := evt.Fields.Flatten()
		assert.Equal(t, typ, fields["network.type"])
	}
}

func TestNetIsIPvN(t *testing.T) {
	const script = `
var net = require('net');

function process(evt) {
   	if (net.isIPv4("192.168.0.1") !== true) {
        throw "isIPv4 failed";
    }

   	if (net.isIPv6("2001:db8::ff00:42:8329") !== true) {
        throw "isIPv6 failed";
    }
}
`

	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Run(&beat.Event{Fields: mapstr.M{}})
	if err != nil {
		t.Fatal(err)
	}
}
