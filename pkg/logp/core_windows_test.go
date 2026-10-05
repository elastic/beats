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

package logp

import (
	"io"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestEventLogOutputCanBeClosed(t *testing.T) {
	cfg := DefaultConfig(DefaultEnvironment)
	cfg.ToFiles = true
	cfg.Beat = t.Name()

	eventLog, err := makeEventLogOutput(cfg, zapcore.DebugLevel)
	if err != nil {
		t.Fatalf("cannot create eventLog output: %s", err)
	}

	closer, ok := eventLog.(io.Closer)
	if !ok {
		t.Fatal("the EventLog Output does not implement io.Closer")
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("Close must not return any error, got: %s", err)
	}
}
