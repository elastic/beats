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

//go:build linux

package journald

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestInputParsers ensures journald input support parsers,
// it only tests a single parser, but that is enough to ensure
// we're correctly using the parsers
func TestInputParsers(t *testing.T) {
	// If this test fails, uncomment the lopg setup line
	// to send logs to stderr
	// logp.DevelopmentSetup()
	out := decompress(t, filepath.Join("testdata", "ndjson-parser.journal.gz"))

	env := newInputTestingEnvironment(t)
	inp := env.mustCreateInput(mapstr.M{
		"paths": []string{out},
		"parsers": []mapstr.M{
			{
				"ndjson": mapstr.M{
					"target": "",
				},
			},
		},
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)
	env.startInput(ctx, inp)
	env.waitUntilEventCount(1)
	event := env.pipeline.clients[0].GetEvents()[0]

	foo, isString := event.Fields["foo"].(string)
	if !isString {
		t.Errorf("expecting field 'foo' to be string, got %T", event.Fields["foo"])
	}

	answer, isInt := event.Fields["answer"].(int64)
	if !isInt {
		t.Errorf("expecting field 'answer' to be int64, got %T", event.Fields["answer"])
	}

	// The JSON in the test journal is: '{"foo": "bar", "answer":42}'
	expectedFoo := "bar"
	expectedAnswer := int64(42)
	if foo != expectedFoo {
		t.Errorf("expecting 'foo' from the Journal JSON to be '%s' got '%s' instead", expectedFoo, foo)
	}
	if answer != expectedAnswer {
		t.Errorf("expecting 'answer' from the Journal JSON to be '%d' got '%d' instead", expectedAnswer, answer)
	}
}

func TestPartialMessageTag(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "ndjson-parser.journal.gz"))
	env := newInputTestingEnvironment(t)
	inp := env.mustCreateInput(mapstr.M{
		"paths": []string{out},
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)
	env.startInput(ctx, inp)
	env.waitUntilEventCount(1)
	event := env.pipeline.clients[0].GetEvents()[0]

	tags, err := event.Fields.GetValue("tags")
	if err != nil {
		t.Fatalf("'tags' not found in event: %s", err)
	}

	tagsStrSlice, ok := tags.([]string)
	if !ok {
		t.Fatalf("expecting 'tags' to be []string, got %T instead", tags)
	}
	if tagsStrSlice[0] != "partial_message" {
		t.Fatalf("expecting the tag 'partial_message', got %v instead", tagsStrSlice)
	}
}
