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

package systemlogs

import (
	"testing"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestJournaldInputIsCreated(t *testing.T) {
	c := map[string]any{
		"files.paths": []string{"/file/does/not/exist"},
		// The 'journald' object needs to exist for the input to be instantiated
		"journald.enabled": true,
	}

	cfg := conf.MustNewConfigFrom(c)

	_, inp, err := configure(cfg)
	if err != nil {
		t.Fatalf("did not expect an error calling newV1Input: %s", err)
	}

	type namer interface {
		Name() string
	}

	i, isNamer := inp.(namer)
	if !isNamer {
		t.Fatalf("expecting an instance of *log.Input, got '%T' instead", inp)
	}

	if got, expected := i.Name(), "journald"; got != expected {
		t.Fatalf("expecting '%s' input, got '%s'", expected, got)
	}
}
