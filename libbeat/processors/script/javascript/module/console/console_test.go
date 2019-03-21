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
	"go.uber.org/zap"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/script/javascript"

	// Register require module.
	_ "github.com/elastic/beats/libbeat/processors/script/javascript/module/require"
)

func TestConsole(t *testing.T) {
	const script = `
var console = require('console');

function process(evt) {
	console.debug("TestConsole Debug");
	console.log("TestConsole Log/Info");
	console.info("TestConsole Info %j", evt.fields);
	console.warn("TestConsole Warning [%s]", evt.fields.message);
	console.error("TestConsole Error processing event: %j", evt.fields);
}
`

	logp.DevelopmentSetup(logp.ToObserverOutput())
	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Run(&beat.Event{Fields: common.MapStr{"message": "hello world!"}})
	if err != nil {
		t.Fatal(err)
	}

	logs := logp.ObserverLogs().FilterMessageSnippet("TestConsole").TakeAll()
	if assert.Len(t, logs, 5) {
		assert.Contains(t, logs[0].Message, "Debug")
		assert.Equal(t, logs[0].Level, zap.DebugLevel)

		assert.Contains(t, logs[1].Message, "Log/Info")
		assert.Equal(t, logs[1].Level, zap.InfoLevel)

		assert.Contains(t, logs[2].Message, "Info")
		assert.Equal(t, logs[2].Level, zap.InfoLevel)

		assert.Contains(t, logs[3].Message, "Warning")
		assert.Equal(t, logs[3].Level, zap.WarnLevel)

		assert.Contains(t, logs[4].Message, "Error")
		assert.Equal(t, logs[4].Level, zap.ErrorLevel)
	}
}
