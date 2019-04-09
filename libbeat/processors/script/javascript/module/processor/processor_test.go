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

package processor

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/script/javascript"

	_ "github.com/elastic/beats/libbeat/processors/script/javascript/module/require"
)

func testEvent() *beat.Event {
	return &beat.Event{
		Fields: common.MapStr{
			"source": common.MapStr{
				"ip": "192.0.2.1",
			},
			"destination": common.MapStr{
				"ip": "192.0.2.1",
			},
			"network": common.MapStr{
				"transport": "igmp",
			},
			"message": "key=hello",
		},
	}
}

func TestNewProcessorAddHostMetadata(t *testing.T) {
	const script = `
var processor = require('processor');

var addHostMetadata = new processor.AddHostMetadata({"netinfo.enabled": true});

function process(evt) {
    addHostMetadata.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	_, err = evt.GetValue("host.hostname")
	assert.NoError(t, err)
}

func TestNewProcessorAddLocale(t *testing.T) {
	const script = `
var processor = require('processor');

var addLocale = new processor.AddLocale();

function process(evt) {
    addLocale.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	_, err = evt.GetValue("event.timezone")
	assert.NoError(t, err)
}

func TestNewProcessorAddProcessMetadata(t *testing.T) {
	const script = `
var processor = require('processor');

var addProcessMetadata = new processor.AddProcessMetadata({
    match_pids: "process.pid",
    overwrite_keys: true,
});

function process(evt) {
    addProcessMetadata.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{Fields: common.MapStr{"process": common.MapStr{"pid": os.Getppid()}}}
	evt, err = p.Run(evt)
	if err != nil {
		t.Fatal(err)
	}

	_, err = evt.GetValue("process.name")
	assert.NoError(t, err)
	t.Logf("%+v", evt.Fields)
}

func TestNewProcessorCommunityID(t *testing.T) {
	const script = `
var processor = require('processor');

var communityID = new processor.CommunityID();

function process(evt) {
    communityID.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	id, _ := evt.GetValue("network.community_id")
	assert.Equal(t, "1:15+Ly6HsDg0sJdTmNktf6rko+os=", id)
}

func TestNewCopyFields(t *testing.T) {
	const script = `
var processor = require('processor');

var copy = new processor.CopyFields({
    fields: [
        {from: "message", to: "log.original"},
    ],
});

function process(evt) {
	copy.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	_, err = evt.GetValue("log.original")
	assert.NoError(t, err)
}

func TestNewProcessorDecodeJSONFields(t *testing.T) {
	const script = `
var processor = require('processor');

var decodeJSON = new processor.DecodeJSONFields({
    fields: ["message"],
    target: "",
});

function process(evt) {
	decodeJSON.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt := testEvent()
	evt.PutValue("message", `{"hello": "world"}`)

	_, err = p.Run(evt)
	if err != nil {
		t.Fatal(err)
	}

	v, _ := evt.GetValue("hello")
	assert.Equal(t, "world", v)
}

func TestNewProcessorDissect(t *testing.T) {
	const script = `
var processor = require('processor');

var chopLog = new processor.Dissect({
    tokenizer: "key=%{key}",
    field: "message",
});

function process(evt) {
    chopLog.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	key, _ := evt.GetValue("dissect.key")
	assert.Equal(t, "hello", key)
}

func TestNewProcessorDNS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows requires explicit DNS server configuration")
	}

	const script = `
var processor = require('processor');

var dns = new processor.DNS({
    type: "reverse",
    fields: {
        "source.ip": "source.domain",
        "destination.ip": "destination.domain"
    },
    tag_on_failure: ["_dns_reverse_lookup_failed"],
});

function process(evt) {
	dns.Run(evt);
    if (evt.Get().tags[0] !== "_dns_reverse_lookup_failed") {
        throw "missing tag";
    }
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewRename(t *testing.T) {
	const script = `
var processor = require('processor');

var rename = new processor.Rename({
    fields: [
        {from: "message", to: "log.original"},
    ],
});

function process(evt) {
	rename.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	_, err = evt.GetValue("log.original")
	assert.NoError(t, err)
}

func TestNewTruncateFields(t *testing.T) {
	const script = `
var processor = require('processor');

var truncate = new processor.TruncateFields({
    fields: [
        "message",
    ],
    max_characters: 4,
});

function process(evt) {
	truncate.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	msg, _ := evt.GetValue("message")
	assert.Equal(t, "key=", msg)
}

func TestNewProcessorChain(t *testing.T) {
	const script = `
var processor = require('processor');

var localeProcessor = new processor.AddLocale();

var chain = new processor.Chain()
    .Add(localeProcessor)
    .Rename({
        fields: [
            {from: "event.timezone", to: "timezone"},
        ],
    })
    .Add(function(evt) {
		evt.Put("hello", "world");
    })
    .Build();

function process(evt) {
	chain.Run(evt);
}
`

	logp.TestingSetup()
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	if err != nil {
		t.Fatal(err)
	}

	_, err = evt.GetValue("timezone")
	assert.NoError(t, err)
	v, _ := evt.GetValue("hello")
	assert.Equal(t, "world", v)
}
