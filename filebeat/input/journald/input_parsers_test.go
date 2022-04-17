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

//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package journald

import (
	"context"
	"path"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// TestInputParsers ensures journald input support parsers,
// it only tests a single parser, but that is enough to ensure
// we're correctly using the parsers
func TestInputParsers(t *testing.T) {
	inputParsersExpected := []string{"1st line\n2nd line\n3rd line", "4th line\n5th line\n6th line"}
	env := newInputTestingEnvironment(t)

	inp := env.mustCreateInput(common.MapStr{
		"paths":                 []string{path.Join("testdata", "input-multiline-parser.journal")},
		"include_matches.match": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
		"parsers": []common.MapStr{
			{
				"multiline": common.MapStr{
					"type":        "count",
					"count_lines": 3,
				},
			},
		},
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)
	env.waitUntilEventCount(len(inputParsersExpected))

	for idx, event := range env.pipeline.clients[0].GetEvents() {
		if got, expected := event.Fields["message"], inputParsersExpected[idx]; got != expected {
			t.Errorf("expecting event message %q, got %q", expected, got)
		}
	}

	cancelInput()
}
