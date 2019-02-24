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

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/script/javascript"

	_ "github.com/elastic/beats/libbeat/processors/script/javascript/module/require"
)

func TestConsole(t *testing.T) {
	const script = `
var console = require('console');

function process(evt) {
	console.log("Info %j", evt.fields);
	console.warn("Warning [%s]", evt.fields.message);
	console.error("Error processing event: %j", evt.fields);
}
`

	logp.TestingSetup(logp.ToObserverOutput())
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Run(&beat.Event{Fields: common.MapStr{"message": "hello world!"}})
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, logp.ObserverLogs().FilterMessageSnippet("Info").All(), 1)
	assert.Len(t, logp.ObserverLogs().FilterMessageSnippet("Warning").All(), 1)
	assert.Len(t, logp.ObserverLogs().FilterMessageSnippet("Error").All(), 1)
}
