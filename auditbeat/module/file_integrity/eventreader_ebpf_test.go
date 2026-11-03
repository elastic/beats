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

//go:build linux && (amd64 || arm64)

package file_integrity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/ebpf"
	"github.com/elastic/ebpfevents"
)

func TestMissingEvents(t *testing.T) {
	fileEvents := ebpf.EventMask(ebpfevents.EventTypeFileCreate | ebpfevents.EventTypeFileRename |
		ebpfevents.EventTypeFileDelete | ebpfevents.EventTypeFileModify)

	// What the loader disables today: the events of the tp_btf programs, which
	// file_integrity doesn't subscribe to.
	assert.Empty(t, missingEvents(fileEvents, nil))
	assert.Empty(t, missingEvents(fileEvents, []ebpfevents.EventType{
		ebpfevents.EventTypeProcessExec, ebpfevents.EventTypeProcessFork,
	}))

	assert.Equal(t, []string{"FileModify"}, missingEvents(fileEvents, []ebpfevents.EventType{
		ebpfevents.EventTypeProcessExec, ebpfevents.EventTypeFileModify,
	}))
}
